const fastify = require('fastify');
const fs = require('fs');
const path = require('path');
const Redis = require('ioredis');
const NodeCache = require('node-cache');
const rangeParser = require('range-parser');
const mime = require('mime-types');
const { pipeline } = require('stream');
const { promisify } = require('util');

class StreamingServer {
    constructor(config) {
        this.config = config;
        this.app = null;
        this.redis = null;
        this.memoryCache = new NodeCache({ 
            stdTTL: config.cache.memoryTTL,
            checkperiod: 120,
            useClones: false
        });
        
        this.metrics = {
            requests: 0,
            cacheHits: 0,
            cacheMisses: 0,
            bytesServed: 0,
            activeStreams: 0
        };
    }

    async initialize() {
        // Initialize Fastify with HTTP/2 support
        this.app = fastify({
            logger: {
                level: this.config.logging.level,
                prettyPrint: process.env.NODE_ENV !== 'production'
            },
            http2: true,
            https: this.config.server.https ? {
                key: fs.readFileSync(this.config.server.https.keyPath),
                cert: fs.readFileSync(this.config.server.https.certPath)
            } : undefined,
            trustProxy: true,
            ignoreTrailingSlash: true,
            maxParamLength: 500
        });

        // Initialize Redis connection
        if (this.config.redis.enabled) {
            this.redis = new Redis({
                host: this.config.redis.host,
                port: this.config.redis.port,
                password: this.config.redis.password,
                db: this.config.redis.db,
                retryDelayOnFailover: 100,
                enableReadyCheck: false,
                maxRetriesPerRequest: 3
            });

            this.redis.on('error', (err) => {
                this.app.log.error('Redis connection error:', err);
            });

            this.redis.on('connect', () => {
                this.app.log.info('Connected to Redis cache');
            });
        }

        await this.registerPlugins();
        await this.registerRoutes();
        await this.registerHooks();
    }

    async registerPlugins() {
        // Security
        await this.app.register(require('@fastify/helmet'), {
            contentSecurityPolicy: false
        });

        // CORS
        await this.app.register(require('@fastify/cors'), {
            origin: true,
            methods: ['GET', 'HEAD', 'OPTIONS'],
            allowedHeaders: ['Range', 'If-Range', 'If-Modified-Since', 'Cache-Control']
        });

        // Compression
        await this.app.register(require('@fastify/compress'), {
            encodings: ['gzip', 'deflate', 'br']
        });

        // Rate limiting
        await this.app.register(require('@fastify/rate-limit'), {
            max: this.config.rateLimit.maxRequests,
            timeWindow: this.config.rateLimit.timeWindow,
            cache: 10000,
            allowList: this.config.rateLimit.whitelist,
            keyGenerator: (request) => {
                return request.headers['x-forwarded-for'] || request.ip;
            }
        });
    }

    async registerRoutes() {
        // Health endpoints
        this.app.get('/health', async (request, reply) => {
            return { 
                status: 'healthy', 
                timestamp: new Date().toISOString(),
                version: process.env.npm_package_version || '1.0.0'
            };
        });

        this.app.get('/metrics', async (request, reply) => {
            const memoryUsage = process.memoryUsage();
            return {
                ...this.metrics,
                memory: {
                    rss: memoryUsage.rss,
                    heapUsed: memoryUsage.heapUsed,
                    heapTotal: memoryUsage.heapTotal,
                    external: memoryUsage.external
                },
                uptime: process.uptime(),
                cache: {
                    memory: {
                        keys: this.memoryCache.keys().length,
                        stats: this.memoryCache.getStats()
                    },
                    redis: this.redis ? await this.getRedisInfo() : null
                }
            };
        });

        // HLS manifest serving
        this.app.get('/hls/:contentId/master.m3u8', async (request, reply) => {
            return this.serveMasterPlaylist(request, reply);
        });

        this.app.get('/hls/:contentId/:quality/playlist.m3u8', async (request, reply) => {
            return this.servePlaylist(request, reply);
        });

        // Video segment streaming with range support
        this.app.get('/hls/:contentId/:quality/segment_:segmentId.ts', async (request, reply) => {
            return this.serveSegment(request, reply);
        });

        // Generic file streaming
        this.app.get('/stream/:contentId/*', async (request, reply) => {
            return this.serveFile(request, reply);
        });

        // Thumbnail serving
        this.app.get('/thumbnails/:contentId/:index', async (request, reply) => {
            return this.serveThumbnail(request, reply);
        });

        // Cache management
        this.app.post('/cache/invalidate/:contentId', async (request, reply) => {
            return this.invalidateCache(request, reply);
        });

        this.app.get('/cache/stats', async (request, reply) => {
            return this.getCacheStats(request, reply);
        });
    }

