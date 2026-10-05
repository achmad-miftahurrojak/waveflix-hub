const StreamingServer = require('./src/server/StreamingServer');
const MultiTierCache = require('./src/cache/MultiTierCache');
const AdaptiveBitrateStreaming = require('./src/streaming/AdaptiveBitrateStreaming');
const config = require('./config/default.json');

class WaveFlixStreamingService {
    constructor() {
        this.config = this.loadConfig();
        this.cache = null;
        this.adaptiveStreaming = null;
        this.server = null;
    }

    loadConfig() {
        // Merge environment variables with default config
        const envConfig = {
            server: {
                port: parseInt(process.env.PORT) || config.server.port,
                host: process.env.HOST || config.server.host
            },
            redis: {
                enabled: process.env.REDIS_ENABLED === 'true' || config.redis.enabled,
                host: process.env.REDIS_HOST || config.redis.host,
                port: parseInt(process.env.REDIS_PORT) || config.redis.port,
                password: process.env.REDIS_PASSWORD || config.redis.password,
                db: parseInt(process.env.REDIS_DB) || config.redis.db
            },
            storage: {
                baseUrl: process.env.STORAGE_SERVICE_URL || config.storage.baseUrl,
                apiKey: process.env.STORAGE_API_KEY || config.storage.apiKey
            },
            cache: {
                ...config.cache,
                l3: {
                    ...config.cache.l3,
                    path: process.env.L3_CACHE_PATH || config.cache.l3.path
                }
            }
        };

        return { ...config, ...envConfig };
    }

    async initialize() {
        console.log('?? Initializing WaveFlix Streaming Service...');
        
        // Initialize multi-tier cache
        this.cache = new MultiTierCache(this.config.cache);
        
        // Initialize adaptive bitrate streaming
        this.adaptiveStreaming = new AdaptiveBitrateStreaming(this.config.streaming);
        
        // Initialize HTTP server
        this.server = new StreamingServer(this.config);
        
        // Integrate components
        this.server.cache = this.cache;
        this.server.adaptiveStreaming = this.adaptiveStreaming;
        
        await this.server.initialize();
        
        console.log('? All components initialized successfully');
    }

    async start() {
        try {
            await this.initialize();
            await this.server.start();
            
            // Start background tasks
            this.startBackgroundTasks();
            
            console.log(`?? WaveFlix Streaming Service is ready!`);
            console.log(`?? Server: http://${this.config.server.host}:${this.config.server.port}`);
            console.log(`???  Cache: ${this.config.redis.enabled ? 'Redis enabled' : 'Memory only'}`);
            console.log(`?? Adaptive streaming: ${this.config.streaming.qualityLevels.length} quality levels`);
            
        } catch (error) {
            console.error('? Failed to start streaming service:', error);
            process.exit(1);
        }
    }

    startBackgroundTasks() {
        // Cache cleanup every hour
        setInterval(async () => {
            try {
                console.log('?? Running cache cleanup...');
                await this.cache.cleanup();
            } catch (error) {
                console.error('Cache cleanup error:', error);
            }
        }, 3600000); // 1 hour

        // Session cleanup every 15 minutes
        setInterval(() => {
            try {
                console.log('?? Cleaning up old streaming sessions...');
                this.adaptiveStreaming.cleanupSessions();
            } catch (error) {
                console.error('Session cleanup error:', error);
            }
        }, 900000); // 15 minutes

        // Metrics logging every 5 minutes
        setInterval(() => {
            try {
                const cacheStats = this.cache.getStats();
                const streamingStats = this.adaptiveStreaming.getGlobalAnalytics();
                
                console.log('?? Performance Metrics:');
                console.log(`   Cache Hit Rate: ${(cacheStats.overall.totalHitRate * 100).toFixed(2)}%`);
                console.log(`   Active Sessions: ${streamingStats.activeSessions}`);
                console.log(`   Quality Switches: ${streamingStats.qualitySwitches}`);
                console.log(`   Bytes Served: ${(cacheStats.overall.bytesServed / 1024 / 1024).toFixed(2)} MB`);
                
            } catch (error) {
                console.error('Metrics logging error:', error);
            }
        }, 300000); // 5 minutes
    }

    async stop() {
        console.log('?? Stopping WaveFlix Streaming Service...');
        
        if (this.server) {
            await this.server.stop();
        }
        
        if (this.cache) {
            await this.cache.close();
        }
        
        console.log('? Streaming service stopped gracefully');
        process.exit(0);
    }
}

// Handle graceful shutdown
process.on('SIGTERM', async () => {
    console.log('Received SIGTERM signal');
    if (global.streamingService) {
        await global.streamingService.stop();
    }
});

process.on('SIGINT', async () => {
    console.log('Received SIGINT signal');
    if (global.streamingService) {
        await global.streamingService.stop();
    }
});

// Start the service
if (require.main === module) {
    global.streamingService = new WaveFlixStreamingService();
    global.streamingService.start();
}

module.exports = WaveFlixStreamingService;
