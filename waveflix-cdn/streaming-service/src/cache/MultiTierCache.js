const Redis = require('ioredis');
const NodeCache = require('node-cache');
const fs = require('fs').promises;
const path = require('path');
const crypto = require('crypto');

class MultiTierCache {
    constructor(config) {
        this.config = config;
        
        // L1 Cache: Memory (Edge Node - 1TB capacity)
        this.l1Cache = new NodeCache({
            stdTTL: config.l1.ttl,
            checkperiod: 120,
            useClones: false,
            maxKeys: config.l1.maxKeys
        });

        // L2 Cache: Redis (Regional Cache - 10TB capacity)
        this.l2Cache = null;
        if (config.l2.enabled) {
            this.l2Cache = new Redis({
                host: config.l2.host,
                port: config.l2.port,
                password: config.l2.password,
                db: config.l2.db,
                maxRetriesPerRequest: 3,
                retryDelayOnFailover: 100
            });
        }

        // L3 Cache: File System (Origin Cache - Unlimited)
        this.l3CachePath = config.l3.path;
        
        this.metrics = {
            l1: { hits: 0, misses: 0, evictions: 0 },
            l2: { hits: 0, misses: 0, evictions: 0 },
            l3: { hits: 0, misses: 0, evictions: 0 },
            totalRequests: 0,
            totalHits: 0,
            bytesServed: 0
        };

        this.initializeL3Cache();
    }

    async initializeL3Cache() {
        try {
            await fs.mkdir(this.l3CachePath, { recursive: true });
        } catch (error) {
            console.error('Failed to initialize L3 cache directory:', error);
        }
    }

    generateCacheKey(contentId, quality, segment) {
        const keyParts = [contentId, quality, segment].filter(Boolean);
        return keyParts.join(':');
    }

    generateL3Path(key) {
        const hash = crypto.createHash('md5').update(key).digest('hex');
        const dir1 = hash.substring(0, 2);
        const dir2 = hash.substring(2, 4);
        return path.join(this.l3CachePath, dir1, dir2, `${hash}.cache`);
    }

    async get(key, options = {}) {
        this.metrics.totalRequests++;
        
        try {
            // Try L1 Cache first (Memory)
            const l1Result = this.l1Cache.get(key);
            if (l1Result !== undefined) {
                this.metrics.l1.hits++;
                this.metrics.totalHits++;
                
                // Promote to front of L1 cache
                this.l1Cache.ttl(key, this.config.l1.ttl);
                
                return {
                    data: l1Result,
                    source: 'L1',
                    hit: true
                };
            }
            this.metrics.l1.misses++;

            // Try L2 Cache (Redis)
            if (this.l2Cache) {
                const l2Result = await this.l2Cache.getBuffer(key);
                if (l2Result) {
                    this.metrics.l2.hits++;
                    this.metrics.totalHits++;
                    
                    // Promote to L1 cache
                    this.l1Cache.set(key, l2Result, this.config.l1.ttl);
                    
                    return {
                        data: l2Result,
                        source: 'L2',
                        hit: true
                    };
                }
                this.metrics.l2.misses++;
            }

            // Try L3 Cache (File System)
            const l3Path = this.generateL3Path(key);
            try {
                const l3Result = await fs.readFile(l3Path);
                this.metrics.l3.hits++;
                this.metrics.totalHits++;
                
                // Promote to L2 and L1
                if (this.l2Cache) {
                    await this.l2Cache.setex(key, this.config.l2.ttl, l3Result);
                }
                this.l1Cache.set(key, l3Result, this.config.l1.ttl);
                
                return {
                    data: l3Result,
                    source: 'L3',
                    hit: true
                };
            } catch (error) {
                // File doesn't exist in L3
                this.metrics.l3.misses++;
            }

            // Cache miss - not found in any tier
            return {
                data: null,
                source: null,
                hit: false
            };

        } catch (error) {
            console.error('Cache get error:', error);
            return {
                data: null,
                source: null,
                hit: false,
                error: error.message
            };
        }
    }

    async set(key, data, options = {}) {
        const dataBuffer = Buffer.isBuffer(data) ? data : Buffer.from(data);
        
        try {
            // Always store in L1 (Memory)
            this.l1Cache.set(key, dataBuffer, options.ttl || this.config.l1.ttl);

            // Store in L2 (Redis) if enabled and data is not too large
            if (this.l2Cache && dataBuffer.length <= this.config.l2.maxValueSize) {
                await this.l2Cache.setex(
                    key, 
                    options.ttl || this.config.l2.ttl, 
                    dataBuffer
                );
            }

            // Store in L3 (File System) for persistence
            const l3Path = this.generateL3Path(key);
            await fs.mkdir(path.dirname(l3Path), { recursive: true });
            await fs.writeFile(l3Path, dataBuffer);

            return true;
        } catch (error) {
            console.error('Cache set error:', error);
            return false;
        }
    }

