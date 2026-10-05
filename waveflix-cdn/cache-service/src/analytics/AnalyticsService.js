const Redis = require('ioredis');

class AnalyticsService {
  constructor(options = {}) {
    this.cacheManager = options.cacheManager;
    this.logger = options.logger;
    this.nodeId = options.nodeId || process.env.NODE_ID || 'cache-node-1';
    this.region = options.region || process.env.REGION || 'us-east-1';
    
    this.redis = null;
    this.metricsAggregationInterval = null;
    
    // Analytics configuration
    this.config = {
      metricsRetention: 7 * 24 * 3600, // 7 days
      aggregationInterval: 60, // 1 minute
      popularContentThreshold: 100, // requests per hour
      cacheOptimizationThreshold: 0.85 // 85% hit ratio
    };

    // Metric buckets for time-series data
    this.metricBuckets = {
      minute: 60,
      hour: 3600,
      day: 86400
    };
  }

  async initialize() {
    try {
      this.redis = new Redis({
        host: process.env.ANALYTICS_REDIS_HOST || 'localhost',
        port: process.env.ANALYTICS_REDIS_PORT || 6381,
        password: process.env.REDIS_PASSWORD,
        keyPrefix: 'analytics:',
        retryDelayOnFailover: 100,
        maxRetriesPerRequest: 3
      });

      // Start metrics aggregation
      await this.startMetricsAggregation();
      
      this.logger?.info('Analytics service initialized', {
        nodeId: this.nodeId,
        region: this.region
      });

    } catch (error) {
      this.logger?.error('Failed to initialize analytics service:', error);
      throw error;
    }
  }

  async startMetricsAggregation() {
    this.metricsAggregationInterval = setInterval(async () => {
      try {
        await this.aggregateMetrics();
        await this.detectAnomalies();
        await this.updatePopularContent();
      } catch (error) {
        this.logger?.error('Metrics aggregation failed:', error);
      }
    }, this.config.aggregationInterval * 1000);
  }

  recordHit(contentId, quality, segment) {
    try {
      const timestamp = Math.floor(Date.now() / 1000);
      const keys = this.generateMetricKeys('hit', timestamp);
      
      const pipeline = this.redis.pipeline();
      
      // Record hit metrics
      for (const [bucket, key] of Object.entries(keys)) {
        pipeline.incr(key);
        pipeline.expire(key, this.config.metricsRetention);
      }

      // Record content-specific metrics
      const contentKey = `content_hits:${contentId}:${timestamp}`;
      pipeline.incr(contentKey);
      pipeline.expire(contentKey, this.config.metricsRetention);

      // Record quality-specific metrics
      if (quality) {
        const qualityKey = `quality_hits:${quality}:${timestamp}`;
        pipeline.incr(qualityKey);
        pipeline.expire(qualityKey, this.config.metricsRetention);
      }

      // Record segment metrics
      if (segment) {
        const segmentKey = `segment_hits:${contentId}:${segment}:${timestamp}`;
        pipeline.incr(segmentKey);
        pipeline.expire(segmentKey, this.config.metricsRetention);
      }

      pipeline.exec();

    } catch (error) {
      this.logger?.error('Failed to record cache hit:', error);
    }
  }

  recordMiss(contentId, quality, segment) {
    try {
      const timestamp = Math.floor(Date.now() / 1000);
      const keys = this.generateMetricKeys('miss', timestamp);
      
      const pipeline = this.redis.pipeline();
      
      // Record miss metrics
      for (const [bucket, key] of Object.entries(keys)) {
        pipeline.incr(key);
        pipeline.expire(key, this.config.metricsRetention);
      }

      // Record content-specific misses
      const contentKey = `content_misses:${contentId}:${timestamp}`;
      pipeline.incr(contentKey);
      pipeline.expire(contentKey, this.config.metricsRetention);

      pipeline.exec();

    } catch (error) {
      this.logger?.error('Failed to record cache miss:', error);
    }
  }

  generateMetricKeys(type, timestamp) {
    const keys = {};
    
    for (const [bucket, duration] of Object.entries(this.metricBuckets)) {
      const bucketTimestamp = Math.floor(timestamp / duration) * duration;
      keys[bucket] = `${type}:${this.nodeId}:${this.region}:${bucket}:${bucketTimestamp}`;
    }
    
    return keys;
  }

