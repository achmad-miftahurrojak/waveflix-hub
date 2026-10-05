package server

import (
"context"
"encoding/json"
"fmt"
"io"
"log"
"net/http"
"os"
"strconv"
"time"

"github.com/waveflix-hub/hls-service/internal/hls"
"github.com/waveflix-hub/hls-service/internal/storage"
"github.com/gorilla/mux"
)

type HLSServer struct {
segmenter       *hls.Segmenter
playlistManager *hls.PlaylistManager
storage         *storage.SegmentStorage
router          *mux.Router
port            string
}

type ProcessRequest struct {
ContentID   string   `json:"content_id"`
InputPath   string   `json:"input_path"`
Qualities   []string `json:"qualities"`
Priority    int      `json:"priority"`
}

type ProcessResponse struct {
JobID     string `json:"job_id"`
Status    string `json:"status"`
Message   string `json:"message"`
}

type JobStatus struct {
JobID        string    `json:"job_id"`
ContentID    string    `json:"content_id"`
Status       string    `json:"status"`
Progress     float64   `json:"progress"`
Quality      string    `json:"current_quality"`
StartedAt    time.Time `json:"started_at"`
CompletedAt  *time.Time `json:"completed_at,omitempty"`
Error        string    `json:"error,omitempty"`
}

func NewHLSServer(port string) *HLSServer {
workDir := os.Getenv("HLS_WORK_DIR")
if workDir == "" {
workDir = "/tmp/hls"
}

ffmpegPath := os.Getenv("FFMPEG_PATH")
if ffmpegPath == "" {
ffmpegPath = "ffmpeg"
}

storageProvider := storage.NewLocalStorage(workDir)
segmentStorage := storage.NewSegmentStorage(storageProvider)

server := &HLSServer{
segmenter:       hls.NewSegmenter(ffmpegPath, workDir),
playlistManager: hls.NewPlaylistManager(workDir),
storage:         segmentStorage,
port:            port,
}

server.setupRoutes()
return server
}

func (s *HLSServer) setupRoutes() {
s.router = mux.NewRouter()

api := s.router.PathPrefix("/api/v1").Subrouter()
api.HandleFunc("/process", s.handleProcess).Methods("POST")
api.HandleFunc("/status/{job_id}", s.handleStatus).Methods("GET")
api.HandleFunc("/content/{content_id}/master.m3u8", s.handleMasterPlaylist).Methods("GET")
api.HandleFunc("/content/{content_id}/{quality}/playlist.m3u8", s.handlePlaylist).Methods("GET")
api.HandleFunc("/content/{content_id}/{quality}/segment_{segment:[0-9]+}.ts", s.handleSegment).Methods("GET")

s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
s.router.Use(s.loggingMiddleware)
s.router.Use(s.corsMiddleware)
}

func (s *HLSServer) handleProcess(w http.ResponseWriter, r *http.Request) {
var req ProcessRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
s.writeError(w, http.StatusBadRequest, "Invalid request body")
return
}

if req.ContentID == "" || req.InputPath == "" {
s.writeError(w, http.StatusBadRequest, "Missing required fields")
return
}

jobID := fmt.Sprintf("%s-%d", req.ContentID, time.Now().Unix())

go s.processContent(jobID, req)

response := ProcessResponse{
JobID:   jobID,
Status:  "processing",
Message: "Content processing started",
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(response)
}

