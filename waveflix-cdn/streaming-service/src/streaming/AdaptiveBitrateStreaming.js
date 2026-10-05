class AdaptiveBitrateStreaming {
    constructor(config) {
        this.config = config;
        this.qualityLevels = config.qualityLevels || [
            { name: '240p', bandwidth: 400000, resolution: '426x240' },
            { name: '360p', bandwidth: 800000, resolution: '640x360' },
            { name: '480p', bandwidth: 1200000, resolution: '854x480' },
            { name: '720p', bandwidth: 2500000, resolution: '1280x720' },
            { name: '1080p', bandwidth: 5000000, resolution: '1920x1080' },
            { name: '4k', bandwidth: 15000000, resolution: '3840x2160' }
        ];
        
        this.sessions = new Map(); // Track streaming sessions
        this.analytics = {
            qualityRequests: {},
            bandwidthMeasurements: [],
            rebufferEvents: 0,
            qualitySwitches: 0
        };
    }

    // Create streaming session with initial quality detection
    createSession(sessionId, clientInfo = {}) {
        const initialQuality = this.selectInitialQuality(clientInfo);
        
        const session = {
            id: sessionId,
            startTime: Date.now(),
            currentQuality: initialQuality,
            clientInfo,
            bandwidthHistory: [],
            rebufferHistory: [],
            qualityHistory: [{ 
                quality: initialQuality, 
                timestamp: Date.now() 
            }],
            segmentsRequested: 0,
            bytesDownloaded: 0,
            totalPlayTime: 0
        };
        
        this.sessions.set(sessionId, session);
        return session;
    }

    // Select initial quality based on client information
    selectInitialQuality(clientInfo) {
        const { userAgent, connection, screenResolution } = clientInfo;
        
        // Default to medium quality
        let selectedQuality = '720p';
        
        // Detect mobile devices
        if (this.isMobileDevice(userAgent)) {
            selectedQuality = '480p';
        }
        
        // Consider connection type
        if (connection) {
            switch (connection.effectiveType) {
                case 'slow-2g':
                case '2g':
                    selectedQuality = '240p';
                    break;
                case '3g':
                    selectedQuality = '360p';
                    break;
                case '4g':
                    selectedQuality = '720p';
                    break;
            }
        }
        
        // Consider screen resolution
        if (screenResolution) {
            const { width, height } = screenResolution;
            const maxDimension = Math.max(width, height);
            
            if (maxDimension <= 480) {
                selectedQuality = '360p';
            } else if (maxDimension <= 720) {
                selectedQuality = '480p';
            } else if (maxDimension <= 1080) {
                selectedQuality = '720p';
            } else if (maxDimension <= 1440) {
                selectedQuality = '1080p';
            } else {
                selectedQuality = '4k';
            }
        }
        
        return selectedQuality;
    }

    // Handle segment request and quality adaptation
    async handleSegmentRequest(sessionId, contentId, currentQuality, segmentId, requestMetrics) {
        const session = this.sessions.get(sessionId);
        if (!session) {
            return { quality: currentQuality, switchReason: null };
        }

        // Update session metrics
        this.updateSessionMetrics(session, requestMetrics);
        
        // Calculate bandwidth
        const bandwidth = this.calculateBandwidth(session);
        
        // Determine if quality switch is needed
        const qualityDecision = this.makeQualityDecision(session, bandwidth);
        
        // Log analytics
        this.recordAnalytics(session, qualityDecision);
        
        return qualityDecision;
    }

    updateSessionMetrics(session, metrics) {
        if (metrics.downloadTime && metrics.segmentSize) {
            const bandwidth = (metrics.segmentSize * 8) / (metrics.downloadTime / 1000); // bps
            session.bandwidthHistory.push({
                bandwidth,
                timestamp: Date.now(),
                segmentSize: metrics.segmentSize,
                downloadTime: metrics.downloadTime
            });
            
            // Keep only recent measurements
            const cutoff = Date.now() - (30 * 1000); // Last 30 seconds
            session.bandwidthHistory = session.bandwidthHistory.filter(
                entry => entry.timestamp > cutoff
            );
        }
        
        if (metrics.rebuffer) {
            session.rebufferHistory.push({
                timestamp: Date.now(),
                duration: metrics.rebuffer.duration
            });
            this.analytics.rebufferEvents++;
        }
        
        session.segmentsRequested++;
        session.bytesDownloaded += metrics.segmentSize || 0;
    }

    calculateBandwidth(session) {
        if (session.bandwidthHistory.length === 0) {
            return null;
        }
        
        // Use exponentially weighted moving average
        let weightedSum = 0;
        let weightSum = 0;
        
        const now = Date.now();
        session.bandwidthHistory.forEach((entry, index) => {
            const age = (now - entry.timestamp) / 1000; // seconds
            const weight = Math.exp(-age / 10); // Decay over 10 seconds
            
            weightedSum += entry.bandwidth * weight;
            weightSum += weight;
        });
        
        return weightSum > 0 ? weightedSum / weightSum : null;
    }

    makeQualityDecision(session, estimatedBandwidth) {
        const currentQuality = session.currentQuality;
        const currentLevel = this.qualityLevels.find(q => q.name === currentQuality);
        
        if (!currentLevel || !estimatedBandwidth) {
            return { quality: currentQuality, switchReason: null };
        }
        
        // Quality switching logic with hysteresis
        const switchUpThreshold = 1.5; // 150% of required bandwidth
        const switchDownThreshold = 0.8; // 80% of required bandwidth
        
        // Check for quality upgrade
        for (let i = this.qualityLevels.length - 1; i >= 0; i--) {
            const level = this.qualityLevels[i];
            
            if (level.bandwidth <= estimatedBandwidth * switchUpThreshold && 
                level.bandwidth > currentLevel.bandwidth) {
                
                // Check recent rebuffer events
                const recentRebuffers = this.getRecentRebuffers(session, 10000); // Last 10 seconds
                if (recentRebuffers.length === 0) {
                    return this.executeQualitySwitch(session, level.name, 'bandwidth_increase');
                }
            }
        }
        
        // Check for quality downgrade
        if (currentLevel.bandwidth > estimatedBandwidth * switchDownThreshold) {
            for (let i = 0; i < this.qualityLevels.length; i++) {
                const level = this.qualityLevels[i];
                
                if (level.bandwidth <= estimatedBandwidth * switchDownThreshold && 
                    level.bandwidth < currentLevel.bandwidth) {
                    
                    return this.executeQualitySwitch(session, level.name, 'bandwidth_decrease');
                }
            }
        }
        
        // Check for rebuffer-based downgrade
        const recentRebuffers = this.getRecentRebuffers(session, 5000); // Last 5 seconds
        if (recentRebuffers.length > 0) {
            const lowerQuality = this.getLowerQuality(currentQuality);
            if (lowerQuality) {
                return this.executeQualitySwitch(session, lowerQuality, 'rebuffer_avoidance');
            }
        }
        
        return { quality: currentQuality, switchReason: null };
    }

    executeQualitySwitch(session, newQuality, reason) {
        const previousQuality = session.currentQuality;
        session.currentQuality = newQuality;
        
        session.qualityHistory.push({
            quality: newQuality,
            timestamp: Date.now(),
            reason
        });
        
        this.analytics.qualitySwitches++;
        
        return {
            quality: newQuality,
            switchReason: reason,
            previousQuality
        };
    }

    getLowerQuality(currentQuality) {
        const currentIndex = this.qualityLevels.findIndex(q => q.name === currentQuality);
        if (currentIndex > 0) {
            return this.qualityLevels[currentIndex - 1].name;
        }
        return null;
    }

    getHigherQuality(currentQuality) {
        const currentIndex = this.qualityLevels.findIndex(q => q.name === currentQuality);
        if (currentIndex < this.qualityLevels.length - 1) {
            return this.qualityLevels[currentIndex + 1].name;
        }
        return null;
    }

    getRecentRebuffers(session, timeWindow) {
        const cutoff = Date.now() - timeWindow;
        return session.rebufferHistory.filter(rebuffer => rebuffer.timestamp > cutoff);
    }

    // Generate optimized playlist with quality ladder
    generateAdaptivePlaylist(contentId, availableQualities, sessionId) {
        const session = this.sessions.get(sessionId);
        const recommendedQuality = session ? session.currentQuality : '720p';
        
        // Sort qualities by bandwidth
        const sortedQualities = availableQualities
            .map(quality => this.qualityLevels.find(q => q.name === quality))
            .filter(Boolean)
            .sort((a, b) => a.bandwidth - b.bandwidth);
        
        let playlist = '#EXTM3U\n';
        playlist += '#EXT-X-VERSION:6\n';
        
        // Add variant streams
        sortedQualities.forEach(quality => {
            playlist += `#EXT-X-STREAM-INF:BANDWIDTH=${quality.bandwidth},RESOLUTION=${quality.resolution}`;
            
            if (quality.name === recommendedQuality) {
                playlist += ',DEFAULT=YES';
            }
            
            playlist += `\n${quality.name}/playlist.m3u8\n`;
        });
        
        return playlist;
    }

    // Preload segments for smooth playback
    generatePreloadHints(sessionId, currentSegment) {
        const session = this.sessions.get(sessionId);
        if (!session) return [];
        
        const hints = [];
        const currentQuality = session.currentQuality;
        
        // Preload next 2-3 segments of current quality
        for (let i = 1; i <= 3; i++) {
            hints.push({
                type: 'segment',
                quality: currentQuality,
                segment: currentSegment + i,
                priority: i === 1 ? 'high' : 'medium'
            });
        }
        
        // Preload first segment of lower quality (for quick downgrade)
        const lowerQuality = this.getLowerQuality(currentQuality);
        if (lowerQuality) {
            hints.push({
                type: 'segment',
                quality: lowerQuality,
                segment: currentSegment + 1,
                priority: 'low'
            });
        }
        
        return hints;
    }

    // Handle seeking with segment-level precision
    handleSeekRequest(sessionId, seekTime, segmentDuration = 6) {
        const targetSegment = Math.floor(seekTime / segmentDuration);
        const session = this.sessions.get(sessionId);
        
        if (session) {
            // Reset bandwidth estimation on seek
            session.bandwidthHistory = [];
            
            // Temporarily use lower quality after seek to avoid rebuffering
            const currentQuality = session.currentQuality;
            const lowerQuality = this.getLowerQuality(currentQuality);
            
            if (lowerQuality) {
                session.currentQuality = lowerQuality;
                session.qualityHistory.push({
                    quality: lowerQuality,
                    timestamp: Date.now(),
                    reason: 'seek_recovery'
                });
            }
        }
        
        return {
            targetSegment,
            quality: session ? session.currentQuality : '720p',
            preloadSegments: this.generatePreloadHints(sessionId, targetSegment)
        };
    }

    // Get session analytics
    getSessionAnalytics(sessionId) {
        const session = this.sessions.get(sessionId);
        if (!session) return null;
        
        const sessionDuration = Date.now() - session.startTime;
        const averageBandwidth = session.bandwidthHistory.length > 0 
            ? session.bandwidthHistory.reduce((sum, entry) => sum + entry.bandwidth, 0) / session.bandwidthHistory.length
            : 0;
        
        return {
            sessionId,
            duration: sessionDuration,
            segmentsRequested: session.segmentsRequested,
            bytesDownloaded: session.bytesDownloaded,
            averageBandwidth,
            qualitySwitches: session.qualityHistory.length - 1,
            rebufferEvents: session.rebufferHistory.length,
            currentQuality: session.currentQuality,
            qualityDistribution: this.calculateQualityDistribution(session)
        };
    }

    calculateQualityDistribution(session) {
        const distribution = {};
        let lastTimestamp = session.startTime;
        
        session.qualityHistory.forEach((entry, index) => {
            const duration = index < session.qualityHistory.length - 1 
                ? session.qualityHistory[index + 1].timestamp - entry.timestamp
                : Date.now() - entry.timestamp;
            
            distribution[entry.quality] = (distribution[entry.quality] || 0) + duration;
        });
        
        return distribution;
    }

    // Global analytics
    getGlobalAnalytics() {
        return {
            ...this.analytics,
            activeSessions: this.sessions.size,
            qualityLevels: this.qualityLevels,
            averageBandwidth: this.analytics.bandwidthMeasurements.length > 0
                ? this.analytics.bandwidthMeasurements.reduce((a, b) => a + b, 0) / this.analytics.bandwidthMeasurements.length
                : 0
        };
    }

    recordAnalytics(session, qualityDecision) {
        // Record quality requests
        const quality = qualityDecision.quality;
        this.analytics.qualityRequests[quality] = (this.analytics.qualityRequests[quality] || 0) + 1;
        
        // Record bandwidth measurements
        if (session.bandwidthHistory.length > 0) {
            const latest = session.bandwidthHistory[session.bandwidthHistory.length - 1];
            this.analytics.bandwidthMeasurements.push(latest.bandwidth);
            
            // Keep only recent measurements
            if (this.analytics.bandwidthMeasurements.length > 1000) {
                this.analytics.bandwidthMeasurements = this.analytics.bandwidthMeasurements.slice(-500);
            }
        }
    }

    isMobileDevice(userAgent) {
        if (!userAgent) return false;
        
        const mobileKeywords = [
            'Mobile', 'Android', 'iPhone', 'iPad', 'Windows Phone',
            'BlackBerry', 'Opera Mini', 'Opera Mobi'
        ];
        
        return mobileKeywords.some(keyword => 
            userAgent.includes(keyword)
        );
    }

    // Cleanup old sessions
    cleanupSessions(maxAge = 3600000) { // 1 hour default
        const cutoff = Date.now() - maxAge;
        
        for (const [sessionId, session] of this.sessions.entries()) {
            if (session.startTime < cutoff) {
                this.sessions.delete(sessionId);
            }
        }
    }

    // Close session
    closeSession(sessionId) {
        return this.sessions.delete(sessionId);
    }
}

module.exports = AdaptiveBitrateStreaming;
