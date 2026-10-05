package main

import (
"context"
"log"
"os"
"strconv"

"github.com/waveflix-hub/processing-service/internal/queue"
"github.com/waveflix-hub/processing-service/internal/server"
"github.com/waveflix-hub/processing-service/internal/workers"
"github.com/go-redis/redis/v8"
)

func main() {
log.Println("[Processing] WaveFlix Processing Service starting...")

// Configuration from environment
port := getEnv("PORT", "8081")
redisURL := getEnv("REDIS_URL", "localhost:6379")
ffmpegPath := getEnv("FFMPEG_PATH", "ffmpeg")
workDir := getEnv("WORK_DIR", "/tmp/processing")
maxConcurrent := getEnvInt("MAX_CONCURRENT_JOBS", 5)

// Create work directory
if err := os.MkdirAll(workDir, 0755); err != nil {
log.Fatalf("[Processing] Failed to create work directory: %v", err)
}

// Setup Redis client
var redisClient *redis.Client
if redisURL != "" {
redisClient = redis.NewClient(&redis.Options{
Addr: redisURL,
})

ctx := context.Background()
if err := redisClient.Ping(ctx).Err(); err != nil {
log.Printf("[Processing] Redis connection failed: %v", err)
redisClient = nil
} else {
log.Println("[Processing] Connected to Redis")
}
}

// Create queue manager
queueManager := queue.NewQueueManager(redisClient, maxConcurrent)

// Register workers
transcodingWorker := workers.NewTranscodingWorker("transcoding-1", ffmpegPath, workDir, 2)
queueManager.RegisterWorker(transcodingWorker)

thumbnailWorker := workers.NewThumbnailWorker("thumbnail-1", ffmpegPath, workDir)
queueManager.RegisterWorker(thumbnailWorker)

// Start queue processing in background
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go queueManager.Start(ctx)

// Start HTTP server
processingServer := server.NewProcessingServer(queueManager, port)

log.Printf("[Processing] Server ready on port %s", port)
if err := processingServer.Start(); err != nil {
log.Fatalf("[Processing] Server failed to start: %v", err)
}
}

func getEnv(key, defaultValue string) string {
if value := os.Getenv(key); value != "" {
return value
}
return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
if value := os.Getenv(key); value != "" {
if intValue, err := strconv.Atoi(value); err == nil {
return intValue
}
}
return defaultValue
}