func (s *HLSServer) processContent(jobID string, req ProcessRequest) {
ctx := context.Background()

log.Printf("[HLS] Starting processing job %s for content %s", jobID, req.ContentID)

qualities := req.Qualities
if len(qualities) == 0 {
qualities = []string{"720p", "480p", "360p"}
}

var qualityProfiles []hls.QualityProfile
for _, quality := range qualities {
if profile, exists := hls.DefaultProfiles[quality]; exists {
qualityProfiles = append(qualityProfiles, profile)
}
}

for _, profile := range qualityProfiles {
progress := make(chan hls.SegmentProgress, 10)
done := make(chan hls.SegmentResult, 1)

job := &hls.SegmentationJob{
ID:        fmt.Sprintf("%s-%s", jobID, profile.Name),
InputPath: req.InputPath,
OutputDir: req.ContentID,
Quality:   profile,
Progress:  progress,
Done:      done,
}

s.segmenter.ProcessVideo(ctx, job)

go func(quality string) {
for prog := range progress {
log.Printf("[HLS] Job %s progress: %s at %sx", prog.JobID, prog.Quality, prog.Speed)
}
}(profile.Name)

result := <-done
if !result.Success {
log.Printf("[HLS] Job %s failed: %v", result.JobID, result.Error)
continue
}

log.Printf("[HLS] Job %s completed: %d segments in %v", result.JobID, result.SegmentCount, result.Duration)

segments, err := s.playlistManager.LoadSegmentsFromDirectory(
fmt.Sprintf("%s/%s/%s", s.segmenter.WorkDir, req.ContentID, profile.Name))
if err != nil {
log.Printf("[HLS] Failed to load segments for %s: %v", profile.Name, err)
continue
}

if err := s.playlistManager.CreatePlaylist(req.ContentID, profile.Name, segments); err != nil {
log.Printf("[HLS] Failed to create playlist for %s: %v", profile.Name, err)
}
}

if err := s.playlistManager.GenerateVariantPlaylist(req.ContentID, qualityProfiles); err != nil {
log.Printf("[HLS] Failed to generate master playlist: %v", err)
}

log.Printf("[HLS] Processing completed for content %s", req.ContentID)
}

func (s *HLSServer) handleStatus(w http.ResponseWriter, r *http.Request) {
vars := mux.Vars(r)
jobID := vars["job_id"]

status := JobStatus{
JobID:     jobID,
Status:    "completed",
Progress:  100.0,
StartedAt: time.Now().Add(-5 * time.Minute),
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(status)
}

func (s *HLSServer) handleMasterPlaylist(w http.ResponseWriter, r *http.Request) {
vars := mux.Vars(r)
contentID := vars["content_id"]

reader, err := s.storage.RetrieveMasterPlaylist(r.Context(), contentID)
if err != nil {
s.writeError(w, http.StatusNotFound, "Master playlist not found")
return
}
defer reader.Close()

w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
w.Header().Set("Cache-Control", "max-age=300")

io.Copy(w, reader)
}

func (s *HLSServer) handlePlaylist(w http.ResponseWriter, r *http.Request) {
vars := mux.Vars(r)
contentID := vars["content_id"]
quality := vars["quality"]

reader, err := s.storage.RetrievePlaylist(r.Context(), contentID, quality)
if err != nil {
s.writeError(w, http.StatusNotFound, "Playlist not found")
return
}
defer reader.Close()

w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
w.Header().Set("Cache-Control", "max-age=30")

io.Copy(w, reader)
}

func (s *HLSServer) handleSegment(w http.ResponseWriter, r *http.Request) {
vars := mux.Vars(r)
contentID := vars["content_id"]
quality := vars["quality"]
segmentStr := vars["segment"]

segmentIndex, err := strconv.Atoi(segmentStr)
if err != nil {
s.writeError(w, http.StatusBadRequest, "Invalid segment number")
return
}

reader, err := s.storage.RetrieveSegment(r.Context(), contentID, quality, segmentIndex)
if err != nil {
s.writeError(w, http.StatusNotFound, "Segment not found")
return
}
defer reader.Close()

w.Header().Set("Content-Type", "video/mp2t")
w.Header().Set("Cache-Control", "max-age=86400, immutable")
w.Header().Set("Accept-Ranges", "bytes")

io.Copy(w, reader)
}

func (s *HLSServer) handleHealth(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(map[string]string{
"status": "healthy",
"time":   time.Now().Format(time.RFC3339),
})
}

func (s *HLSServer) writeError(w http.ResponseWriter, status int, message string) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(status)
json.NewEncoder(w).Encode(map[string]string{
"error": message,
})
}

func (s *HLSServer) loggingMiddleware(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
start := time.Now()
next.ServeHTTP(w, r)
log.Printf("[HLS] %s %s %v", r.Method, r.URL.Path, time.Since(start))
})
}

func (s *HLSServer) corsMiddleware(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Access-Control-Allow-Origin", "*")
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

if r.Method == "OPTIONS" {
w.WriteHeader(http.StatusOK)
return
}

next.ServeHTTP(w, r)
})
}

func (s *HLSServer) Start() error {
log.Printf("[HLS] Starting HLS server on port %s", s.port)
return http.ListenAndServe(":"+s.port, s.router)
}
