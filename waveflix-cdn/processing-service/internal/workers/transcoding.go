package workers

import (
"bufio"
"context"
"fmt"
"io"
"os"
"os/exec"
"path/filepath"
"regexp"
"strconv"
"strings"
"time"

"github.com/waveflix-hub/processing-service/internal/queue"
)

type TranscodingWorker struct {
workerID    string
ffmpegPath  string
workDir     string
profiles    map[string]TranscodingProfile
maxJobs     int
currentJobs int
}

type TranscodingProfile struct {
Name         string `json:"name"`
Resolution   string `json:"resolution"`
VideoBitrate int    `json:"video_bitrate"`
AudioBitrate int    `json:"audio_bitrate"`
FPS          int    `json:"fps"`
Codec        string `json:"codec"`
Preset       string `json:"preset"`
CRF          int    `json:"crf"`
}

type TranscodingJob struct {
InputPath    string               `json:"input_path"`
OutputPath   string               `json:"output_path"`
ContentID    string               `json:"content_id"`
Profile      TranscodingProfile   `json:"profile"`
Format       string               `json:"format"` // "hls", "mp4", "webm"
Metadata     map[string]interface{} `json:"metadata"`
SegmentTime  int                  `json:"segment_time,omitempty"`
PlaylistSize int                  `json:"playlist_size,omitempty"`
}

var DefaultProfiles = map[string]TranscodingProfile{
"240p": {
Name: "240p", Resolution: "426x240", VideoBitrate: 400, AudioBitrate: 64,
FPS: 24, Codec: "libx264", Preset: "fast", CRF: 28,
},
"360p": {
Name: "360p", Resolution: "640x360", VideoBitrate: 800, AudioBitrate: 96,
FPS: 30, Codec: "libx264", Preset: "fast", CRF: 26,
},
"480p": {
Name: "480p", Resolution: "854x480", VideoBitrate: 1200, AudioBitrate: 128,
FPS: 30, Codec: "libx264", Preset: "fast", CRF: 24,
},
"720p": {
Name: "720p", Resolution: "1280x720", VideoBitrate: 2500, AudioBitrate: 128,
FPS: 30, Codec: "libx264", Preset: "fast", CRF: 23,
},
"1080p": {
Name: "1080p", Resolution: "1920x1080", VideoBitrate: 5000, AudioBitrate: 192,
FPS: 30, Codec: "libx264", Preset: "fast", CRF: 21,
},
"4k": {
Name: "4k", Resolution: "3840x2160", VideoBitrate: 15000, AudioBitrate: 256,
FPS: 30, Codec: "libx265", Preset: "medium", CRF: 20,
},
}

func NewTranscodingWorker(workerID, ffmpegPath, workDir string, maxJobs int) *TranscodingWorker {
return &TranscodingWorker{
workerID:   workerID,
ffmpegPath: ffmpegPath,
workDir:    workDir,
profiles:   DefaultProfiles,
maxJobs:    maxJobs,
}
}

func (tw *TranscodingWorker) GetWorkerID() string {
return tw.workerID
}

func (tw *TranscodingWorker) CanHandle(jobType queue.JobType) bool {
return jobType == queue.JobTypeTranscode || jobType == queue.JobTypeSegment
}

func (tw *TranscodingWorker) ProcessJob(ctx context.Context, job *queue.Job) (*queue.JobResult, error) {
if tw.currentJobs >= tw.maxJobs {
return &queue.JobResult{
JobID:   job.ID,
Success: false,
Error:   "worker at capacity",
}, nil
}

tw.currentJobs++
defer func() { tw.currentJobs-- }()

transcodingJob, err := tw.parseTranscodingJob(job)
if err != nil {
return &queue.JobResult{
JobID:   job.ID,
Success: false,
Error:   fmt.Sprintf("invalid job payload: %v", err),
}, nil
}

result, err := tw.executeTranscoding(ctx, job, transcodingJob)
if err != nil {
return &queue.JobResult{
JobID:   job.ID,
Success: false,
Error:   err.Error(),
}, nil
}

return result, nil
}

