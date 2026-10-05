package server

import (
"encoding/json"
"fmt"
"log"
"net/http"
"time"

"github.com/waveflix-hub/processing-service/internal/queue"
"github.com/gorilla/mux"
"github.com/google/uuid"
)

type ProcessingServer struct {
queueManager *queue.QueueManager
router       *mux.Router
port         string
}

type JobRequest struct {
Type        string                 `json:"type"`
Priority    int                    `json:"priority"`
Payload     map[string]interface{} `json:"payload"`
MaxRetries  int                    `json:"max_retries,omitempty"`
RetryDelay  string                 `json:"retry_delay,omitempty"`
Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type JobResponse struct {
JobID     string `json:"job_id"`
Status    string `json:"status"`
Message   string `json:"message"`
Error     string `json:"error,omitempty"`
}

type BatchJobRequest struct {
Jobs []JobRequest `json:"jobs"`
}

type BatchJobResponse struct {
JobIDs  []string `json:"job_ids"`
Status  string   `json:"status"`
Message string   `json:"message"`
Failed  int      `json:"failed,omitempty"`
}

func NewProcessingServer(queueManager *queue.QueueManager, port string) *ProcessingServer {
server := &ProcessingServer{
queueManager: queueManager,
port:         port,
}

server.setupRoutes()
return server
}

func (ps *ProcessingServer) setupRoutes() {
ps.router = mux.NewRouter()

api := ps.router.PathPrefix("/api/v1").Subrouter()

// Job management
api.HandleFunc("/jobs", ps.handleSubmitJob).Methods("POST")
api.HandleFunc("/jobs/batch", ps.handleBatchSubmit).Methods("POST")
api.HandleFunc("/jobs/{job_id}", ps.handleGetJob).Methods("GET")
api.HandleFunc("/jobs/{job_id}/status", ps.handleGetJobStatus).Methods("GET")

// Queue management
api.HandleFunc("/queue/status", ps.handleQueueStatus).Methods("GET")
api.HandleFunc("/queue/metrics", ps.handleQueueMetrics).Methods("GET")

// Specific job types
api.HandleFunc("/transcode", ps.handleTranscode).Methods("POST")
api.HandleFunc("/thumbnail", ps.handleThumbnail).Methods("POST")

// Health and monitoring
ps.router.HandleFunc("/health", ps.handleHealth).Methods("GET")
ps.router.HandleFunc("/ready", ps.handleReady).Methods("GET")

ps.router.Use(ps.loggingMiddleware)
ps.router.Use(ps.corsMiddleware)
}

func (ps *ProcessingServer) handleSubmitJob(w http.ResponseWriter, r *http.Request) {
var req JobRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
ps.writeError(w, http.StatusBadRequest, "Invalid request body")
return
}

job, err := ps.createJobFromRequest(&req)
if err != nil {
ps.writeError(w, http.StatusBadRequest, err.Error())
return
}

if err := ps.queueManager.EnqueueJob(job); err != nil {
ps.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to enqueue job: %v", err))
return
}

response := JobResponse{
JobID:   job.ID,
Status:  string(job.Status),
Message: "Job submitted successfully",
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(response)
}

func (ps *ProcessingServer) handleBatchSubmit(w http.ResponseWriter, r *http.Request) {
var req BatchJobRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
ps.writeError(w, http.StatusBadRequest, "Invalid request body")
return
}

var jobs []*queue.Job
var jobIDs []string
failed := 0

for _, jobReq := range req.Jobs {
job, err := ps.createJobFromRequest(&jobReq)
if err != nil {
failed++
continue
}
jobs = append(jobs, job)
jobIDs = append(jobIDs, job.ID)
}

if len(jobs) == 0 {
ps.writeError(w, http.StatusBadRequest, "No valid jobs in batch")
return
}

if err := ps.queueManager.BatchEnqueue(jobs); err != nil {
ps.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to enqueue batch: %v", err))
return
}

response := BatchJobResponse{
JobIDs:  jobIDs,
Status:  "submitted",
Message: fmt.Sprintf("Batch submitted: %d jobs enqueued", len(jobs)),
Failed:  failed,
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(response)
}

func (ps *ProcessingServer) handleGetJob(w http.ResponseWriter, r *http.Request) {
vars := mux.Vars(r)
jobID := vars["job_id"]

job, err := ps.queueManager.GetJobStatus(jobID)
if err != nil {
ps.writeError(w, http.StatusNotFound, "Job not found")
return
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(job)
}

func (ps *ProcessingServer) handleGetJobStatus(w http.ResponseWriter, r *http.Request) {
vars := mux.Vars(r)
jobID := vars["job_id"]

job, err := ps.queueManager.GetJobStatus(jobID)
if err != nil {
ps.writeError(w, http.StatusNotFound, "Job not found")
return
}

status := map[string]interface{}{
"job_id":       job.ID,
"status":       job.Status,
"progress":     job.Progress,
"created_at":   job.CreatedAt,
"started_at":   job.StartedAt,
"completed_at": job.CompletedAt,
"worker_id":    job.WorkerID,
"retry_count":  job.RetryCount,
"last_error":   job.LastError,
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(status)
}

func (ps *ProcessingServer) handleQueueStatus(w http.ResponseWriter, r *http.Request) {
status := ps.queueManager.GetQueueStatus()

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(status)
}

func (ps *ProcessingServer) handleQueueMetrics(w http.ResponseWriter, r *http.Request) {
metrics := ps.queueManager.GetQueueMetrics()

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(metrics)
}

func (ps *ProcessingServer) handleTranscode(w http.ResponseWriter, r *http.Request) {
var payload map[string]interface{}
if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
ps.writeError(w, http.StatusBadRequest, "Invalid request body")
return
}

priority := queue.NormalPriority
if p, ok := payload["priority"].(float64); ok {
priority = queue.Priority(int(p))
}

job := &queue.Job{
ID:       uuid.New().String(),
Type:     queue.JobTypeTranscode,
Priority: priority,
Payload:  payload,
Metadata: map[string]interface{}{
"endpoint": "transcode",
},
}

if err := ps.queueManager.EnqueueJob(job); err != nil {
ps.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to enqueue job: %v", err))
return
}

response := JobResponse{
JobID:   job.ID,
Status:  string(job.Status),
Message: "Transcoding job submitted",
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(response)
}

func (ps *ProcessingServer) handleThumbnail(w http.ResponseWriter, r *http.Request) {
var payload map[string]interface{}
if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
ps.writeError(w, http.StatusBadRequest, "Invalid request body")
return
}

job := &queue.Job{
ID:       uuid.New().String(),
Type:     queue.JobTypeThumbnail,
Priority: queue.NormalPriority,
Payload:  payload,
Metadata: map[string]interface{}{
"endpoint": "thumbnail",
},
}

if err := ps.queueManager.EnqueueJob(job); err != nil {
ps.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to enqueue job: %v", err))
return
}

response := JobResponse{
JobID:   job.ID,
Status:  string(job.Status),
Message: "Thumbnail job submitted",
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(response)
}

func (ps *ProcessingServer) handleHealth(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(map[string]string{
"status": "healthy",
"time":   time.Now().Format(time.RFC3339),
})
}

func (ps *ProcessingServer) handleReady(w http.ResponseWriter, r *http.Request) {
status := ps.queueManager.GetQueueStatus()

ready := true
if queueLength, ok := status["queue_length"].(int); ok && queueLength > 1000 {
ready = false
}

if ready {
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(map[string]interface{}{
"ready": true,
"status": status,
})
} else {
w.WriteHeader(http.StatusServiceUnavailable)
json.NewEncoder(w).Encode(map[string]interface{}{
"ready": false,
"status": status,
})
}
}

func (ps *ProcessingServer) createJobFromRequest(req *JobRequest) (*queue.Job, error) {
jobType, err := ps.parseJobType(req.Type)
if err != nil {
return nil, err
}

priority := queue.Priority(req.Priority)
if priority < queue.LowPriority || priority > queue.CriticalPriority {
priority = queue.NormalPriority
}

job := &queue.Job{
ID:       uuid.New().String(),
Type:     jobType,
Priority: priority,
Payload:  req.Payload,
Metadata: req.Metadata,
}

if req.MaxRetries > 0 {
job.MaxRetries = req.MaxRetries
}

if req.RetryDelay != "" {
if delay, err := time.ParseDuration(req.RetryDelay); err == nil {
job.RetryDelay = delay
}
}

return job, nil
}

func (ps *ProcessingServer) parseJobType(typeStr string) (queue.JobType, error) {
switch typeStr {
case "transcode":
return queue.JobTypeTranscode, nil
case "segment":
return queue.JobTypeSegment, nil
case "thumbnail":
return queue.JobTypeThumbnail, nil
case "analyze":
return queue.JobTypeAnalyze, nil
default:
return "", fmt.Errorf("unknown job type: %s", typeStr)
}
}

func (ps *ProcessingServer) writeError(w http.ResponseWriter, status int, message string) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(status)
json.NewEncoder(w).Encode(map[string]string{
"error": message,
})
}

func (ps *ProcessingServer) loggingMiddleware(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
start := time.Now()
next.ServeHTTP(w, r)
log.Printf("[Processing] %s %s %v", r.Method, r.URL.Path, time.Since(start))
})
}

func (ps *ProcessingServer) corsMiddleware(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Access-Control-Allow-Origin", "*")
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

if r.Method == "OPTIONS" {
w.WriteHeader(http.StatusOK)
return
}

next.ServeHTTP(w, r)
})
}

func (ps *ProcessingServer) Start() error {
log.Printf("[Processing] Starting processing server on port %s", ps.port)
return http.ListenAndServe(":"+ps.port, ps.router)
}