    async registerHooks() {
        // Request logging and metrics
        this.app.addHook('onRequest', async (request, reply) => {
            this.metrics.requests++;
            request.startTime = Date.now();
        });

        // Response time and metrics
        this.app.addHook('onResponse', async (request, reply) => {
            const responseTime = Date.now() - request.startTime;
            this.app.log.info({
                method: request.method,
                url: request.url,
                statusCode: reply.statusCode,
                responseTime: `${responseTime}ms`,
                userAgent: request.headers['user-agent']
            });
        });

        // Error handling
        this.app.setErrorHandler(async (error, request, reply) => {
            this.app.log.error({
                error: error.message,
                stack: error.stack,
                url: request.url,
                method: request.method
            });

            const statusCode = error.statusCode || 500;
            return reply.status(statusCode).send({
                error: error.message,
                statusCode,
                timestamp: new Date().toISOString()
            });
        });
    }

    async serveMasterPlaylist(request, reply) {
        const { contentId } = request.params;
        const cacheKey = `master:${contentId}`;
        
        try {
            // Check memory cache first
            let playlist = this.memoryCache.get(cacheKey);
            
            if (!playlist) {
                // Check Redis cache
                if (this.redis) {
                    playlist = await this.redis.get(cacheKey);
                }
                
                if (!playlist) {
                    // Fetch from storage service
                    playlist = await this.fetchFromStorage(`/hls/${contentId}/master.m3u8`);
                    
                    if (playlist) {
                        // Cache the playlist
                        this.memoryCache.set(cacheKey, playlist, this.config.cache.playlistTTL);
                        if (this.redis) {
                            await this.redis.setex(cacheKey, this.config.cache.playlistTTL, playlist);
                        }
                    }
                    this.metrics.cacheMisses++;
                } else {
                    this.memoryCache.set(cacheKey, playlist, this.config.cache.playlistTTL);
                    this.metrics.cacheHits++;
                }
            } else {
                this.metrics.cacheHits++;
            }

            if (!playlist) {
                return reply.status(404).send({ error: 'Master playlist not found' });
            }

            return reply
                .header('Content-Type', 'application/vnd.apple.mpegurl')
                .header('Cache-Control', `public, max-age=${this.config.cache.playlistTTL}`)
                .header('Access-Control-Allow-Origin', '*')
                .send(playlist);

        } catch (error) {
            this.app.log.error('Error serving master playlist:', error);
            return reply.status(500).send({ error: 'Internal server error' });
        }
    }

    async servePlaylist(request, reply) {
        const { contentId, quality } = request.params;
        const cacheKey = `playlist:${contentId}:${quality}`;
        
        try {
            let playlist = this.memoryCache.get(cacheKey);
            
            if (!playlist) {
                if (this.redis) {
                    playlist = await this.redis.get(cacheKey);
                }
                
                if (!playlist) {
                    playlist = await this.fetchFromStorage(`/hls/${contentId}/${quality}/playlist.m3u8`);
                    
                    if (playlist) {
                        const ttl = this.config.cache.playlistTTL;
                        this.memoryCache.set(cacheKey, playlist, ttl);
                        if (this.redis) {
                            await this.redis.setex(cacheKey, ttl, playlist);
                        }
                    }
                    this.metrics.cacheMisses++;
                } else {
                    this.memoryCache.set(cacheKey, playlist, this.config.cache.playlistTTL);
                    this.metrics.cacheHits++;
                }
            } else {
                this.metrics.cacheHits++;
            }

            if (!playlist) {
                return reply.status(404).send({ error: 'Playlist not found' });
            }

            return reply
                .header('Content-Type', 'application/vnd.apple.mpegurl')
                .header('Cache-Control', `public, max-age=${this.config.cache.playlistTTL}`)
                .header('Access-Control-Allow-Origin', '*')
                .send(playlist);

        } catch (error) {
            this.app.log.error('Error serving playlist:', error);
            return reply.status(500).send({ error: 'Internal server error' });
        }
    }