  async aggregateMetrics() {
    try {
      const currentTime = Math.floor(Date.now() / 1000);
      const stats = await this.cacheManager.getStats();
      
      // Aggregate current cache statistics
      const aggregatedStats = {
        timestamp: currentTime,
        node_id: this.nodeId,
        region: this.region,
        cache_stats: {
          hit_ratio: stats.hit_ratio || 0,
          total_hits: stats.hits || 0,
          total_misses: stats.misses || 0,
          l1_utilization: stats.cache_info.l1_size / stats.cache_info.l1_max,
          avg_response_time: stats.avg_response_time || 0
        },
        tier_distribution: stats.tier_stats || { L1: 0, L2: 0, L3: 0 }
      };

      // Store aggregated statistics
      const key = `aggregated_stats:${this.nodeId}:${currentTime}`;
      await this.redis.setex(key, this.config.metricsRetention, JSON.stringify(aggregatedStats));

      // Update regional statistics
      await this.updateRegionalStats(aggregatedStats);

    } catch (error) {
      this.logger?.error('Failed to aggregate metrics:', error);
    }
  }

  async updateRegionalStats(nodeStats) {
    try {
      const regionKey = `regional_stats:${this.region}`;
      const regionalData = await this.redis.hgetall(regionKey);
      
      const currentStats = regionalData ? JSON.parse(regionalData.stats || '{}') : {
        total_nodes: 0,
        avg_hit_ratio: 0,
        total_hits: 0,
        total_misses: 0,
        nodes: {}
      };

      // Update node data
      currentStats.nodes[this.nodeId] = nodeStats.cache_stats;
      currentStats.total_nodes = Object.keys(currentStats.nodes).length;

      // Calculate regional averages
      const nodeValues = Object.values(currentStats.nodes);
      currentStats.avg_hit_ratio = nodeValues.reduce((sum, node) => sum + node.hit_ratio, 0) / nodeValues.length;
      currentStats.total_hits = nodeValues.reduce((sum, node) => sum + node.total_hits, 0);
      currentStats.total_misses = nodeValues.reduce((sum, node) => sum + node.total_misses, 0);

      await this.redis.hset(regionKey, {
        stats: JSON.stringify(currentStats),
        updated_at: Date.now()
      });
      await this.redis.expire(regionKey, this.config.metricsRetention);

    } catch (error) {
      this.logger?.error('Failed to update regional stats:', error);
    }
  }

  async detectAnomalies() {
    try {
      const stats = await this.cacheManager.getStats();
      const alerts = [];

      // Check hit ratio
      if (stats.hit_ratio < this.config.cacheOptimizationThreshold) {
        alerts.push({
          type: 'LOW_HIT_RATIO',
          severity: 'WARNING',
          message: `Cache hit ratio ${(stats.hit_ratio * 100).toFixed(2)}% below threshold`,
          threshold: this.config.cacheOptimizationThreshold,
          current: stats.hit_ratio,
          timestamp: Date.now()
        });
      }

      // Check L1 cache utilization
      const l1Utilization = stats.cache_info.l1_size / stats.cache_info.l1_max;
      if (l1Utilization > 0.9) {
        alerts.push({
          type: 'HIGH_CACHE_UTILIZATION',
          severity: 'WARNING',
          message: `L1 cache utilization ${(l1Utilization * 100).toFixed(2)}% is high`,
          threshold: 0.9,
          current: l1Utilization,
          timestamp: Date.now()
        });
      }

      // Check response time
      if (stats.avg_response_time > 1000) { // 1 second
        alerts.push({
          type: 'HIGH_RESPONSE_TIME',
          severity: 'CRITICAL',
          message: `Average response time ${stats.avg_response_time}ms is high`,
          threshold: 1000,
          current: stats.avg_response_time,
          timestamp: Date.now()
        });
      }

      // Store and log alerts
      if (alerts.length > 0) {
        const alertKey = `alerts:${this.nodeId}:${Date.now()}`;
        await this.redis.setex(alertKey, 3600, JSON.stringify(alerts)); // 1 hour retention

        for (const alert of alerts) {
          this.logger?.warn('Anomaly detected', alert);
        }
      }

    } catch (error) {
      this.logger?.error('Failed to detect anomalies:', error);
    }
  }

