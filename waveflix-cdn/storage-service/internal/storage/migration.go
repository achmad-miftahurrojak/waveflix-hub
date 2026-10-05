package storage

import (
"context"
"log"
"sync"
"time"
)

type TierMigrator struct {
storage        *TieredStorage
rules          []MigrationRule
running        bool
stopChan       chan struct{}
mutex          sync.RWMutex
}

func NewTierMigrator(storage *TieredStorage, rules []MigrationRule) *TierMigrator {
return &TierMigrator{
storage:  storage,
rules:    rules,
stopChan: make(chan struct{}),
}
}

func (tm *TierMigrator) Start(ctx context.Context) {
tm.mutex.Lock()
if tm.running {
tm.mutex.Unlock()
return
}
tm.running = true
tm.mutex.Unlock()

log.Println("[Migration] Starting tier migration service")

ticker := time.NewTicker(30 * time.Minute) // Run every 30 minutes
defer ticker.Stop()

// Initial migration check
go tm.runMigrationCycle(ctx)

for {
select {
case <-ctx.Done():
tm.stop()
return
case <-tm.stopChan:
return
case <-ticker.C:
go tm.runMigrationCycle(ctx)
}
}
}

func (tm *TierMigrator) Stop() {
tm.stop()
}

func (tm *TierMigrator) stop() {
tm.mutex.Lock()
defer tm.mutex.Unlock()

if tm.running {
tm.running = false
close(tm.stopChan)
log.Println("[Migration] Stopped tier migration service")
}
}

func (tm *TierMigrator) runMigrationCycle(ctx context.Context) {
log.Println("[Migration] Starting migration cycle")

startTime := time.Now()
migrated := 0

// Get all files from metadata store
files := tm.storage.metadata.List("")

for _, fileInfo := range files {
if ctx.Err() != nil {
break
}

targetTier := tm.evaluateMigration(fileInfo)
if targetTier != fileInfo.Tier {
if err := tm.storage.Move(ctx, fileInfo.Key, targetTier); err != nil {
log.Printf("[Migration] Failed to migrate %s to %s: %v", fileInfo.Key, targetTier, err)
} else {
migrated++
log.Printf("[Migration] Migrated %s from %s to %s", fileInfo.Key, fileInfo.Tier, targetTier)
}
}
}

duration := time.Since(startTime)
log.Printf("[Migration] Completed migration cycle: %d files migrated in %v", migrated, duration)
}

func (tm *TierMigrator) evaluateMigration(fileInfo *FileInfo) StorageTier {
currentTier := fileInfo.Tier

for _, rule := range tm.rules {
if !rule.Enabled || rule.FromTier != currentTier {
continue
}

if tm.shouldMigrate(fileInfo, rule) {
return rule.ToTier
}
}

return currentTier
}

func (tm *TierMigrator) shouldMigrate(fileInfo *FileInfo, rule MigrationRule) bool {
now := time.Now()

// Check age threshold
if rule.AgeThreshold > 0 {
age := now.Sub(fileInfo.LastAccessed)
if age < rule.AgeThreshold {
return false
}
}

// Check access threshold
if rule.AccessThreshold > 0 {
if fileInfo.AccessCount > rule.AccessThreshold {
return false // Too frequently accessed
}
}

// Check size threshold
if rule.SizeThreshold > 0 {
if fileInfo.Size < rule.SizeThreshold {
return false // Too small
}
}

return true
}

func (tm *TierMigrator) GetStats() MigrationStats {
tm.mutex.RLock()
defer tm.mutex.RUnlock()

return MigrationStats{
Running: tm.running,
Rules:   len(tm.rules),
}
}

type MigrationStats struct {
Running         bool `json:"running"`
Rules           int  `json:"rules"`
LastRun         time.Time `json:"last_run"`
FilesMigrated   int64 `json:"files_migrated"`
BytesMigrated   int64 `json:"bytes_migrated"`
AverageDuration time.Duration `json:"average_duration"`
}

type ReplicationManager struct {
storage *TieredStorage
config  ReplicationConfig
mutex   sync.RWMutex
}

func NewReplicationManager(storage *TieredStorage, config ReplicationConfig) *ReplicationManager {
return &ReplicationManager{
storage: storage,
config:  config,
}
}

