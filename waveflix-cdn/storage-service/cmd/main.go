package main

import (
"context"
"log"
"os"
"time"

"github.com/waveflix-hub/storage-service/internal/server"
"github.com/waveflix-hub/storage-service/internal/storage"
)

func main() {
log.Println("[Storage] WaveFlix Storage Service starting...")

// Configuration from environment
port := getEnv("PORT", "8082")
hotTierPath := getEnv("HOT_TIER_PATH", "/storage/hot")
warmTierPath := getEnv("WARM_TIER_PATH", "/storage/warm")
coldTierPath := getEnv("COLD_TIER_PATH", "/storage/cold")

// Create storage directories
createDirectory(hotTierPath)
createDirectory(warmTierPath)
createDirectory(coldTierPath)

// Configure tiered storage
config := &storage.TieredConfig{
HotTierPath:  hotTierPath,
WarmTierPath: warmTierPath,
ColdTierPath: coldTierPath,
HotTierTTL:   24 * time.Hour,   // Hot tier TTL: 24 hours
WarmTierTTL:  7 * 24 * time.Hour, // Warm tier TTL: 7 days
MigrationRules: []storage.MigrationRule{
{
FromTier:        storage.HotTier,
ToTier:          storage.WarmTier,
AccessThreshold: 5,  // Less than 5 accesses
AgeThreshold:    24 * time.Hour, // Older than 24 hours
Enabled:         true,
},
{
FromTier:        storage.WarmTier,
ToTier:          storage.ColdTier,
AccessThreshold: 2,  // Less than 2 accesses
AgeThreshold:    7 * 24 * time.Hour, // Older than 7 days
Enabled:         true,
},
{
FromTier:        storage.ColdTier,
ToTier:          storage.HotTier,
AccessThreshold: 50, // More than 50 accesses (promote hot content)
AgeThreshold:    0,  // Any age
Enabled:         true,
},
},
Replication: storage.ReplicationConfig{
Enabled:     true,
MinReplicas: 2,
MaxReplicas: 3,
CrossTier:   true,
},
}

// Initialize tiered storage
tieredStorage := storage.NewTieredStorage(config)

// Start migration service
ctx := context.Background()
go tieredStorage.StartMigration(ctx)

// Start HTTP server
storageServer := server.NewStorageServer(tieredStorage, port)

log.Printf("[Storage] Server ready on port %s", port)
log.Printf("[Storage] Tiers configured: Hot=%s, Warm=%s, Cold=%s", 
hotTierPath, warmTierPath, coldTierPath)

if err := storageServer.Start(); err != nil {
log.Fatalf("[Storage] Server failed to start: %v", err)
}
}

func getEnv(key, defaultValue string) string {
if value := os.Getenv(key); value != "" {
return value
}
return defaultValue
}

func createDirectory(path string) {
if err := os.MkdirAll(path, 0755); err != nil {
log.Printf("[Storage] Warning: failed to create directory %s: %v", path, err)
} else {
log.Printf("[Storage] Created storage directory: %s", path)
}
}
