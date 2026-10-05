const express = require('express');
const cors = require('cors');
const helmet = require('helmet');
const morgan = require('morgan');
const compression = require('compression');
const winston = require('winston');

const CacheManager = require('./cache/CacheManager');
const EdgeCoordinator = require('./coordination/EdgeCoordinator');
const AnalyticsService = require('./analytics/AnalyticsService');

const logger = winston.createLogger({
  level: 'info',
  format: winston.format.combine(
    winston.format.timestamp(),
    winston.format.json()
  ),
  transports: [
    new winston.transports.File({ filename: 'error.log', level: 'error' }),
    new winston.transports.File({ filename: 'combined.log' }),
    new winston.transports.Console({ format: winston.format.simple() })
  ]
});

class CacheService {
  constructor() {
    this.app = express();
    this.port = process.env.PORT || 3003;
    this.cacheManager = null;
    this.edgeCoordinator = null;
    this.analytics = null;
  }

  async initialize() {
    // Initialize middleware
    this.app.use(helmet());
    this.app.use(cors());
    this.app.use(compression());
    this.app.use(morgan('combined'));
    this.app.use(express.json());

    // Initialize cache components
    this.cacheManager = new CacheManager({
      redisUrl: process.env.REDIS_URL || 'redis://localhost:6379',
      logger
    });

    this.edgeCoordinator = new EdgeCoordinator({
      nodeId: process.env.NODE_ID || 'cache-node-1',
      region: process.env.REGION || 'us-east-1',
      cacheManager: this.cacheManager,
      logger
    });

    this.analytics = new AnalyticsService({
      cacheManager: this.cacheManager,
      logger
    });

    await this.cacheManager.initialize();
    await this.edgeCoordinator.initialize();
    await this.analytics.initialize();

    this.setupRoutes();
  }

  setupRoutes() {
    // Health check
    this.app.get('/health', (req, res) => {
      res.json({
        status: 'healthy',
        timestamp: new Date().toISOString(),
        nodeId: this.edgeCoordinator.nodeId,
        region: this.edgeCoordinator.region
      });
    });

    // Cache operations
    this.app.get('/cache/:contentId', async (req, res) => {
      try {
        const { contentId } = req.params;
        const { quality = '720p', segment } = req.query;

        const cacheKey = segment 
          ? `content:segment:${contentId}:${quality}:${segment}`
          : `content:meta:${contentId}`;

        const result = await this.cacheManager.get(cacheKey);
        
        if (result) {
          this.analytics.recordHit(contentId, quality, segment);
          res.set('X-Cache-Status', 'HIT');
          res.set('X-Cache-Tier', result.tier);
          
          if (segment) {
            res.set('Content-Type', 'video/mp2t');
          } else {
            res.set('Content-Type', 'application/json');
          }
          
          res.send(result.data);
        } else {
          this.analytics.recordMiss(contentId, quality, segment);
          res.set('X-Cache-Status', 'MISS');
          res.status(404).json({ error: 'Content not found in cache' });
        }
      } catch (error) {
        logger.error('Cache retrieval error:', error);
        res.status(500).json({ error: 'Internal server error' });
      }
    });

    this.app.put('/cache/:contentId', async (req, res) => {
      try {
        const { contentId } = req.params;
        const { quality = '720p', segment, data, ttl = 3600 } = req.body;

        const cacheKey = segment 
          ? `content:segment:${contentId}:${quality}:${segment}`
          : `content:meta:${contentId}`;

        await this.cacheManager.set(cacheKey, data, { ttl, contentId, quality, segment });
        
        res.json({ 
          success: true,
          key: cacheKey,
          ttl
        });
      } catch (error) {
        logger.error('Cache storage error:', error);
        res.status(500).json({ error: 'Internal server error' });
      }
    });

    this.app.delete('/cache/:contentId', async (req, res) => {
      try {
        const { contentId } = req.params;
        await this.cacheManager.invalidate(contentId);
        res.json({ success: true });
      } catch (error) {
        logger.error('Cache invalidation error:', error);
        res.status(500).json({ error: 'Internal server error' });
      }
    });

    // Analytics endpoints
    this.app.get('/analytics/stats', async (req, res) => {
      try {
        const stats = await this.analytics.getStats();
        res.json(stats);
      } catch (error) {
        logger.error('Analytics error:', error);
        res.status(500).json({ error: 'Internal server error' });
      }
    });

    // Edge coordination
    this.app.get('/coordination/nodes', async (req, res) => {
      try {
        const nodes = await this.edgeCoordinator.getHealthyNodes();
        res.json(nodes);
      } catch (error) {
        logger.error('Coordination error:', error);
        res.status(500).json({ error: 'Internal server error' });
      }
    });
  }

  async start() {
    await this.initialize();
    
    this.server = this.app.listen(this.port, () => {
      logger.info(`Cache service running on port ${this.port}`);
    });
  }

  async stop() {
    if (this.server) {
      this.server.close();
    }
    if (this.cacheManager) {
      await this.cacheManager.disconnect();
    }
    if (this.edgeCoordinator) {
      await this.edgeCoordinator.disconnect();
    }
  }
}

module.exports = CacheService;

// Start service if run directly
if (require.main === module) {
  const service = new CacheService();
  service.start().catch(error => {
    console.error('Failed to start cache service:', error);
    process.exit(1);
  });

  process.on('SIGTERM', async () => {
    console.log('SIGTERM received, shutting down gracefully');
    await service.stop();
    process.exit(0);
  });

  process.on('SIGINT', async () => {
    console.log('SIGINT received, shutting down gracefully');
    await service.stop();
    process.exit(0);
  });
}