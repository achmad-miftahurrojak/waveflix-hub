package storage

import (
"context"
"crypto/md5"
"encoding/hex"
"fmt"
"io"
"log"
"os"
"path/filepath"
"sync"
"time"
)

type StorageTier int

const (
HotTier StorageTier = iota
WarmTier
ColdTier
)

func (st StorageTier) String() string {
switch st {
case HotTier:
return "hot"
case WarmTier:
return "warm"
case ColdTier:
return "cold"
default:
return "unknown"
}
}

type StorageProvider interface {
Store(ctx context.Context, key string, data io.Reader, tier StorageTier) error
Retrieve(ctx context.Context, key string) (io.ReadCloser, StorageTier, error)
Delete(ctx context.Context, key string) error
Exists(ctx context.Context, key string) (bool, StorageTier, error)
List(ctx context.Context, prefix string) ([]FileInfo, error)
Move(ctx context.Context, key string, fromTier, toTier StorageTier) error
GetTierInfo(tier StorageTier) TierInfo
}

type FileInfo struct {
Key          string      `json:"key"`
Size         int64       `json:"size"`
Tier         StorageTier `json:"tier"`
LastAccessed time.Time   `json:"last_accessed"`
LastModified time.Time   `json:"last_modified"`
ContentType  string      `json:"content_type"`
ETag         string      `json:"etag"`
AccessCount  int64       `json:"access_count"`
}

type TierInfo struct {
Name         string        `json:"name"`
Capacity     int64         `json:"capacity"`
Used         int64         `json:"used"`
Available    int64         `json:"available"`
AccessCost   float64       `json:"access_cost"`
StorageCost  float64       `json:"storage_cost"`
Latency      time.Duration `json:"latency"`
Throughput   int64         `json:"throughput"`
}

type TieredStorage struct {
providers map[StorageTier]StorageProvider
metadata  *MetadataStore
analytics *AccessAnalytics
migrator  *TierMigrator
config    *TieredConfig
mutex     sync.RWMutex
}

type TieredConfig struct {
HotTierPath    string        `json:"hot_tier_path"`
WarmTierPath   string        `json:"warm_tier_path"`
ColdTierPath   string        `json:"cold_tier_path"`
HotTierTTL     time.Duration `json:"hot_tier_ttl"`
WarmTierTTL    time.Duration `json:"warm_tier_ttl"`
MigrationRules []MigrationRule `json:"migration_rules"`
Replication    ReplicationConfig `json:"replication"`
}

type MigrationRule struct {
FromTier      StorageTier   `json:"from_tier"`
ToTier        StorageTier   `json:"to_tier"`
AccessThreshold int64       `json:"access_threshold"`
AgeThreshold    time.Duration `json:"age_threshold"`
SizeThreshold   int64       `json:"size_threshold"`
Enabled       bool          `json:"enabled"`
}

type ReplicationConfig struct {
Enabled     bool `json:"enabled"`
MinReplicas int  `json:"min_replicas"`
MaxReplicas int  `json:"max_replicas"`
CrossTier   bool `json:"cross_tier"`
}

func NewTieredStorage(config *TieredConfig) *TieredStorage {
ts := &TieredStorage{
providers: make(map[StorageTier]StorageProvider),
config:    config,
metadata:  NewMetadataStore(),
analytics: NewAccessAnalytics(),
}

// Initialize providers for each tier
ts.providers[HotTier] = NewLocalProvider(config.HotTierPath, HotTier)
ts.providers[WarmTier] = NewLocalProvider(config.WarmTierPath, WarmTier)
ts.providers[ColdTier] = NewLocalProvider(config.ColdTierPath, ColdTier)

// Initialize migrator
ts.migrator = NewTierMigrator(ts, config.MigrationRules)

return ts
}