    async delete(key) {
        try {
            // Remove from L1
            this.l1Cache.del(key);

            // Remove from L2
            if (this.l2Cache) {
                await this.l2Cache.del(key);
            }

            // Remove from L3
            const l3Path = this.generateL3Path(key);
            try {
                await fs.unlink(l3Path);
            } catch (error) {
                // File might not exist, ignore error
            }

            return true;
        } catch (error) {
            console.error('Cache delete error:', error);
            return false;
        }
    }

    async invalidatePattern(pattern) {
        let deletedCount = 0;

        try {
            // Invalidate L1 cache
            const l1Keys = this.l1Cache.keys();
            const matchingL1Keys = l1Keys.filter(key => 
                new RegExp(pattern.replace(/\*/g, '.*')).test(key)
            );
            
            matchingL1Keys.forEach(key => {
                this.l1Cache.del(key);
                deletedCount++;
            });

            // Invalidate L2 cache
            if (this.l2Cache) {
                const l2Keys = await this.l2Cache.keys(pattern);
                if (l2Keys.length > 0) {
                    await this.l2Cache.del(...l2Keys);
                    deletedCount += l2Keys.length;
                }
            }

            // Note: L3 invalidation would require a more complex implementation
            // For now, we rely on TTL expiration

            return deletedCount;
        } catch (error) {
            console.error('Cache invalidation error:', error);
            return 0;
        }
    }

    async warmCache(contentId, priorities = []) {
        // Pre-load popular content into cache hierarchy
        const warmingTasks = [];
        
        for (const priority of priorities) {
            const { quality, segments } = priority;
            
            for (const segmentId of segments) {
                const key = this.generateCacheKey(contentId, quality, `segment_${segmentId}`);
                
                warmingTasks.push(
                    this.warmCacheEntry(key, contentId, quality, segmentId)
                );
            }
        }

        const results = await Promise.allSettled(warmingTasks);
        const successful = results.filter(r => r.status === 'fulfilled').length;
        
        return {
            total: warmingTasks.length,
            successful,
            failed: warmingTasks.length - successful
        };
    }

    async warmCacheEntry(key, contentId, quality, segmentId) {
        // Check if already cached
        const cached = await this.get(key);
        if (cached.hit) {
            return true;
        }

        // Fetch from origin and cache
        try {
            const segmentData = await this.fetchFromOrigin(contentId, quality, segmentId);
            if (segmentData) {
                await this.set(key, segmentData);
                return true;
            }
        } catch (error) {
            console.error(`Failed to warm cache for ${key}:`, error);
        }
        
        return false;
    }

    async fetchFromOrigin(contentId, quality, segmentId) {
        // This would fetch from the storage service
        // Placeholder implementation
        return null;
    }

    getStats() {
        const l1Stats = this.l1Cache.getStats();
        
        return {
            metrics: this.metrics,
            l1: {
                ...l1Stats,
                hitRate: this.metrics.l1.hits / (this.metrics.l1.hits + this.metrics.l1.misses) || 0,
                keys: this.l1Cache.keys().length
            },
            l2: {
                hitRate: this.metrics.l2.hits / (this.metrics.l2.hits + this.metrics.l2.misses) || 0,
                connected: this.l2Cache ? true : false
            },
            l3: {
                hitRate: this.metrics.l3.hits / (this.metrics.l3.hits + this.metrics.l3.misses) || 0,
                path: this.l3CachePath
            },
            overall: {
                totalHitRate: this.metrics.totalHits / this.metrics.totalRequests || 0,
                requests: this.metrics.totalRequests,
                hits: this.metrics.totalHits,
                bytesServed: this.metrics.bytesServed
            }
        };
    }

    async cleanup() {
        // Cleanup expired entries in L3 cache
        try {
            await this.cleanupL3Cache();
        } catch (error) {
            console.error('L3 cache cleanup error:', error);
        }
    }

    async cleanupL3Cache() {
        const cleanupCutoff = Date.now() - (this.config.l3.ttl * 1000);
        let cleanedCount = 0;

        const traverseDirectory = async (dirPath) => {
            try {
                const entries = await fs.readdir(dirPath, { withFileTypes: true });
                
                for (const entry of entries) {
                    const fullPath = path.join(dirPath, entry.name);
                    
                    if (entry.isDirectory()) {
                        await traverseDirectory(fullPath);
                    } else if (entry.isFile() && entry.name.endsWith('.cache')) {
                        const stats = await fs.stat(fullPath);
                        
                        if (stats.mtime.getTime() < cleanupCutoff) {
                            await fs.unlink(fullPath);
                            cleanedCount++;
                        }
                    }
                }
            } catch (error) {
                // Directory might not exist or be inaccessible
            }
        };

        await traverseDirectory(this.l3CachePath);
        return cleanedCount;
    }

    async close() {
        if (this.l2Cache) {
            await this.l2Cache.quit();
        }
    }
}

module.exports = MultiTierCache;
