package server

import (
"encoding/json"
"fmt"
"io"
"log"
"net/http"
"strconv"
"strings"
"time"

"github.com/waveflix-hub/storage-service/internal/storage"
"github.com/gorilla/mux"
)

type StorageServer struct {
tieredStorage *storage.TieredStorage
router        *mux.Router
port          string
}

type UploadRequest struct {
Key         string `json:"key"`
ContentType string `json:"content_type"`
Tier        string `json:"tier,omitempty"`
}

type UploadResponse struct {
Key       string `json:"key"`
Success   bool   `json:"success"`
Message   string `json:"message"`
Size      int64  `json:"size,omitempty"`
ETag      string `json:"etag,omitempty"`
}

type MoveRequest struct {
ToTier string `json:"to_tier"`
}

type FileListResponse struct {
Files []storage.FileInfo `json:"files"`
Total int                `json:"total"`
}

func NewStorageServer(tieredStorage *storage.TieredStorage, port string) *StorageServer {
server := &StorageServer{
tieredStorage: tieredStorage,
port:          port,
}

server.setupRoutes()
return server
}

func (ss *StorageServer) setupRoutes() {
ss.router = mux.NewRouter()

api := ss.router.PathPrefix("/api/v1").Subrouter()

// File operations
api.HandleFunc("/files", ss.handleUpload).Methods("POST")
api.HandleFunc("/files/{key:.*}", ss.handleDownload).Methods("GET")
api.HandleFunc("/files/{key:.*}", ss.handleDelete).Methods("DELETE")
api.HandleFunc("/files/{key:.*}/move", ss.handleMove).Methods("POST")
api.HandleFunc("/files/{key:.*}/info", ss.handleFileInfo).Methods("GET")

// Listing and search
api.HandleFunc("/files", ss.handleListFiles).Methods("GET")

// Storage management
api.HandleFunc("/storage/stats", ss.handleStorageStats).Methods("GET")
api.HandleFunc("/storage/tiers", ss.handleTierInfo).Methods("GET")
api.HandleFunc("/storage/migration/start", ss.handleStartMigration).Methods("POST")
api.HandleFunc("/storage/migration/stats", ss.handleMigrationStats).Methods("GET")

// Analytics
api.HandleFunc("/analytics/access", ss.handleAccessStats).Methods("GET")
api.HandleFunc("/analytics/top", ss.handleTopFiles).Methods("GET")

// Health and monitoring
ss.router.HandleFunc("/health", ss.handleHealth).Methods("GET")
ss.router.HandleFunc("/ready", ss.handleReady).Methods("GET")

ss.router.Use(ss.loggingMiddleware)
ss.router.Use(ss.corsMiddleware)
}

func (ss *StorageServer) handleUpload(w http.ResponseWriter, r *http.Request) {
contentType := r.Header.Get("Content-Type")
if contentType == "" {
contentType = "application/octet-stream"
}

key := r.Header.Get("X-File-Key")
if key == "" {
ss.writeError(w, http.StatusBadRequest, "Missing X-File-Key header")
return
}

err := ss.tieredStorage.Store(r.Context(), key, r.Body, contentType)
if err != nil {
ss.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Upload failed: %v", err))
return
}

response := UploadResponse{
Key:     key,
Success: true,
Message: "File uploaded successfully",
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(response)
}

func (ss *StorageServer) handleDownload(w http.ResponseWriter, r *http.Request) {
vars := mux.Vars(r)
key := vars["key"]

reader, err := ss.tieredStorage.Retrieve(r.Context(), key)
if err != nil {
ss.writeError(w, http.StatusNotFound, "File not found")
return
}
defer reader.Close()

// Get file info for headers
if fileInfo, err := ss.tieredStorage.GetFileInfo(key); err == nil {
w.Header().Set("Content-Type", fileInfo.ContentType)
w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size, 10))
w.Header().Set("ETag", fileInfo.ETag)
w.Header().Set("Last-Modified", fileInfo.LastModified.Format(http.TimeFormat))

// Cache headers based on tier
switch fileInfo.Tier {
case storage.HotTier:
w.Header().Set("Cache-Control", "public, max-age=3600")
case storage.WarmTier:
w.Header().Set("Cache-Control", "public, max-age=86400")
case storage.ColdTier:
w.Header().Set("Cache-Control", "public, max-age=604800")
}
}

io.Copy(w, reader)
}

func (ss *StorageServer) handleDelete(w http.ResponseWriter, r *http.Request) {
vars := mux.Vars(r)
key := vars["key"]

err := ss.tieredStorage.Delete(r.Context(), key)
if err != nil {
ss.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Delete failed: %v", err))
return
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
"success": true,
"message": "File deleted successfully",
})
}

func (ss *StorageServer) handleMove(w http.ResponseWriter, r *http.Request) {
vars := mux.Vars(r)
key := vars["key"]

var req MoveRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
ss.writeError(w, http.StatusBadRequest, "Invalid request body")
return
}

