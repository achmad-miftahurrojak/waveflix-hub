const Redis = require('ioredis');
const { LRUCache } = require('lru-cache');
const NodeCache = require('node-cache');

class CacheManager {
  constructor(options = {}) {
    this.redisUrl = options.redisUrl;
    this.logger = options.logger;
    this.nodeId = options.nodeId || process.env.NODE_ID || 'cache-node-1';
    this.region = options.region || process.env.REGION || 'us-east-1';
    
    // Cache tier configurations
    this.tiers = {
      L1: {
        name: 'L1_EDGE',
        capacity: 1 * 1024 * 1024 * 1024 * 1024, // 1TB in bytes
        ttlDefault: 3600, // 1 hour
        priority: 1
      },
      L2: {
        name: 'L2_REGIONAL',
        capacity: 10 * 1024 * 1024 * 1024 * 1024, // 10TB in bytes
        ttlDefault: 86400, // 24 hours
        priority: 2
      },
      L3: {
        name: 'L3_ORIGIN',
        capacity: Number.MAX_SAFE_INTEGER, // Unlimited
        ttlDefault: 604800, // 7 days
        priority: 3
      }
    };

    // Initialize cache instances
    this.l1Cache = null; // In-memory LRU
    this.l2Cache = null; // Redis cluster
    this.l3Cache = null; // Redis persistent
    
    this.hitStats = new NodeCache({ stdTTL: 3600 });
  }

  async initialize() {
    try {
      // L1 Cache - In-memory LRU cache for edge nodes
      this.l1Cache = new LRUCache({
        maxSize: this.tiers.L1.capacity,
        sizeCalculation: (value) => {
          return Buffer.isBuffer(value) ? value.length : JSON.stringify(value).length;
        },
        ttl: this.tiers.L1.ttlDefault * 1000, // Convert to milliseconds
        allowStale: true,
        updateAgeOnGet: true
      });

      // L2 Cache - Redis cluster for regional caching
      this.l2Cache = new Redis.Cluster([
        {
          host: process.env.REDIS_L2_HOST || 'localhost',
          port: process.env.REDIS_L2_PORT || 6379
        }
      ], {
        redisOptions: {
          password: process.env.REDIS_PASSWORD
        },
        maxRetriesPerRequest: 3,
        retryDelayOnFailover: 100,
        keyPrefix: `l2:${this.region}:`
      });

      // L3 Cache - Redis persistent for origin caching
      this.l3Cache = new Redis({
        host: process.env.REDIS_L3_HOST || 'localhost',
        port: process.env.REDIS_L3_PORT || 6380,
        password: process.env.REDIS_PASSWORD,
        keyPrefix: 'l3:origin:',
        retryDelayOnFailover: 100,
        maxRetriesPerRequest: 3,
        db: 0
      });

      // Set up event listeners
      this.setupEventListeners();
      
      this.logger?.info('Cache manager initialized successfully', {
        nodeId: this.nodeId,
        region: this.region,
        tiers: Object.keys(this.tiers)
      });

    } catch (error) {
      this.logger?.error('Failed to initialize cache manager:', error);
      throw error;
    }
  }

  setupEventListeners() {
    // L2 Cache events
    this.l2Cache.on('connect', () => {
      this.logger?.info('L2 cache connected');
    });

    this.l2Cache.on('error', (error) => {
      this.logger?.error('L2 cache error:', error);
    });

    // L3 Cache events
    this.l3Cache.on('connect', () => {
      this.logger?.info('L3 cache connected');
    });

    this.l3Cache.on('error', (error) => {
      this.logger?.error('L3 cache error:', error);
    });
  }