func (tw *TranscodingWorker) parseTranscodingJob(job *queue.Job) (*TranscodingJob, error) {
payload := job.Payload

transJob := &TranscodingJob{}

if inputPath, ok := payload["input_path"].(string); ok {
transJob.InputPath = inputPath
} else {
return nil, fmt.Errorf("missing input_path")
}

if outputPath, ok := payload["output_path"].(string); ok {
transJob.OutputPath = outputPath
} else {
return nil, fmt.Errorf("missing output_path")
}

if contentID, ok := payload["content_id"].(string); ok {
transJob.ContentID = contentID
} else {
return nil, fmt.Errorf("missing content_id")
}

if profileName, ok := payload["profile"].(string); ok {
if profile, exists := tw.profiles[profileName]; exists {
transJob.Profile = profile
} else {
return nil, fmt.Errorf("unknown profile: %s", profileName)
}
} else {
transJob.Profile = tw.profiles["720p"] // default
}

transJob.Format = "hls" // default
if format, ok := payload["format"].(string); ok {
transJob.Format = format
}

transJob.SegmentTime = 6 // default
if segTime, ok := payload["segment_time"].(float64); ok {
transJob.SegmentTime = int(segTime)
}

transJob.PlaylistSize = 10 // default
if playlistSize, ok := payload["playlist_size"].(float64); ok {
transJob.PlaylistSize = int(playlistSize)
}

if metadata, ok := payload["metadata"].(map[string]interface{}); ok {
transJob.Metadata = metadata
}

return transJob, nil
}

func (tw *TranscodingWorker) executeTranscoding(ctx context.Context, job *queue.Job, transJob *TranscodingJob) (*queue.JobResult, error) {
outputDir := filepath.Join(tw.workDir, transJob.ContentID, transJob.Profile.Name)
if err := os.MkdirAll(outputDir, 0755); err != nil {
return nil, fmt.Errorf("failed to create output directory: %v", err)
}

var outputPath string
var args []string

switch transJob.Format {
case "hls":
args, outputPath = tw.buildHLSArgs(transJob, outputDir)
case "mp4":
args, outputPath = tw.buildMP4Args(transJob, outputDir)
default:
return nil, fmt.Errorf("unsupported format: %s", transJob.Format)
}

cmd := exec.CommandContext(ctx, tw.ffmpegPath, args...)

stderrPipe, err := cmd.StderrPipe()
if err != nil {
return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
}

if err := cmd.Start(); err != nil {
return nil, fmt.Errorf("failed to start ffmpeg: %v", err)
}

// Monitor progress
go tw.monitorProgress(stderrPipe, job)

startTime := time.Now()
err = cmd.Wait()
duration := time.Since(startTime)

if err != nil {
return &queue.JobResult{
JobID:   job.ID,
Success: false,
Error:   fmt.Sprintf("transcoding failed: %v", err),
}, nil
}

result := &queue.JobResult{
JobID:   job.ID,
Success: true,
Result: map[string]interface{}{
"output_path":     outputPath,
"duration":        duration.Seconds(),
"profile":         transJob.Profile.Name,
"format":          transJob.Format,
"content_id":      transJob.ContentID,
},
}

if transJob.Format == "hls" {
segmentCount, err := tw.countSegments(outputDir)
if err == nil {
result.Result["segment_count"] = segmentCount
}

playlistPath := filepath.Join(outputDir, "playlist.m3u8")
if _, err := os.Stat(playlistPath); err == nil {
result.Result["playlist_path"] = playlistPath
}
}

return result, nil
}

func (tw *TranscodingWorker) buildHLSArgs(transJob *TranscodingJob, outputDir string) ([]string, string) {
playlistPath := filepath.Join(outputDir, "playlist.m3u8")
segmentPattern := filepath.Join(outputDir, "segment_%03d.ts")

args := []string{
"-i", transJob.InputPath,
"-c:v", transJob.Profile.Codec,
"-b:v", fmt.Sprintf("%dk", transJob.Profile.VideoBitrate),
"-c:a", "aac",
"-b:a", fmt.Sprintf("%dk", transJob.Profile.AudioBitrate),
"-s", transJob.Profile.Resolution,
"-r", strconv.Itoa(transJob.Profile.FPS),
"-g", strconv.Itoa(transJob.Profile.FPS * 2), // keyframe interval
"-keyint_min", strconv.Itoa(transJob.Profile.FPS),
"-sc_threshold", "0",
"-f", "hls",
"-hls_time", strconv.Itoa(transJob.SegmentTime),
"-hls_list_size", strconv.Itoa(transJob.PlaylistSize),
"-hls_segment_filename", segmentPattern,
"-hls_flags", "independent_segments+split_by_time",
"-preset", transJob.Profile.Preset,
"-crf", strconv.Itoa(transJob.Profile.CRF),
playlistPath,
}

// Add codec-specific options
if transJob.Profile.Codec == "libx265" {
args = append(args, "-x265-params", "keyint=60:min-keyint=30")
}

return args, playlistPath
}

