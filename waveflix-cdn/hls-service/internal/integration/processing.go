package integration

import (
"bytes"
"context"
"encoding/json"
"fmt"
"io"
"net/http"
"time"
)

type ProcessingServiceClient struct {
baseURL string
client  *http.Client
}

type TranscodingRequest struct {
ContentID   string                 `json:"content_id"`
InputPath   string                 `json:"input_path"`
OutputPath  string                 `json:"output_path"`
Profiles    []TranscodingProfile   `json:"profiles"`
Metadata    map[string]interface{} `json:"metadata"`
Priority    int                    `json:"priority"`
}

type TranscodingProfile struct {
Name         string `json:"name"`
Resolution   string `json:"resolution"`
VideoBitrate int    `json:"video_bitrate"`
AudioBitrate int    `json:"audio_bitrate"`
FPS          int    `json:"fps"`
Codec        string `json:"codec"`
}

type TranscodingResponse struct {
JobID     string `json:"job_id"`
Status    string `json:"status"`
Message   string `json:"message"`
Error     string `json:"error,omitempty"`
}

type JobStatusResponse struct {
JobID       string    `json:"job_id"`
Status      string    `json:"status"`
Progress    float64   `json:"progress"`
StartedAt   time.Time `json:"started_at"`
CompletedAt time.Time `json:"completed_at,omitempty"`
Error       string    `json:"error,omitempty"`
}

func NewProcessingServiceClient(baseURL string) *ProcessingServiceClient {
return &ProcessingServiceClient{
baseURL: baseURL,
client: &http.Client{
Timeout: 30 * time.Second,
},
}
}

func (psc *ProcessingServiceClient) SubmitTranscodingJob(ctx context.Context, req *TranscodingRequest) (*TranscodingResponse, error) {
jsonData, err := json.Marshal(req)
if err != nil {
return nil, fmt.Errorf("failed to marshal request: %w", err)
}

httpReq, err := http.NewRequestWithContext(ctx, "POST", 
fmt.Sprintf("%s/api/v1/transcode", psc.baseURL), bytes.NewBuffer(jsonData))
if err != nil {
return nil, fmt.Errorf("failed to create request: %w", err)
}

httpReq.Header.Set("Content-Type", "application/json")

resp, err := psc.client.Do(httpReq)
if err != nil {
return nil, fmt.Errorf("failed to send request: %w", err)
}
defer resp.Body.Close()

body, err := io.ReadAll(resp.Body)
if err != nil {
return nil, fmt.Errorf("failed to read response: %w", err)
}

if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
}

var response TranscodingResponse
if err := json.Unmarshal(body, &response); err != nil {
return nil, fmt.Errorf("failed to unmarshal response: %w", err)
}

return &response, nil
}

func (psc *ProcessingServiceClient) GetJobStatus(ctx context.Context, jobID string) (*JobStatusResponse, error) {
httpReq, err := http.NewRequestWithContext(ctx, "GET", 
fmt.Sprintf("%s/api/v1/jobs/%s/status", psc.baseURL, jobID), nil)
if err != nil {
return nil, fmt.Errorf("failed to create request: %w", err)
}

resp, err := psc.client.Do(httpReq)
if err != nil {
return nil, fmt.Errorf("failed to send request: %w", err)
}
defer resp.Body.Close()

body, err := io.ReadAll(resp.Body)
if err != nil {
return nil, fmt.Errorf("failed to read response: %w", err)
}

if resp.StatusCode != http.StatusOK {
return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
}

var response JobStatusResponse
if err := json.Unmarshal(body, &response); err != nil {
return nil, fmt.Errorf("failed to unmarshal response: %w", err)
}

return &response, nil
}

func (psc *ProcessingServiceClient) WaitForCompletion(ctx context.Context, jobID string, pollInterval time.Duration) (*JobStatusResponse, error) {
ticker := time.NewTicker(pollInterval)
defer ticker.Stop()

for {
select {
case <-ctx.Done():
return nil, ctx.Err()
case <-ticker.C:
status, err := psc.GetJobStatus(ctx, jobID)
if err != nil {
return nil, err
}

switch status.Status {
case "completed":
return status, nil
case "failed", "error":
return status, fmt.Errorf("job failed: %s", status.Error)
}
}
}
}

type HLSIntegration struct {
processingClient *ProcessingServiceClient
}

func NewHLSIntegration(processingServiceURL string) *HLSIntegration {
return &HLSIntegration{
processingClient: NewProcessingServiceClient(processingServiceURL),
}
}

func (hi *HLSIntegration) ProcessVideoToHLS(ctx context.Context, contentID, inputPath string, qualities []string) error {
var profiles []TranscodingProfile

qualityMap := map[string]TranscodingProfile{
"240p":  {Name: "240p", Resolution: "426x240", VideoBitrate: 400, AudioBitrate: 64, FPS: 24, Codec: "libx264"},
"360p":  {Name: "360p", Resolution: "640x360", VideoBitrate: 800, AudioBitrate: 96, FPS: 30, Codec: "libx264"},
"480p":  {Name: "480p", Resolution: "854x480", VideoBitrate: 1200, AudioBitrate: 128, FPS: 30, Codec: "libx264"},
"720p":  {Name: "720p", Resolution: "1280x720", VideoBitrate: 2500, AudioBitrate: 128, FPS: 30, Codec: "libx264"},
"1080p": {Name: "1080p", Resolution: "1920x1080", VideoBitrate: 5000, AudioBitrate: 192, FPS: 30, Codec: "libx264"},
"4k":    {Name: "4k", Resolution: "3840x2160", VideoBitrate: 15000, AudioBitrate: 256, FPS: 30, Codec: "libx265"},
}

for _, quality := range qualities {
if profile, exists := qualityMap[quality]; exists {
profiles = append(profiles, profile)
}
}

req := &TranscodingRequest{
ContentID:  contentID,
InputPath:  inputPath,
OutputPath: fmt.Sprintf("/hls/%s", contentID),
Profiles:   profiles,
Priority:   1,
Metadata: map[string]interface{}{
"format":      "hls",
"segment_time": 6,
"playlist_size": 10,
},
}

resp, err := hi.processingClient.SubmitTranscodingJob(ctx, req)
if err != nil {
return fmt.Errorf("failed to submit transcoding job: %w", err)
}

if resp.Status == "error" {
return fmt.Errorf("transcoding job failed: %s", resp.Error)
}

_, err = hi.processingClient.WaitForCompletion(ctx, resp.JobID, 10*time.Second)
if err != nil {
return fmt.Errorf("transcoding job did not complete successfully: %w", err)
}

return nil
}
