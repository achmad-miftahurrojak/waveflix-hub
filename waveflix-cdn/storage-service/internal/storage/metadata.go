package storage

import (
"encoding/json"
"fmt"
"sync"
"time"
)

type MetadataStore struct {
data  map[string]*FileInfo
mutex sync.RWMutex
}

func NewMetadataStore() *MetadataStore {
return &MetadataStore{
data: make(map[string]*FileInfo),
}
}

func (ms *MetadataStore) Set(key string, fileInfo *FileInfo) {
ms.mutex.Lock()
defer ms.mutex.Unlock()
ms.data[key] = fileInfo
}

func (ms *MetadataStore) Get(key string) (*FileInfo, error) {
ms.mutex.RLock()
defer ms.mutex.RUnlock()

if fileInfo, exists := ms.data[key]; exists {
return fileInfo, nil
}

return nil, fmt.Errorf("metadata not found for key: %s", key)
}

func (ms *MetadataStore) Delete(key string) {
ms.mutex.Lock()
defer ms.mutex.Unlock()
delete(ms.data, key)
}

func (ms *MetadataStore) List(prefix string) []*FileInfo {
ms.mutex.RLock()
defer ms.mutex.RUnlock()

var result []*FileInfo
for key, fileInfo := range ms.data {
if prefix == "" || len(key) >= len(prefix) && key[:len(prefix)] == prefix {
result = append(result, fileInfo)
}
}

return result
}

func (ms *MetadataStore) Count() int {
ms.mutex.RLock()
defer ms.mutex.RUnlock()
return len(ms.data)
}

func (ms *MetadataStore) Export() ([]byte, error) {
ms.mutex.RLock()
defer ms.mutex.RUnlock()
return json.Marshal(ms.data)
}

func (ms *MetadataStore) Import(data []byte) error {
var imported map[string]*FileInfo
if err := json.Unmarshal(data, &imported); err != nil {
return err
}

ms.mutex.Lock()
defer ms.mutex.Unlock()
ms.data = imported

return nil
}

type AccessAnalytics struct {
accessCounts map[string]int64
lastAccess   map[string]time.Time
hourlyStats  map[string]map[int]int64 // key -> hour -> access count
mutex        sync.RWMutex
}

func NewAccessAnalytics() *AccessAnalytics {
return &AccessAnalytics{
accessCounts: make(map[string]int64),
lastAccess:   make(map[string]time.Time),
hourlyStats:  make(map[string]map[int]int64),
}
}

func (aa *AccessAnalytics) RecordAccess(key string, tier StorageTier) {
aa.mutex.Lock()
defer aa.mutex.Unlock()

now := time.Now()
hour := now.Hour()

aa.accessCounts[key]++
aa.lastAccess[key] = now

if _, exists := aa.hourlyStats[key]; !exists {
aa.hourlyStats[key] = make(map[int]int64)
}
aa.hourlyStats[key][hour]++
}

func (aa *AccessAnalytics) GetAccessCount(key string) int64 {
aa.mutex.RLock()
defer aa.mutex.RUnlock()
return aa.accessCounts[key]
}

func (aa *AccessAnalytics) GetLastAccess(key string) time.Time {
aa.mutex.RLock()
defer aa.mutex.RUnlock()
return aa.lastAccess[key]
}

func (aa *AccessAnalytics) GetAccessFrequency(key string) float64 {
aa.mutex.RLock()
defer aa.mutex.RUnlock()

lastAccess, exists := aa.lastAccess[key]
if !exists {
return 0
}

hoursSince := time.Since(lastAccess).Hours()
if hoursSince < 1 {
hoursSince = 1
}

return float64(aa.accessCounts[key]) / hoursSince
}

func (aa *AccessAnalytics) GetHourlyStats(key string) map[int]int64 {
aa.mutex.RLock()
defer aa.mutex.RUnlock()

if stats, exists := aa.hourlyStats[key]; exists {
result := make(map[int]int64)
for hour, count := range stats {
result[hour] = count
}
return result
}

return make(map[int]int64)
}

func (aa *AccessAnalytics) GetTopFiles(limit int) []FileAccessStats {
aa.mutex.RLock()
defer aa.mutex.RUnlock()

type keyCount struct {
key   string
count int64
}

var pairs []keyCount
for key, count := range aa.accessCounts {
pairs = append(pairs, keyCount{key, count})
}

// Simple bubble sort for top N
for i := 0; i < len(pairs)-1; i++ {
for j := 0; j < len(pairs)-i-1; j++ {
if pairs[j].count < pairs[j+1].count {
pairs[j], pairs[j+1] = pairs[j+1], pairs[j]
}
}
}

var result []FileAccessStats
maxItems := limit
if len(pairs) < maxItems {
maxItems = len(pairs)
}

for i := 0; i < maxItems; i++ {
result = append(result, FileAccessStats{
Key:         pairs[i].key,
AccessCount: pairs[i].count,
LastAccess:  aa.lastAccess[pairs[i].key],
})
}

return result
}

func (aa *AccessAnalytics) RemoveFile(key string) {
aa.mutex.Lock()
defer aa.mutex.Unlock()

delete(aa.accessCounts, key)
delete(aa.lastAccess, key)
delete(aa.hourlyStats, key)
}

func (aa *AccessAnalytics) GetStats() map[string]interface{} {
aa.mutex.RLock()
defer aa.mutex.RUnlock()

totalAccess := int64(0)
for _, count := range aa.accessCounts {
totalAccess += count
}

return map[string]interface{}{
"total_files":    len(aa.accessCounts),
"total_accesses": totalAccess,
"avg_accesses":   float64(totalAccess) / float64(len(aa.accessCounts)),
}
}

func (aa *AccessAnalytics) Cleanup(olderThan time.Duration) {
aa.mutex.Lock()
defer aa.mutex.Unlock()

cutoff := time.Now().Add(-olderThan)

for key, lastAccess := range aa.lastAccess {
if lastAccess.Before(cutoff) {
delete(aa.accessCounts, key)
delete(aa.lastAccess, key)
delete(aa.hourlyStats, key)
}
}
}

type FileAccessStats struct {
Key         string    `json:"key"`
AccessCount int64     `json:"access_count"`
LastAccess  time.Time `json:"last_access"`
}