    async serveSegment(request, reply) {
        const { contentId, quality, segmentId } = request.params;
        const range = request.headers.range;
        
        try {
            this.metrics.activeStreams++;
            
            const segmentPath = `/hls/${contentId}/${quality}/segment_${segmentId}.ts`;
            const fileInfo = await this.getFileInfo(segmentPath);
            
            if (!fileInfo) {
                return reply.status(404).send({ error: 'Segment not found' });
            }

            // Handle range requests for seeking
            let start = 0;
            let end = fileInfo.size - 1;
            let statusCode = 200;

            if (range) {
                const ranges = rangeParser(fileInfo.size, range);
                if (ranges === -1 || ranges === -2 || !ranges.length) {
                    return reply.status(416).send({ error: 'Range not satisfiable' });
                }
                
                const { start: rangeStart, end: rangeEnd } = ranges[0];
                start = rangeStart;
                end = rangeEnd;
                statusCode = 206;
                
                reply.header('Content-Range', `bytes ${start}-${end}/${fileInfo.size}`);
                reply.header('Accept-Ranges', 'bytes');
            }

            const contentLength = end - start + 1;
            
            reply
                .status(statusCode)
                .header('Content-Type', 'video/mp2t')
                .header('Content-Length', contentLength)
                .header('Cache-Control', `public, max-age=${this.config.cache.segmentTTL}, immutable`)
                .header('Access-Control-Allow-Origin', '*');

            // Stream the segment
            const stream = await this.createReadStream(segmentPath, { start, end });
            
            stream.on('end', () => {
                this.metrics.activeStreams--;
                this.metrics.bytesServed += contentLength;
            });

            stream.on('error', (error) => {
                this.metrics.activeStreams--;
                this.app.log.error('Stream error:', error);
            });

            return reply.send(stream);

        } catch (error) {
            this.metrics.activeStreams--;
            this.app.log.error('Error serving segment:', error);
            return reply.status(500).send({ error: 'Internal server error' });
        }
    }

    async serveFile(request, reply) {
        const { contentId } = request.params;
        const filePath = request.params['*'];
        const fullPath = `/content/${contentId}/${filePath}`;
        
        try {
            const fileInfo = await this.getFileInfo(fullPath);
            
            if (!fileInfo) {
                return reply.status(404).send({ error: 'File not found' });
            }

            const mimeType = mime.lookup(filePath) || 'application/octet-stream';
            const range = request.headers.range;
            
            let start = 0;
            let end = fileInfo.size - 1;
            let statusCode = 200;

            if (range) {
                const ranges = rangeParser(fileInfo.size, range);
                if (ranges !== -1 && ranges !== -2 && ranges.length) {
                    const { start: rangeStart, end: rangeEnd } = ranges[0];
                    start = rangeStart;
                    end = rangeEnd;
                    statusCode = 206;
                    
                    reply.header('Content-Range', `bytes ${start}-${end}/${fileInfo.size}`);
                    reply.header('Accept-Ranges', 'bytes');
                }
            }

            const contentLength = end - start + 1;
            
            reply
                .status(statusCode)
                .header('Content-Type', mimeType)
                .header('Content-Length', contentLength)
                .header('Cache-Control', 'public, max-age=3600')
                .header('Access-Control-Allow-Origin', '*');

            const stream = await this.createReadStream(fullPath, { start, end });
            
            stream.on('end', () => {
                this.metrics.bytesServed += contentLength;
            });

            return reply.send(stream);

        } catch (error) {
            this.app.log.error('Error serving file:', error);
            return reply.status(500).send({ error: 'Internal server error' });
        }
    }

    async serveThumbnail(request, reply) {
        const { contentId, index } = request.params;
        const cacheKey = `thumbnail:${contentId}:${index}`;
        
        try {
            // Check if thumbnail is in cache
            let thumbnail = this.memoryCache.get(cacheKey);
            
            if (!thumbnail) {
                const thumbnailPath = `/thumbnails/${contentId}/thumb_${index}.jpg`;
                thumbnail = await this.fetchFromStorage(thumbnailPath, { binary: true });
                
                if (thumbnail) {
                    this.memoryCache.set(cacheKey, thumbnail, this.config.cache.thumbnailTTL);
                }
                this.metrics.cacheMisses++;
            } else {
                this.metrics.cacheHits++;
            }

            if (!thumbnail) {
                return reply.status(404).send({ error: 'Thumbnail not found' });
            }

            return reply
                .header('Content-Type', 'image/jpeg')
                .header('Cache-Control', `public, max-age=${this.config.cache.thumbnailTTL}`)
                .header('Access-Control-Allow-Origin', '*')
                .send(thumbnail);

        } catch (error) {
            this.app.log.error('Error serving thumbnail:', error);
            return reply.status(500).send({ error: 'Internal server error' });
        }
    }