func (tw *TranscodingWorker) buildMP4Args(transJob *TranscodingJob, outputDir string) ([]string, string) {
outputPath := filepath.Join(outputDir, "output.mp4")

args := []string{
"-i", transJob.InputPath,
"-c:v", transJob.Profile.Codec,
"-b:v", fmt.Sprintf("%dk", transJob.Profile.VideoBitrate),
"-c:a", "aac",
"-b:a", fmt.Sprintf("%dk", transJob.Profile.AudioBitrate),
"-s", transJob.Profile.Resolution,
"-r", strconv.Itoa(transJob.Profile.FPS),
"-preset", transJob.Profile.Preset,
"-crf", strconv.Itoa(transJob.Profile.CRF),
"-movflags", "+faststart",
outputPath,
}

return args, outputPath
}

func (tw *TranscodingWorker) monitorProgress(stderr io.ReadCloser, job *queue.Job) {
defer stderr.Close()

scanner := bufio.NewScanner(stderr)
progressRegex := regexp.MustCompile(`time=(\d{2}:\d{2}:\d{2}\.\d{2}).*speed=\s*(\S+)x`)
durationRegex := regexp.MustCompile(`Duration:\s+(\d{2}:\d{2}:\d{2}\.\d{2})`)

var totalDuration time.Duration

for scanner.Scan() {
line := scanner.Text()

// Parse total duration from ffmpeg output
if matches := durationRegex.FindStringSubmatch(line); len(matches) > 1 {
if dur, err := parseTimeString(matches[1]); err == nil {
totalDuration = dur
}
}

// Parse current progress
if matches := progressRegex.FindStringSubmatch(line); len(matches) > 2 {
if currentTime, err := parseTimeString(matches[1]); err == nil && totalDuration > 0 {
progress := float64(currentTime) / float64(totalDuration) * 100
if progress > 100 {
progress = 100
}
job.Progress = progress
}
}
}
}

func parseTimeString(timeStr string) (time.Duration, error) {
parts := strings.Split(timeStr, ":")
if len(parts) != 3 {
return 0, fmt.Errorf("invalid time format")
}

hours, _ := strconv.Atoi(parts[0])
minutes, _ := strconv.Atoi(parts[1])
seconds, _ := strconv.ParseFloat(parts[2], 64)

duration := time.Duration(hours)*time.Hour + 
time.Duration(minutes)*time.Minute + 
time.Duration(seconds*float64(time.Second))

return duration, nil
}

func (tw *TranscodingWorker) countSegments(outputDir string) (int, error) {
pattern := filepath.Join(outputDir, "segment_*.ts")
files, err := filepath.Glob(pattern)
if err != nil {
return 0, err
}
return len(files), nil
}

type ThumbnailWorker struct {
workerID   string
ffmpegPath string
workDir    string
}

func NewThumbnailWorker(workerID, ffmpegPath, workDir string) *ThumbnailWorker {
return &ThumbnailWorker{
workerID:   workerID,
ffmpegPath: ffmpegPath,
workDir:    workDir,
}
}

func (tw *ThumbnailWorker) GetWorkerID() string {
return tw.workerID
}

func (tw *ThumbnailWorker) CanHandle(jobType queue.JobType) bool {
return jobType == queue.JobTypeThumbnail
}

func (tw *ThumbnailWorker) ProcessJob(ctx context.Context, job *queue.Job) (*queue.JobResult, error) {
inputPath, ok := job.Payload["input_path"].(string)
if !ok {
return &queue.JobResult{
JobID:   job.ID,
Success: false,
Error:   "missing input_path",
}, nil
}

contentID, ok := job.Payload["content_id"].(string)
if !ok {
return &queue.JobResult{
JobID:   job.ID,
Success: false,
Error:   "missing content_id",
}, nil
}

outputDir := filepath.Join(tw.workDir, contentID, "thumbnails")
if err := os.MkdirAll(outputDir, 0755); err != nil {
return &queue.JobResult{
JobID:   job.ID,
Success: false,
Error:   fmt.Sprintf("failed to create output directory: %v", err),
}, nil
}

timestamps := []string{"00:00:10", "00:01:00", "00:05:00"}
var thumbnails []string

for i, timestamp := range timestamps {
outputPath := filepath.Join(outputDir, fmt.Sprintf("thumb_%d.jpg", i))

args := []string{
"-i", inputPath,
"-ss", timestamp,
"-vframes", "1",
"-vf", "scale=320:240",
"-y",
outputPath,
}

cmd := exec.CommandContext(ctx, tw.ffmpegPath, args...)
if err := cmd.Run(); err != nil {
continue // Skip failed thumbnails
}

thumbnails = append(thumbnails, outputPath)
}

return &queue.JobResult{
JobID:   job.ID,
Success: len(thumbnails) > 0,
Result: map[string]interface{}{
"thumbnails":  thumbnails,
"content_id":  contentID,
"count":       len(thumbnails),
},
}, nil
}