func (ts *TieredStorage) Store(ctx context.Context, key string, data io.Reader, contentType string) error {
// Default to hot tier for new content
tier := HotTier

// Calculate content hash for deduplication
hasher := md5.New()
teeReader := io.TeeReader(data, hasher)

// Store in appropriate tier
provider := ts.providers[tier]
if err := provider.Store(ctx, key, teeReader, tier); err != nil {
return fmt.Errorf("failed to store in %s tier: %w", tier, err)
}

// Calculate final hash
etag := hex.EncodeToString(hasher.Sum(nil))

// Get file size
var size int64
if seeker, ok := data.(io.Seeker); ok {
if current, err := seeker.Seek(0, io.SeekCurrent); err == nil {
if end, err := seeker.Seek(0, io.SeekEnd); err == nil {
size = end - current
seeker.Seek(current, io.SeekStart)
}
}
}

// Update metadata
fileInfo := &FileInfo{
Key:          key,
Size:         size,
Tier:         tier,
LastAccessed: time.Now(),
LastModified: time.Now(),
ContentType:  contentType,
ETag:         etag,
AccessCount:  1,
}

ts.metadata.Set(key, fileInfo)

// Handle replication if enabled
if ts.config.Replication.Enabled {
go ts.handleReplication(ctx, key, fileInfo)
}

log.Printf("[Storage] Stored %s in %s tier (size: %d bytes)", key, tier, size)
return nil
}

func (ts *TieredStorage) Retrieve(ctx context.Context, key string) (io.ReadCloser, error) {
ts.mutex.RLock()
defer ts.mutex.RUnlock()

// Get metadata to determine current tier
fileInfo, err := ts.metadata.Get(key)
if err != nil {
return nil, fmt.Errorf("file not found: %s", key)
}

// Update access analytics
ts.analytics.RecordAccess(key, fileInfo.Tier)

// Try to retrieve from current tier
provider := ts.providers[fileInfo.Tier]
reader, actualTier, err := provider.Retrieve(ctx, key)
if err == nil {
// Update metadata with access info
fileInfo.LastAccessed = time.Now()
fileInfo.AccessCount++
ts.metadata.Set(key, fileInfo)

// Check if file should be promoted to hotter tier
go ts.checkPromotion(key, fileInfo)

return reader, nil
}

// If not found in expected tier, search other tiers
for tier, provider := range ts.providers {
if tier == fileInfo.Tier {
continue
}

if reader, actualTier, err = provider.Retrieve(ctx, key); err == nil {
// Update metadata with correct tier
fileInfo.Tier = actualTier
fileInfo.LastAccessed = time.Now()
fileInfo.AccessCount++
ts.metadata.Set(key, fileInfo)

log.Printf("[Storage] Found %s in unexpected tier: %s", key, actualTier)
return reader, nil
}
}

return nil, fmt.Errorf("file not found in any tier: %s", key)
}

func (ts *TieredStorage) Delete(ctx context.Context, key string) error {
ts.mutex.Lock()
defer ts.mutex.Unlock()

var errors []error
deleted := false

// Try to delete from all tiers
for tier, provider := range ts.providers {
if err := provider.Delete(ctx, key); err == nil {
deleted = true
log.Printf("[Storage] Deleted %s from %s tier", key, tier)
} else if !os.IsNotExist(err) {
errors = append(errors, fmt.Errorf("%s tier: %w", tier, err))
}
}

// Remove from metadata
ts.metadata.Delete(key)
ts.analytics.RemoveFile(key)

if !deleted && len(errors) > 0 {
return fmt.Errorf("failed to delete from any tier: %v", errors)
}

return nil
}