  async get(key) {
    const startTime = Date.now();
    let result = null;
    let tier = null;

    try {
      // Try L1 cache first (edge node cache)
      result = this.l1Cache.get(key);
      if (result) {
        tier = 'L1';
        this.recordCacheHit(key, tier, Date.now() - startTime);
        return { data: result, tier, cached_at: Date.now() };
      }

      // Try L2 cache (regional cache)
      const l2Result = await this.l2Cache.get(key);
      if (l2Result) {
        const parsed = JSON.parse(l2Result);
        tier = 'L2';
        
        // Promote to L1 cache
        this.l1Cache.set(key, parsed.data, { 
          ttl: this.calculateTTL(key, 'L1') * 1000 
        });
        
        this.recordCacheHit(key, tier, Date.now() - startTime);
        return { data: parsed.data, tier, cached_at: parsed.cached_at };
      }

      // Try L3 cache (origin cache)
      const l3Result = await this.l3Cache.get(key);
      if (l3Result) {
        const parsed = JSON.parse(l3Result);
        tier = 'L3';
        
        // Promote to L2 and L1 caches
        const cacheData = { 
          data: parsed.data, 
          cached_at: parsed.cached_at 
        };
        
        await this.l2Cache.setex(key, this.calculateTTL(key, 'L2'), JSON.stringify(cacheData));
        this.l1Cache.set(key, parsed.data, { 
          ttl: this.calculateTTL(key, 'L1') * 1000 
        });
        
        this.recordCacheHit(key, tier, Date.now() - startTime);
        return { data: parsed.data, tier, cached_at: parsed.cached_at };
      }

      // Cache miss - record statistics
      this.recordCacheMiss(key, Date.now() - startTime);
      return null;

    } catch (error) {
      this.logger?.error('Cache get error:', { key, error: error.message });
      throw error;
    }
  }

  async set(key, data, options = {}) {
    try {
      const { ttl, priority = 5, contentId, quality, segment } = options;
      const cacheData = {
        data,
        cached_at: Date.now(),
        metadata: {
          contentId,
          quality,
          segment,
          size: Buffer.isBuffer(data) ? data.length : JSON.stringify(data).length,
          priority
        }
      };

      // Determine cache tier based on priority and content type
      const targetTier = this.determineCacheTier(options);
      
      // Set in appropriate tier and propagate up
      switch (targetTier) {
        case 'L3':
          await this.l3Cache.setex(
            key, 
            ttl || this.tiers.L3.ttlDefault, 
            JSON.stringify(cacheData)
          );
          // Fall through to set in L2 and L1
        
        case 'L2':
          await this.l2Cache.setex(
            key, 
            ttl || this.tiers.L2.ttlDefault, 
            JSON.stringify(cacheData)
          );
          // Fall through to set in L1
        
        case 'L1':
          this.l1Cache.set(key, data, { 
            ttl: (ttl || this.tiers.L1.ttlDefault) * 1000 
          });
          break;
      }

      // Update popularity metrics for content prediction
      if (contentId) {
        await this.updatePopularityMetrics(contentId, quality, segment);
      }

      this.logger?.info('Cache set successful', { 
        key, 
        tier: targetTier, 
        size: cacheData.metadata.size,
        ttl: ttl || this.tiers[targetTier].ttlDefault
      });

    } catch (error) {
      this.logger?.error('Cache set error:', { key, error: error.message });
      throw error;
    }
  }

  async invalidate(contentId) {
    try {
      // Build pattern to match all related cache keys
      const patterns = [
        `content:meta:${contentId}`,
        `content:playlist:${contentId}:*`,
        `content:segment:${contentId}:*`
      ];

      const invalidationPromises = [];

      for (const pattern of patterns) {
        // L1 cache - manual iteration (LRU cache doesn't support pattern matching)
        for (const [key] of this.l1Cache.entrySet()) {
          if (this.matchesPattern(key, pattern)) {
            this.l1Cache.delete(key);
          }
        }

        // L2 cache - use Redis SCAN
        invalidationPromises.push(this.invalidateRedisPattern(this.l2Cache, pattern));
        
        // L3 cache - use Redis SCAN  
        invalidationPromises.push(this.invalidateRedisPattern(this.l3Cache, pattern));
      }

      await Promise.all(invalidationPromises);
      
      this.logger?.info('Cache invalidation completed', { contentId });

    } catch (error) {
      this.logger?.error('Cache invalidation error:', { contentId, error: error.message });
      throw error;
    }
  }