func (rm *ReplicationManager) EnsureReplication(ctx context.Context, key string) error {
if !rm.config.Enabled {
return nil
}

fileInfo, err := rm.storage.metadata.Get(key)
if err != nil {
return err
}

// Count existing replicas
replicaCount := 1 // Original file

// Check for existing replicas
for tier := range rm.storage.providers {
if tier == fileInfo.Tier {
continue
}

replicaKey := rm.getReplicaKey(key, tier)
if exists, _, _ := rm.storage.providers[tier].Exists(ctx, replicaKey); exists {
replicaCount++
}
}

// Create additional replicas if needed
if replicaCount < rm.config.MinReplicas {
needed := rm.config.MinReplicas - replicaCount
created := 0

for tier := range rm.storage.providers {
if created >= needed || tier == fileInfo.Tier {
continue
}

replicaKey := rm.getReplicaKey(key, tier)
if exists, _, _ := rm.storage.providers[tier].Exists(ctx, replicaKey); exists {
continue
}

if err := rm.createReplica(ctx, key, tier); err != nil {
log.Printf("[Replication] Failed to create replica %s: %v", replicaKey, err)
continue
}

created++
log.Printf("[Replication] Created replica %s", replicaKey)
}
}

return nil
}

func (rm *ReplicationManager) createReplica(ctx context.Context, key string, targetTier StorageTier) error {
// Retrieve original file
reader, err := rm.storage.Retrieve(ctx, key)
if err != nil {
return err
}
defer reader.Close()

// Store replica
replicaKey := rm.getReplicaKey(key, targetTier)
provider := rm.storage.providers[targetTier]

return provider.Store(ctx, replicaKey, reader, targetTier)
}

func (rm *ReplicationManager) getReplicaKey(key string, tier StorageTier) string {
return key + ".replica." + tier.String()
}

func (rm *ReplicationManager) CleanupReplicas(ctx context.Context, key string) error {
for tier := range rm.storage.providers {
replicaKey := rm.getReplicaKey(key, tier)
if err := rm.storage.providers[tier].Delete(ctx, replicaKey); err != nil {
log.Printf("[Replication] Failed to cleanup replica %s: %v", replicaKey, err)
}
}
return nil
}

type DeduplicationManager struct {
storage     *TieredStorage
checksums   map[string][]string // checksum -> list of keys
mutex       sync.RWMutex
}

func NewDeduplicationManager(storage *TieredStorage) *DeduplicationManager {
return &DeduplicationManager{
storage:   storage,
checksums: make(map[string][]string),
}
}

func (dm *DeduplicationManager) RegisterFile(key, checksum string) {
dm.mutex.Lock()
defer dm.mutex.Unlock()

dm.checksums[checksum] = append(dm.checksums[checksum], key)
}

func (dm *DeduplicationManager) FindDuplicates(checksum string) []string {
dm.mutex.RLock()
defer dm.mutex.RUnlock()

if keys, exists := dm.checksums[checksum]; exists {
result := make([]string, len(keys))
copy(result, keys)
return result
}

return nil
}

func (dm *DeduplicationManager) RemoveFile(key, checksum string) {
dm.mutex.Lock()
defer dm.mutex.Unlock()

if keys, exists := dm.checksums[checksum]; exists {
for i, k := range keys {
if k == key {
dm.checksums[checksum] = append(keys[:i], keys[i+1:]...)
break
}
}

if len(dm.checksums[checksum]) == 0 {
delete(dm.checksums, checksum)
}
}
}

func (dm *DeduplicationManager) GetStats() DeduplicationStats {
dm.mutex.RLock()
defer dm.mutex.RUnlock()

totalFiles := 0
duplicateGroups := 0
duplicateFiles := 0

for _, keys := range dm.checksums {
totalFiles += len(keys)
if len(keys) > 1 {
duplicateGroups++
duplicateFiles += len(keys) - 1 // Subtract 1 for the original
}
}

return DeduplicationStats{
TotalFiles:      totalFiles,
DuplicateGroups: duplicateGroups,
DuplicateFiles:  duplicateFiles,
SpaceSaved:      0, // Would need file sizes to calculate
}
}

type DeduplicationStats struct {
TotalFiles      int   `json:"total_files"`
DuplicateGroups int   `json:"duplicate_groups"`
DuplicateFiles  int   `json:"duplicate_files"`
SpaceSaved      int64 `json:"space_saved"`
}
