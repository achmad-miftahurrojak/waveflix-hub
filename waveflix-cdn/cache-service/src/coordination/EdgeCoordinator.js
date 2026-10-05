const Redis = require('ioredis');

class EdgeCoordinator {
  constructor(options = {}) {
    this.nodeId = options.nodeId;
    this.region = options.region;
    this.cacheManager = options.cacheManager;
    this.logger = options.logger;
    
    this.redis = null;
    this.heartbeatInterval = null;
    this.healthCheckInterval = null;
    
    // Node health configuration
    this.healthConfig = {
      heartbeatTtl: 30, // seconds
      heartbeatInterval: 10, // seconds
      healthCheckInterval: 15, // seconds
      maxUnhealthyTime: 60, // seconds
      loadThreshold: 0.8 // 80% load threshold
    };

    // Routing weights for intelligent cache hierarchy
    this.routingWeights = {
      latency: 0.4,
      load: 0.3,
      hit_ratio: 0.2,
      proximity: 0.1
    };
  }

  async initialize() {
    try {
      // Connect to Redis for coordination
      this.redis = new Redis({
        host: process.env.COORDINATION_REDIS_HOST || 'localhost',
        port: process.env.COORDINATION_REDIS_PORT || 6379,
        password: process.env.REDIS_PASSWORD,
        keyPrefix: 'coord:',
        retryDelayOnFailover: 100,
        maxRetriesPerRequest: 3
      });

      // Start heartbeat and health monitoring
      await this.startHeartbeat();
      await this.startHealthMonitoring();
      
      this.logger?.info('Edge coordinator initialized', {
        nodeId: this.nodeId,
        region: this.region
      });

    } catch (error) {
      this.logger?.error('Failed to initialize edge coordinator:', error);
      throw error;
    }
  }

  async startHeartbeat() {
    // Register this node
    await this.registerNode();
    
    // Send periodic heartbeats
    this.heartbeatInterval = setInterval(async () => {
      try {
        await this.sendHeartbeat();
      } catch (error) {
        this.logger?.error('Heartbeat failed:', error);
      }
    }, this.healthConfig.heartbeatInterval * 1000);
  }

  async startHealthMonitoring() {
    this.healthCheckInterval = setInterval(async () => {
      try {
        await this.performHealthCheck();
        await this.updateRoutingTable();
      } catch (error) {
        this.logger?.error('Health check failed:', error);
      }
    }, this.healthConfig.healthCheckInterval * 1000);
  }

  async registerNode() {
    const nodeInfo = {
      nodeId: this.nodeId,
      region: this.region,
      status: 'healthy',
      tier: this.determineTier(),
      capabilities: this.getCapabilities(),
      registered_at: Date.now(),
      last_heartbeat: Date.now()
    };

    await this.redis.hset(`nodes:${this.nodeId}`, nodeInfo);
    await this.redis.sadd(`regions:${this.region}`, this.nodeId);
    
    this.logger?.info('Node registered', nodeInfo);
  }

  async sendHeartbeat() {
    const metrics = await this.collectMetrics();
    const heartbeatData = {
      nodeId: this.nodeId,
      region: this.region,
      status: 'healthy',
      last_heartbeat: Date.now(),
      metrics: JSON.stringify(metrics)
    };

    // Update node information with TTL
    await this.redis.hset(`nodes:${this.nodeId}`, heartbeatData);
    await this.redis.expire(`nodes:${this.nodeId}`, this.healthConfig.heartbeatTtl);
    
    // Add to active nodes list
    await this.redis.sadd(`active_nodes:${this.region}`, this.nodeId);
    await this.redis.expire(`active_nodes:${this.region}`, this.healthConfig.heartbeatTtl);
  }

  async collectMetrics() {
    try {
      const cacheStats = await this.cacheManager.getStats();
      const systemMetrics = await this.getSystemMetrics();
      
      return {
        cache: {
          hit_ratio: cacheStats.hit_ratio || 0,
          l1_utilization: cacheStats.cache_info.l1_size / cacheStats.cache_info.l1_max,
          avg_response_time: cacheStats.avg_response_time || 0,
          hits: cacheStats.hits || 0,
          misses: cacheStats.misses || 0
        },
        system: systemMetrics,
        timestamp: Date.now()
      };
    } catch (error) {
      this.logger?.error('Failed to collect metrics:', error);
      return {
        cache: { hit_ratio: 0, l1_utilization: 0, avg_response_time: 0 },
        system: { cpu: 0, memory: 0, load: 0 },
        timestamp: Date.now()
      };
    }
  }

  async getSystemMetrics() {
    // Basic system metrics - in production, use proper monitoring
    const used = process.memoryUsage();
    const total = process.env.NODE_MEMORY_LIMIT || (4 * 1024 * 1024 * 1024); // 4GB default
    
    return {
      cpu: 0, // Would implement actual CPU monitoring
      memory: used.heapUsed / total,
      load: Math.random() * 0.5, // Mock load - implement actual load monitoring
      uptime: process.uptime()
    };
  }

  async performHealthCheck() {
    try {
      // Check cache connectivity
      await this.cacheManager.l1Cache.set('health_check', Date.now(), { ttl: 5000 });
      const healthValue = this.cacheManager.l1Cache.get('health_check');
      
      if (!healthValue) {
        throw new Error('L1 cache health check failed');
      }

      // Check Redis connectivity
      await this.redis.ping();
      
      // Update health status
      await this.redis.hset(`nodes:${this.nodeId}`, 'status', 'healthy');
      
    } catch (error) {
      this.logger?.error('Health check failed:', error);
      await this.redis.hset(`nodes:${this.nodeId}`, 'status', 'unhealthy');
      throw error;
    }
  }