  async updatePopularContent() {
    try {
      const currentHour = Math.floor(Date.now() / 3600000);
      const previousHour = currentHour - 1;
      
      // Get content access patterns from previous hour
      const pattern = `content_hits:*:${previousHour * 3600}`;
      const keys = await this.scanKeys(pattern);
      
      const contentPopularity = {};
      
      for (const key of keys) {
        const hits = await this.redis.get(key);
        const contentId = key.split(':')[1];
        
        contentPopularity[contentId] = (contentPopularity[contentId] || 0) + parseInt(hits || 0);
      }

      // Update popular content rankings
      const popularContentKey = `popular_content:${currentHour}`;
      const pipeline = this.redis.pipeline();
      
      for (const [contentId, hits] of Object.entries(contentPopularity)) {
        if (hits >= this.config.popularContentThreshold) {
          pipeline.zadd(popularContentKey, hits, contentId);
        }
      }
      
      pipeline.expire(popularContentKey, this.config.metricsRetention);
      await pipeline.exec();

      // Trigger cache preloading for popular content
      const topContent = await this.redis.zrevrange(popularContentKey, 0, 9); // Top 10
      await this.triggerCachePreloading(topContent);

    } catch (error) {
      this.logger?.error('Failed to update popular content:', error);
    }
  }

  async triggerCachePreloading(popularContent) {
    try {
      for (const contentId of popularContent) {
        // Check if content is already cached
        const metaKey = `content:meta:${contentId}`;
        const cached = await this.cacheManager.get(metaKey);
        
        if (!cached) {
          // Trigger preload (this would typically send a message to content service)
          this.logger?.info('Triggering preload for popular content', { contentId });
          
          const preloadKey = `preload_queue:${contentId}`;
          await this.redis.lpush('preload_requests', JSON.stringify({
            contentId,
            priority: 'high',
            requested_by: this.nodeId,
            timestamp: Date.now()
          }));
        }
      }
    } catch (error) {
      this.logger?.error('Failed to trigger cache preloading:', error);
    }
  }

  async scanKeys(pattern) {
    const keys = [];
    const stream = this.redis.scanStream({
      match: pattern,
      count: 100
    });

    return new Promise((resolve, reject) => {
      stream.on('data', (resultKeys) => {
        keys.push(...resultKeys);
      });

      stream.on('end', () => {
        resolve(keys);
      });

      stream.on('error', reject);
    });
  }

  async getStats() {
    try {
      const currentTime = Math.floor(Date.now() / 1000);
      const hourAgo = currentTime - 3600;
      
      // Get cache statistics
      const cacheStats = await this.cacheManager.getStats();
      
      // Get regional statistics
      const regionalKey = `regional_stats:${this.region}`;
      const regionalData = await this.redis.hgetall(regionalKey);
      const regionalStats = regionalData ? JSON.parse(regionalData.stats || '{}') : null;
      
      // Get popular content
      const popularKey = `popular_content:${Math.floor(Date.now() / 3600000)}`;
      const popularContent = await this.redis.zrevrange(popularKey, 0, 9, 'WITHSCORES');
      
      // Get recent alerts
      const alertKeys = await this.scanKeys(`alerts:${this.nodeId}:*`);
      const recentAlerts = [];
      
      for (const key of alertKeys.slice(-5)) { // Last 5 alerts
        const alertData = await this.redis.get(key);
        if (alertData) {
          recentAlerts.push(JSON.parse(alertData));
        }
      }

      return {
        node: {
          nodeId: this.nodeId,
          region: this.region,
          cache_stats: cacheStats,
          timestamp: Date.now()
        },
        regional: regionalStats,
        popular_content: this.formatPopularContent(popularContent),
        alerts: recentAlerts.flat(),
        performance: {
          cache_optimization_needed: cacheStats.hit_ratio < this.config.cacheOptimizationThreshold,
          preload_candidates: popularContent.length / 2 // Half of popular content
        }
      };

    } catch (error) {
      this.logger?.error('Failed to get analytics stats:', error);
      throw error;
    }
  }

  formatPopularContent(redisData) {
    const formatted = [];
    for (let i = 0; i < redisData.length; i += 2) {
      formatted.push({
        contentId: redisData[i],
        hits: parseInt(redisData[i + 1])
      });
    }
    return formatted;
  }

  async disconnect() {
    try {
      if (this.metricsAggregationInterval) {
        clearInterval(this.metricsAggregationInterval);
      }
      
      if (this.redis) {
        await this.redis.disconnect();
      }

      this.logger?.info('Analytics service disconnected');
    } catch (error) {
      this.logger?.error('Error during analytics service disconnect:', error);
    }
  }
}

module.exports = AnalyticsService;