toTier, err := ss.parseTier(req.ToTier)
if err != nil {
ss.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid tier: %s", req.ToTier))
return
}

err = ss.tieredStorage.Move(r.Context(), key, toTier)
if err != nil {
ss.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Move failed: %v", err))
return
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
"success": true,
"message": fmt.Sprintf("File moved to %s tier", req.ToTier),
})
}

func (ss *StorageServer) handleFileInfo(w http.ResponseWriter, r *http.Request) {
vars := mux.Vars(r)
key := vars["key"]

fileInfo, err := ss.tieredStorage.GetFileInfo(key)
if err != nil {
ss.writeError(w, http.StatusNotFound, "File not found")
return
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(fileInfo)
}

func (ss *StorageServer) handleListFiles(w http.ResponseWriter, r *http.Request) {
prefix := r.URL.Query().Get("prefix")
tier := r.URL.Query().Get("tier")

files := ss.tieredStorage.ListFiles(prefix)

// Filter by tier if specified
if tier != "" {
if targetTier, err := ss.parseTier(tier); err == nil {
var filtered []storage.FileInfo
for _, file := range files {
if file.Tier == targetTier {
filtered = append(filtered, file)
}
}
files = filtered
}
}

response := FileListResponse{
Files: files,
Total: len(files),
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(response)
}

func (ss *StorageServer) handleStorageStats(w http.ResponseWriter, r *http.Request) {
stats := ss.tieredStorage.GetStats()

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(stats)
}

func (ss *StorageServer) handleTierInfo(w http.ResponseWriter, r *http.Request) {
tiers := map[string]storage.TierInfo{
"hot":  ss.tieredStorage.GetTierInfo(storage.HotTier),
"warm": ss.tieredStorage.GetTierInfo(storage.WarmTier),
"cold": ss.tieredStorage.GetTierInfo(storage.ColdTier),
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(tiers)
}

func (ss *StorageServer) handleStartMigration(w http.ResponseWriter, r *http.Request) {
go ss.tieredStorage.StartMigration(r.Context())

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
"success": true,
"message": "Migration started",
})
}

func (ss *StorageServer) handleMigrationStats(w http.ResponseWriter, r *http.Request) {
stats := ss.tieredStorage.GetMigrationStats()

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(stats)
}

func (ss *StorageServer) handleAccessStats(w http.ResponseWriter, r *http.Request) {
stats := ss.tieredStorage.GetAccessStats()

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(stats)
}

func (ss *StorageServer) handleTopFiles(w http.ResponseWriter, r *http.Request) {
limitStr := r.URL.Query().Get("limit")
limit := 10

if limitStr != "" {
if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
limit = parsed
}
}

topFiles := ss.tieredStorage.GetTopFiles(limit)

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
"files": topFiles,
"limit": limit,
})
}

func (ss *StorageServer) handleHealth(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(map[string]string{
"status": "healthy",
"time":   time.Now().Format(time.RFC3339),
})
}

func (ss *StorageServer) handleReady(w http.ResponseWriter, r *http.Request) {
stats := ss.tieredStorage.GetStats()

ready := true

// Check if any tier is over 90% capacity
for tierName, tierStats := range stats {
if tierInfo, ok := tierStats.(storage.TierInfo); ok {
if tierInfo.Capacity > 0 {
utilization := float64(tierInfo.Used) / float64(tierInfo.Capacity)
if utilization > 0.9 {
ready = false
log.Printf("[Storage] Tier %s is %0.1f%% full", tierName, utilization*100)
}
}
}
}

if ready {
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(map[string]interface{}{
"ready":  true,
"stats":  stats,
})
} else {
w.WriteHeader(http.StatusServiceUnavailable)
json.NewEncoder(w).Encode(map[string]interface{}{
"ready":  false,
"reason": "Storage capacity exceeded",
"stats":  stats,
})
}
}

func (ss *StorageServer) parseTier(tierStr string) (storage.StorageTier, error) {
switch strings.ToLower(tierStr) {
case "hot":
return storage.HotTier, nil
case "warm":
return storage.WarmTier, nil
case "cold":
return storage.ColdTier, nil
default:
return storage.HotTier, fmt.Errorf("unknown tier: %s", tierStr)
}
}

func (ss *StorageServer) writeError(w http.ResponseWriter, status int, message string) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(status)
json.NewEncoder(w).Encode(map[string]string{
"error": message,
})
}

func (ss *StorageServer) loggingMiddleware(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
start := time.Now()
next.ServeHTTP(w, r)
log.Printf("[Storage] %s %s %v", r.Method, r.URL.Path, time.Since(start))
})
}

func (ss *StorageServer) corsMiddleware(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Access-Control-Allow-Origin", "*")
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-File-Key")

if r.Method == "OPTIONS" {
w.WriteHeader(http.StatusOK)
return
}

next.ServeHTTP(w, r)
})
}

func (ss *StorageServer) Start() error {
log.Printf("[Storage] Starting storage server on port %s", ss.port)
return http.ListenAndServe(":"+ss.port, ss.router)
}