    async fetchFromStorage(path, options = {}) {
        // This would integrate with the storage service
        // For now, return a placeholder implementation
        try {
            const response = await fetch(`${this.config.storage.baseUrl}/api/v1/files${path}`, {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${this.config.storage.apiKey}`
                }
            });

            if (!response.ok) {
                return null;
            }

            if (options.binary) {
                return await response.buffer();
            } else {
                return await response.text();
            }
        } catch (error) {
            this.app.log.error('Error fetching from storage:', error);
            return null;
        }
    }

    async getFileInfo(path) {
        try {
            const response = await fetch(`${this.config.storage.baseUrl}/api/v1/files${path}/info`, {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${this.config.storage.apiKey}`
                }
            });

            if (!response.ok) {
                return null;
            }

            return await response.json();
        } catch (error) {
            this.app.log.error('Error getting file info:', error);
            return null;
        }
    }

    async createReadStream(path, options = {}) {
        // This would create a stream from the storage service
        // Implementation would depend on storage backend
        const response = await fetch(`${this.config.storage.baseUrl}/api/v1/files${path}`, {
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${this.config.storage.apiKey}`,
                'Range': options.start !== undefined ? `bytes=${options.start}-${options.end}` : undefined
            }
        });

        if (!response.ok) {
            throw new Error(`Failed to fetch file: ${response.statusText}`);
        }

        return response.body;
    }

    async invalidateCache(request, reply) {
        const { contentId } = request.params;
        
        try {
            // Clear memory cache
            const keys = this.memoryCache.keys();
            const keysToDelete = keys.filter(key => key.includes(contentId));
            
            keysToDelete.forEach(key => {
                this.memoryCache.del(key);
            });

            // Clear Redis cache
            if (this.redis) {
                const redisKeys = await this.redis.keys(`*${contentId}*`);
                if (redisKeys.length > 0) {
                    await this.redis.del(...redisKeys);
                }
            }

            return {
                success: true,
                message: `Invalidated ${keysToDelete.length} cache entries for content ${contentId}`
            };

        } catch (error) {
            this.app.log.error('Error invalidating cache:', error);
            return reply.status(500).send({ error: 'Cache invalidation failed' });
        }
    }

    async getCacheStats(request, reply) {
        const memoryStats = this.memoryCache.getStats();
        let redisStats = null;

        if (this.redis) {
            redisStats = await this.getRedisInfo();
        }

        return {
            memory: {
                keys: memoryStats.keys,
                hits: memoryStats.hits,
                misses: memoryStats.misses,
                hitRate: memoryStats.hits / (memoryStats.hits + memoryStats.misses),
                vsize: memoryStats.vsize
            },
            redis: redisStats,
            global: {
                totalHits: this.metrics.cacheHits,
                totalMisses: this.metrics.cacheMisses,
                hitRate: this.metrics.cacheHits / (this.metrics.cacheHits + this.metrics.cacheMisses)
            }
        };
    }

    async getRedisInfo() {
        if (!this.redis) return null;

        try {
            const info = await this.redis.info('memory');
            const keyspace = await this.redis.info('keyspace');
            
            return {
                connected: true,
                memory: info,
                keyspace: keyspace
            };
        } catch (error) {
            return { connected: false, error: error.message };
        }
    }

    async start() {
        try {
            const port = this.config.server.port;
            const host = this.config.server.host;
            
            await this.app.listen({ port, host });
            
            this.app.log.info(`?? Streaming server started on ${host}:${port}`);
            this.app.log.info(`?? HTTP/2 enabled: ${!!this.config.server.https}`);
            this.app.log.info(`???  Redis cache: ${this.config.redis.enabled ? 'enabled' : 'disabled'}`);
            
        } catch (error) {
            this.app.log.error('Error starting server:', error);
            process.exit(1);
        }
    }

    async stop() {
        if (this.redis) {
            await this.redis.quit();
        }
        
        if (this.app) {
            await this.app.close();
        }
        
        this.app.log.info('Streaming server stopped');
    }
}

module.exports = StreamingServer;
