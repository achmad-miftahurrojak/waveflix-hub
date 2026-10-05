const CacheManager = require('./CacheManager');

describe('CacheManager', () => {
  let cacheManager;
  
  beforeEach(async () => {
    cacheManager = new CacheManager({
      redisUrl: 'redis://localhost:6379',
      logger: {
        info: jest.fn(),
        error: jest.fn(),
        warn: jest.fn(),
        debug: jest.fn()
      }
    });
  });

  afterEach(async () => {
    if (cacheManager) {
      await cacheManager.disconnect();
    }
  });

  describe('Cache Tier Functionality', () => {
    test('should initialize L1, L2, and L3 cache tiers', async () => {
      // Mock Redis connections for testing
      const mockL2Cache = {
        get: jest.fn(),
        setex: jest.fn(),
        del: jest.fn(),
        scanStream: jest.fn(),
        on: jest.fn(),
        disconnect: jest.fn()
      };
      
      const mockL3Cache = {
        get: jest.fn(),
        setex: jest.fn(),
        del: jest.fn(),
        scanStream: jest.fn(),
        on: jest.fn(),
        disconnect: jest.fn(),
        zincrby: jest.fn(),
        expire: jest.fn()
      };

      cacheManager.l2Cache = mockL2Cache;
      cacheManager.l3Cache = mockL3Cache;
      
      // Initialize L1 cache (LRU)
      await cacheManager.initialize();
      
      expect(cacheManager.l1Cache).toBeDefined();
      expect(cacheManager.l2Cache).toBeDefined();
      expect(cacheManager.l3Cache).toBeDefined();
    });

    test('should handle cache hierarchy correctly - L1 hit', async () => {
      // Mock setup
      cacheManager.l1Cache = new Map();
      cacheManager.l1Cache.set = jest.fn();
      cacheManager.l1Cache.get = jest.fn().mockReturnValue('cached_data');
      
      const result = await cacheManager.get('test_key');
      
      expect(result).toEqual({
        data: 'cached_data',
        tier: 'L1',
        cached_at: expect.any(Number)
      });
      expect(cacheManager.l1Cache.get).toHaveBeenCalledWith('test_key');
    });

    test('should promote from L2 to L1 on cache miss', async () => {
      // Setup mocks
      cacheManager.l1Cache = {
        get: jest.fn().mockReturnValue(null),
        set: jest.fn(),
        size: 0,
        max: 1000000
      };
      
      cacheManager.l2Cache = {
        get: jest.fn().mockResolvedValue(JSON.stringify({
          data: 'l2_cached_data',
          cached_at: Date.now()
        }))
      };
      
      cacheManager.l3Cache = {
        get: jest.fn().mockResolvedValue(null)
      };

      const result = await cacheManager.get('test_key');
      
      expect(result.tier).toBe('L2');
      expect(result.data).toBe('l2_cached_data');
      expect(cacheManager.l1Cache.set).toHaveBeenCalled();
    });

    test('should promote from L3 to L2 and L1 on cache miss', async () => {
      // Setup mocks
      cacheManager.l1Cache = {
        get: jest.fn().mockReturnValue(null),
        set: jest.fn(),
        size: 0,
        max: 1000000
      };
      
      cacheManager.l2Cache = {
        get: jest.fn().mockResolvedValue(null),
        setex: jest.fn()
      };
      
      cacheManager.l3Cache = {
        get: jest.fn().mockResolvedValue(JSON.stringify({
          data: 'l3_cached_data',
          cached_at: Date.now()
        }))
      };

      const result = await cacheManager.get('test_key');
      
      expect(result.tier).toBe('L3');
      expect(result.data).toBe('l3_cached_data');
      expect(cacheManager.l2Cache.setex).toHaveBeenCalled();
      expect(cacheManager.l1Cache.set).toHaveBeenCalled();
    });
  });

  describe('Cache Operations', () => {
    test('should set cache data in appropriate tier', async () => {
      // Setup mocks
      cacheManager.l1Cache = {
        set: jest.fn()
      };
      
      cacheManager.l2Cache = {
        setex: jest.fn()
      };
      
      cacheManager.l3Cache = {
        setex: jest.fn(),
        zincrby: jest.fn(),
        expire: jest.fn()
      };

      await cacheManager.set('test_key', 'test_data', {
        ttl: 3600,
        contentId: 'content123',
        quality: '720p'
      });
      
      expect(cacheManager.l1Cache.set).toHaveBeenCalledWith(
        'test_key',
        'test_data',
        { ttl: 3600000 }
      );
    });

    test('should invalidate cache across all tiers', async () => {
      // Setup mocks
      cacheManager.l1Cache = {
        entrySet: jest.fn().mockReturnValue([
          ['content:meta:test123', 'data1'],
          ['content:segment:test123:720p:001', 'data2']
        ]),
        delete: jest.fn()
      };
      
      cacheManager.l2Cache = {
        scanStream: jest.fn().mockReturnValue({
          on: jest.fn((event, callback) => {
            if (event === 'data') {
              callback(['content:meta:test123']);
            } else if (event === 'end') {
              callback();
            }
          })
        }),
        del: jest.fn()
      };
      
      cacheManager.l3Cache = {
        scanStream: jest.fn().mockReturnValue({
          on: jest.fn((event, callback) => {
            if (event === 'data') {
              callback(['content:meta:test123']);
            } else if (event === 'end') {
              callback();
            }
          })
        }),
        del: jest.fn()
      };

      await cacheManager.invalidate('test123');
      
      expect(cacheManager.l1Cache.delete).toHaveBeenCalled();
    });
  });

  describe('Cache Statistics and Monitoring', () => {
    test('should record cache hit statistics', () => {
      cacheManager.recordCacheHit('test_key', 'L1', 50);
      
      const stats = cacheManager.hitStats.get('cache_stats');
      expect(stats.hits).toBe(1);
      expect(stats.tier_stats.L1).toBe(1);
      expect(stats.hit_ratio).toBeGreaterThan(0);
    });

    test('should record cache miss statistics', () => {
      cacheManager.recordCacheMiss('test_key', 100);
      
      const stats = cacheManager.hitStats.get('cache_stats');
      expect(stats.misses).toBe(1);
      expect(stats.hit_ratio).toBe(0);
    });

    test('should calculate TTL based on content type', () => {
      // Test segment TTL
      const segmentTTL = cacheManager.calculateTTL('content:segment:123:720p:001', 'L1');
      expect(segmentTTL).toBe(3600);
      
      // Test playlist TTL
      const playlistTTL = cacheManager.calculateTTL('content:playlist:123:720p', 'L1');
      expect(playlistTTL).toBe(300);
      
      // Test metadata TTL (default)
      const metaTTL = cacheManager.calculateTTL('content:meta:123', 'L1');
      expect(metaTTL).toBe(3600);
    });
  });

  describe('Intelligent Routing', () => {
    test('should determine cache tier based on priority', () => {
      // High priority content
      const highPriorityTier = cacheManager.determineCacheTier({ priority: 1 });
      expect(highPriorityTier).toBe('L3');
      
      // Video segment
      const segmentTier = cacheManager.determineCacheTier({ segment: '001' });
      expect(segmentTier).toBe('L2');
      
      // Default metadata
      const defaultTier = cacheManager.determineCacheTier({});
      expect(defaultTier).toBe('L1');
    });

    test('should update popularity metrics', async () => {
      cacheManager.l3Cache = {
        zincrby: jest.fn(),
        expire: jest.fn()
      };
      
      await cacheManager.updatePopularityMetrics('content123', '720p', '001');
      
      expect(cacheManager.l3Cache.zincrby).toHaveBeenCalled();
      expect(cacheManager.l3Cache.expire).toHaveBeenCalled();
    });
  });

  describe('Pattern Matching', () => {
    test('should match cache key patterns correctly', () => {
      expect(cacheManager.matchesPattern('content:meta:123', 'content:meta:123')).toBe(true);
      expect(cacheManager.matchesPattern('content:segment:123:720p:001', 'content:segment:123:*')).toBe(true);
      expect(cacheManager.matchesPattern('content:playlist:123:720p', 'content:segment:*')).toBe(false);
    });
  });

  describe('Error Handling', () => {
    test('should handle Redis connection errors gracefully', async () => {
      cacheManager.l2Cache = {
        get: jest.fn().mockRejectedValue(new Error('Redis connection failed'))
      };
      
      cacheManager.l3Cache = {
        get: jest.fn().mockRejectedValue(new Error('Redis connection failed'))
      };
      
      cacheManager.l1Cache = {
        get: jest.fn().mockReturnValue(null)
      };

      await expect(cacheManager.get('test_key')).rejects.toThrow('Redis connection failed');
    });

    test('should handle cache set errors gracefully', async () => {
      cacheManager.l1Cache = {
        set: jest.fn().mockImplementation(() => {
          throw new Error('Cache full');
        })
      };
      
      await expect(cacheManager.set('test_key', 'test_data')).rejects.toThrow('Cache full');
    });
  });
});