  async invalidateRedisPattern(redisClient, pattern) {
    const stream = redisClient.scanStream({
      match: pattern,
      count: 100
    });

    return new Promise((resolve, reject) => {
      const keys = [];
      
      stream.on('data', (resultKeys) => {
        keys.push(...resultKeys);
      });

      stream.on('end', async () => {
        if (keys.length > 0) {
          try {
            await redisClient.del(...keys);
            this.logger?.debug('Invalidated keys:', { pattern, count: keys.length });
          } catch (error) {
            this.logger?.error('Failed to delete keys:', { pattern, error: error.message });
          }
        }
        resolve();
      });

      stream.on('error', reject);
    });
  }

  matchesPattern(key, pattern) {
    const regex = new RegExp(pattern.replace(/\*/g, '.*'));
    return regex.test(key);
  }

  determineCacheTier(options) {
    const { priority = 5, segment, contentId } = options;
    
    // High priority content goes to all tiers
    if (priority <= 2) return 'L3';
    
    // Video segments prioritized for edge caching
    if (segment) return 'L2';
    
    // Default to L1 for metadata and playlists
    return 'L1';
  }

  calculateTTL(key, tier) {
    // Dynamic TTL based on content type and tier
    if (key.includes('segment:')) {
      // Video segments cached longer
      return {
        'L1': 3600,  // 1 hour
        'L2': 86400, // 24 hours  
        'L3': 604800 // 7 days
      }[tier];
    }
    
    if (key.includes('playlist:')) {
      // Playlists cached shorter (may change frequently)
      return {
        'L1': 300,   // 5 minutes
        'L2': 1800,  // 30 minutes
        'L3': 3600   // 1 hour
      }[tier];
    }
    
    // Default metadata TTL
    return this.tiers[tier].ttlDefault;
  }

  async updatePopularityMetrics(contentId, quality, segment) {
    try {
      const timestamp = Math.floor(Date.now() / 3600000); // Hour bucket
      const key = `popular:hourly:${timestamp}`;
      
      await this.l3Cache.zincrby(key, 1, `${contentId}:${quality || 'all'}`);
      await this.l3Cache.expire(key, 604800); // 7 days
      
    } catch (error) {
      this.logger?.error('Failed to update popularity metrics:', error);
    }
  }

  recordCacheHit(key, tier, responseTime) {
    const stats = this.hitStats.get('cache_stats') || {
      hits: 0,
      misses: 0,
      hit_ratio: 0,
      avg_response_time: 0,
      tier_stats: { L1: 0, L2: 0, L3: 0 }
    };

    stats.hits++;
    stats.tier_stats[tier]++;
    stats.hit_ratio = stats.hits / (stats.hits + stats.misses);
    stats.avg_response_time = (stats.avg_response_time + responseTime) / 2;

    this.hitStats.set('cache_stats', stats);
  }

  recordCacheMiss(key, responseTime) {
    const stats = this.hitStats.get('cache_stats') || {
      hits: 0,
      misses: 0,
      hit_ratio: 0,
      avg_response_time: 0,
      tier_stats: { L1: 0, L2: 0, L3: 0 }
    };

    stats.misses++;
    stats.hit_ratio = stats.hits / (stats.hits + stats.misses);
    stats.avg_response_time = (stats.avg_response_time + responseTime) / 2;

    this.hitStats.set('cache_stats', stats);
  }

  async getStats() {
    const stats = this.hitStats.get('cache_stats') || {
      hits: 0,
      misses: 0,
      hit_ratio: 0,
      avg_response_time: 0,
      tier_stats: { L1: 0, L2: 0, L3: 0 }
    };

    return {
      ...stats,
      cache_info: {
        l1_size: this.l1Cache.size,
        l1_max: this.l1Cache.max,
        node_id: this.nodeId,
        region: this.region
      },
      timestamp: Date.now()
    };
  }

  async disconnect() {
    try {
      if (this.l2Cache) {
        await this.l2Cache.disconnect();
      }
      if (this.l3Cache) {
        await this.l3Cache.disconnect();  
      }
      this.logger?.info('Cache manager disconnected');
    } catch (error) {
      this.logger?.error('Error during cache disconnect:', error);
    }
  }
}

module.exports = CacheManager;