func (ts *TieredStorage) Move(ctx context.Context, key string, toTier StorageTier) error {
ts.mutex.Lock()
defer ts.mutex.Unlock()

fileInfo, err := ts.metadata.Get(key)
if err != nil {
return fmt.Errorf("file not found: %s", key)
}

if fileInfo.Tier == toTier {
return nil // Already in target tier
}

fromTier := fileInfo.Tier
fromProvider := ts.providers[fromTier]
toProvider := ts.providers[toTier]

// Retrieve from source tier
reader, _, err := fromProvider.Retrieve(ctx, key)
if err != nil {
return fmt.Errorf("failed to retrieve from %s tier: %w", fromTier, err)
}
defer reader.Close()

// Store in target tier
if err := toProvider.Store(ctx, key, reader, toTier); err != nil {
return fmt.Errorf("failed to store in %s tier: %w", toTier, err)
}

// Delete from source tier
if err := fromProvider.Delete(ctx, key); err != nil {
log.Printf("[Storage] Warning: failed to delete %s from %s tier after move: %v", key, fromTier, err)
}

// Update metadata
fileInfo.Tier = toTier
fileInfo.LastModified = time.Now()
ts.metadata.Set(key, fileInfo)

log.Printf("[Storage] Moved %s from %s to %s tier", key, fromTier, toTier)
return nil
}

func (ts *TieredStorage) GetStats() map[string]interface{} {
ts.mutex.RLock()
defer ts.mutex.RUnlock()

stats := make(map[string]interface{})

for tier, provider := range ts.providers {
tierInfo := provider.GetTierInfo(tier)
stats[tier.String()] = tierInfo
}

stats["analytics"] = ts.analytics.GetStats()
stats["metadata_count"] = ts.metadata.Count()

return stats
}

func (ts *TieredStorage) StartMigration(ctx context.Context) {
go ts.migrator.Start(ctx)
}

func (ts *TieredStorage) checkPromotion(key string, fileInfo *FileInfo) {
// Check if file should be promoted to a hotter tier based on access patterns
if fileInfo.Tier == HotTier {
return // Already in hottest tier
}

accessFreq := ts.analytics.GetAccessFrequency(key)

// Promote to hot tier if accessed frequently
if accessFreq > 10 && fileInfo.Tier != HotTier {
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := ts.Move(ctx, key, HotTier); err != nil {
log.Printf("[Storage] Failed to promote %s to hot tier: %v", key, err)
}
}
}

func (ts *TieredStorage) handleReplication(ctx context.Context, key string, fileInfo *FileInfo) {
if ts.config.Replication.MinReplicas <= 1 {
return
}

// Create replicas in different tiers if cross-tier replication is enabled
replicaCount := 1 // Original file

if ts.config.Replication.CrossTier {
for tier, provider := range ts.providers {
if tier == fileInfo.Tier || replicaCount >= ts.config.Replication.MaxReplicas {
continue
}

replicaKey := fmt.Sprintf("%s.replica.%s", key, tier.String())

// Retrieve original file
reader, err := ts.Retrieve(ctx, key)
if err != nil {
continue
}

// Store replica
if err := provider.Store(ctx, replicaKey, reader, tier); err != nil {
log.Printf("[Storage] Failed to create replica %s in %s tier: %v", replicaKey, tier, err)
reader.Close()
continue
}

reader.Close()
replicaCount++

log.Printf("[Storage] Created replica %s in %s tier", replicaKey, tier)
}
}
}

type LocalProvider struct {
basePath string
tier     StorageTier
mutex    sync.RWMutex
}

func NewLocalProvider(basePath string, tier StorageTier) *LocalProvider {
if err := os.MkdirAll(basePath, 0755); err != nil {
log.Printf("[Storage] Warning: failed to create directory %s: %v", basePath, err)
}

return &LocalProvider{
basePath: basePath,
tier:     tier,
}
}

func (lp *LocalProvider) Store(ctx context.Context, key string, data io.Reader, tier StorageTier) error {
filePath := filepath.Join(lp.basePath, key)

if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
return fmt.Errorf("failed to create directory: %w", err)
}

file, err := os.Create(filePath)
if err != nil {
return fmt.Errorf("failed to create file: %w", err)
}
defer file.Close()

if _, err := io.Copy(file, data); err != nil {
os.Remove(filePath)
return fmt.Errorf("failed to write file: %w", err)
}

return nil
}