  async updateRoutingTable() {
    try {
      const allNodes = await this.getHealthyNodes();
      const routingTable = {};

      for (const node of allNodes) {
        const score = await this.calculateRoutingScore(node);
        routingTable[node.nodeId] = {
          ...node,
          routing_score: score,
          updated_at: Date.now()
        };
      }

      // Sort nodes by routing score (higher is better)
      const sortedNodes = Object.values(routingTable).sort(
        (a, b) => b.routing_score - a.routing_score
      );

      await this.redis.set(
        `routing_table:${this.region}`,
        JSON.stringify(sortedNodes),
        'EX',
        this.healthConfig.healthCheckInterval * 2
      );

    } catch (error) {
      this.logger?.error('Failed to update routing table:', error);
    }
  }

  async calculateRoutingScore(node) {
    try {
      const metrics = JSON.parse(node.metrics || '{}');
      
      // Normalize metrics to 0-1 scale
      const latencyScore = Math.max(0, 1 - (metrics.cache?.avg_response_time || 0) / 1000);
      const loadScore = Math.max(0, 1 - (metrics.system?.load || 0));
      const hitRatioScore = metrics.cache?.hit_ratio || 0;
      const proximityScore = node.region === this.region ? 1 : 0.5;

      // Calculate weighted score
      const score = (
        latencyScore * this.routingWeights.latency +
        loadScore * this.routingWeights.load +
        hitRatioScore * this.routingWeights.hit_ratio +
        proximityScore * this.routingWeights.proximity
      );

      return Math.round(score * 100) / 100; // Round to 2 decimal places

    } catch (error) {
      this.logger?.error('Failed to calculate routing score:', error);
      return 0;
    }
  }

  async getHealthyNodes() {
    try {
      const activeNodes = await this.redis.smembers(`active_nodes:${this.region}`);
      const nodes = [];

      for (const nodeId of activeNodes) {
        const nodeData = await this.redis.hgetall(`nodes:${nodeId}`);
        if (nodeData && nodeData.status === 'healthy') {
          nodes.push({
            nodeId,
            region: nodeData.region,
            status: nodeData.status,
            tier: nodeData.tier,
            capabilities: nodeData.capabilities,
            metrics: nodeData.metrics,
            last_heartbeat: parseInt(nodeData.last_heartbeat)
          });
        }
      }

      return nodes;

    } catch (error) {
      this.logger?.error('Failed to get healthy nodes:', error);
      return [];
    }
  }

  async findOptimalNode(contentId, options = {}) {
    try {
      const { quality, segment, preferredTier } = options;
      const routingTable = await this.redis.get(`routing_table:${this.region}`);
      
      if (!routingTable) {
        return null;
      }

      const nodes = JSON.parse(routingTable);
      
      // Filter nodes based on criteria
      let candidateNodes = nodes.filter(node => {
        if (preferredTier && node.tier !== preferredTier) return false;
        if (node.routing_score < 0.5) return false; // Minimum quality threshold
        return true;
      });

      if (candidateNodes.length === 0) {
        candidateNodes = nodes.slice(0, 3); // Fallback to top 3 nodes
      }

      // For video segments, prefer nodes with high cache hit ratio
      if (segment) {
        candidateNodes.sort((a, b) => {
          const aMetrics = JSON.parse(a.metrics || '{}');
          const bMetrics = JSON.parse(b.metrics || '{}');
          return (bMetrics.cache?.hit_ratio || 0) - (aMetrics.cache?.hit_ratio || 0);
        });
      }

      return candidateNodes[0] || null;

    } catch (error) {
      this.logger?.error('Failed to find optimal node:', error);
      return null;
    }
  }

  async routeRequest(contentId, options = {}) {
    const optimalNode = await this.findOptimalNode(contentId, options);
    
    if (!optimalNode || optimalNode.nodeId === this.nodeId) {
      // Handle locally
      return { strategy: 'local', node: optimalNode };
    }

    // Route to optimal node
    return { 
      strategy: 'route', 
      node: optimalNode,
      endpoint: `http://${optimalNode.nodeId}:3003/cache/${contentId}`
    };
  }

  determineTier() {
    // Determine cache tier based on environment or configuration
    const tier = process.env.CACHE_TIER;
    if (tier && ['L1', 'L2', 'L3'].includes(tier)) {
      return tier;
    }

    // Default tier determination logic
    if (this.region.includes('edge')) return 'L1';
    if (this.region.includes('regional')) return 'L2';
    return 'L3'; // Origin
  }

  getCapabilities() {
    return {
      video_transcoding: process.env.ENABLE_TRANSCODING === 'true',
      content_processing: process.env.ENABLE_PROCESSING === 'true',
      analytics: process.env.ENABLE_ANALYTICS === 'true',
      cache_tiers: [this.determineTier()]
    };
  }

  async disconnect() {
    try {
      // Clear intervals
      if (this.heartbeatInterval) {
        clearInterval(this.heartbeatInterval);
      }
      if (this.healthCheckInterval) {
        clearInterval(this.healthCheckInterval);
      }

      // Remove from active nodes
      await this.redis.srem(`active_nodes:${this.region}`, this.nodeId);
      
      // Disconnect Redis
      if (this.redis) {
        await this.redis.disconnect();
      }

      this.logger?.info('Edge coordinator disconnected');
    } catch (error) {
      this.logger?.error('Error during edge coordinator disconnect:', error);
    }
  }
}

module.exports = EdgeCoordinator;