func (lp *LocalProvider) Retrieve(ctx context.Context, key string) (io.ReadCloser, StorageTier, error) {
filePath := filepath.Join(lp.basePath, key)

file, err := os.Open(filePath)
if err != nil {
return nil, lp.tier, err
}

return file, lp.tier, nil
}

func (lp *LocalProvider) Delete(ctx context.Context, key string) error {
filePath := filepath.Join(lp.basePath, key)
return os.Remove(filePath)
}

func (lp *LocalProvider) Exists(ctx context.Context, key string) (bool, StorageTier, error) {
filePath := filepath.Join(lp.basePath, key)

if _, err := os.Stat(filePath); err != nil {
if os.IsNotExist(err) {
return false, lp.tier, nil
}
return false, lp.tier, err
}

return true, lp.tier, nil
}

func (lp *LocalProvider) List(ctx context.Context, prefix string) ([]FileInfo, error) {
var files []FileInfo
prefixPath := filepath.Join(lp.basePath, prefix)

err := filepath.Walk(prefixPath, func(path string, info os.FileInfo, err error) error {
if err != nil {
return err
}

if info.IsDir() {
return nil
}

relPath, err := filepath.Rel(lp.basePath, path)
if err != nil {
return err
}

fileInfo := FileInfo{
Key:          filepath.ToSlash(relPath),
Size:         info.Size(),
Tier:         lp.tier,
LastModified: info.ModTime(),
LastAccessed: info.ModTime(), // Use modtime as approximation
}

files = append(files, fileInfo)
return nil
})

return files, err
}

func (lp *LocalProvider) Move(ctx context.Context, key string, fromTier, toTier StorageTier) error {
return fmt.Errorf("move operation not supported within single provider")
}

func (lp *LocalProvider) GetTierInfo(tier StorageTier) TierInfo {
var used, available int64

if _, err := os.Stat(lp.basePath); err == nil {
// This is a simplified calculation
// In production, you'd use filesystem-specific calls
available = 1024 * 1024 * 1024 * 100 // 100GB approximation

filepath.Walk(lp.basePath, func(path string, info os.FileInfo, err error) error {
if err == nil && !info.IsDir() {
used += info.Size()
}
return nil
})
}

var latency time.Duration
var cost float64

switch tier {
case HotTier:
latency = 1 * time.Millisecond
cost = 0.10
case WarmTier:
latency = 10 * time.Millisecond
cost = 0.05
case ColdTier:
latency = 100 * time.Millisecond
cost = 0.01
}

return TierInfo{
Name:        tier.String(),
Capacity:    available + used,
Used:        used,
Available:   available,
AccessCost:  cost,
StorageCost: cost / 10,
Latency:     latency,
Throughput:  1024 * 1024 * 100, // 100MB/s
}
}
// Additional methods for TieredStorage

func (ts *TieredStorage) GetFileInfo(key string) (*FileInfo, error) {
return ts.metadata.Get(key)
}

func (ts *TieredStorage) ListFiles(prefix string) []FileInfo {
files := ts.metadata.List(prefix)
result := make([]FileInfo, len(files))
for i, file := range files {
result[i] = *file
}
return result
}

func (ts *TieredStorage) GetTierInfo(tier StorageTier) TierInfo {
if provider, exists := ts.providers[tier]; exists {
return provider.GetTierInfo(tier)
}
return TierInfo{}
}

func (ts *TieredStorage) GetMigrationStats() interface{} {
if ts.migrator != nil {
return ts.migrator.GetStats()
}
return map[string]interface{}{"enabled": false}
}

func (ts *TieredStorage) GetAccessStats() interface{} {
if ts.analytics != nil {
return ts.analytics.GetStats()
}
return map[string]interface{}{}
}

func (ts *TieredStorage) GetTopFiles(limit int) []FileAccessStats {
if ts.analytics != nil {
return ts.analytics.GetTopFiles(limit)
}
return []FileAccessStats{}
}

