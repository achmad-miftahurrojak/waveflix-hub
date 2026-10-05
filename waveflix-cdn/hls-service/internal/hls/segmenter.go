package hls

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
)

type Segmenter struct {
FFmpegPath    string
WorkDir       string
SegmentTime   int
PlaylistSize  int
}

type SegmentationJob struct {
ID           string
InputPath    string
OutputDir    string
Quality      QualityProfile
Progress     chan SegmentProgress
Done         chan SegmentResult
}

type SegmentProgress struct {
JobID        string
Processed    int
Total        int
CurrentFile  string
Speed        string
Quality      string
}

type SegmentResult struct {
JobID        string
Success      bool
Error        error
PlaylistPath string
SegmentCount int
Duration     time.Duration
}

type QualityProfile struct {
Name       string
Resolution string
VideoBitrate int
AudioBitrate int
FPS        int
Codec      string
}

var DefaultProfiles = map[string]QualityProfile{
"240p": {Name: "240p", Resolution: "426x240", VideoBitrate: 400, AudioBitrate: 64, FPS: 24, Codec: "libx264"},
"360p": {Name: "360p", Resolution: "640x360", VideoBitrate: 800, AudioBitrate: 96, FPS: 30, Codec: "libx264"},
"480p": {Name: "480p", Resolution: "854x480", VideoBitrate: 1200, AudioBitrate: 128, FPS: 30, Codec: "libx264"},
"720p": {Name: "720p", Resolution: "1280x720", VideoBitrate: 2500, AudioBitrate: 128, FPS: 30, Codec: "libx264"},
"1080p": {Name: "1080p", Resolution: "1920x1080", VideoBitrate: 5000, AudioBitrate: 192, FPS: 30, Codec: "libx264"},
"4k": {Name: "4k", Resolution: "3840x2160", VideoBitrate: 15000, AudioBitrate: 256, FPS: 30, Codec: "libx265"},
}

func NewSegmenter(ffmpegPath, workDir string) *Segmenter {
return &Segmenter{
FFmpegPath:   ffmpegPath,
WorkDir:      workDir,
SegmentTime:  6,
PlaylistSize: 10,
}
}

func (s *Segmenter) ProcessVideo(ctx context.Context, job *SegmentationJob) {
go func() {
defer close(job.Done)

outputDir := filepath.Join(s.WorkDir, job.OutputDir, job.Quality.Name)
if err := os.MkdirAll(outputDir, 0755); err != nil {
job.Done <- SegmentResult{JobID: job.ID, Success: false, Error: err}
return
}

playlistPath := filepath.Join(outputDir, "playlist.m3u8")
segmentPattern := filepath.Join(outputDir, "segment_%03d.ts")

args := s.buildFFmpegArgs(job.InputPath, playlistPath, segmentPattern, job.Quality)

cmd := exec.CommandContext(ctx, s.FFmpegPath, args...)

stderrPipe, err := cmd.StderrPipe()
if err != nil {
job.Done <- SegmentResult{JobID: job.ID, Success: false, Error: err}
return
}

if err := cmd.Start(); err != nil {
job.Done <- SegmentResult{JobID: job.ID, Success: false, Error: err}
return
}

go s.parseFFmpegProgress(stderrPipe, job)

startTime := time.Now()
err = cmd.Wait()
duration := time.Since(startTime)

if err != nil {
job.Done <- SegmentResult{JobID: job.ID, Success: false, Error: err}
return
}

segmentCount, err := s.countSegments(outputDir)
if err != nil {
job.Done <- SegmentResult{JobID: job.ID, Success: false, Error: err}
return
}

job.Done <- SegmentResult{
JobID:        job.ID,
Success:      true,
PlaylistPath: playlistPath,
SegmentCount: segmentCount,
Duration:     duration,
}
}()
}

func (s *Segmenter) buildFFmpegArgs(inputPath, playlistPath, segmentPattern string, quality QualityProfile) []string {
args := []string{
"-i", inputPath,
"-c:v", quality.Codec,
"-b:v", fmt.Sprintf("%dk", quality.VideoBitrate),
"-c:a", "aac",
"-b:a", fmt.Sprintf("%dk", quality.AudioBitrate),
"-s", quality.Resolution,
"-r", strconv.Itoa(quality.FPS),
"-g", strconv.Itoa(quality.FPS * 2),
"-keyint_min", strconv.Itoa(quality.FPS),
"-sc_threshold", "0",
"-f", "hls",
"-hls_time", strconv.Itoa(s.SegmentTime),
"-hls_list_size", strconv.Itoa(s.PlaylistSize),
"-hls_segment_filename", segmentPattern,
"-hls_flags", "independent_segments+split_by_time",
"-preset", "fast",
"-crf", "23",
playlistPath,
}

if quality.Codec == "libx265" {
args = append(args, "-x265-params", "keyint=60:min-keyint=30")
}

return args
}

func (s *Segmenter) parseFFmpegProgress(stderr io.ReadCloser, job *SegmentationJob) {
defer stderr.Close()

scanner := bufio.NewScanner(stderr)
progressRegex := regexp.MustCompile(`time=(\d{2}:\d{2}:\d{2}\.\d{2}).*speed=\s*(\S+)x`)

for scanner.Scan() {
output := scanner.Text()
matches := progressRegex.FindStringSubmatch(output)

if len(matches) >= 3 {
select {
case job.Progress <- SegmentProgress{
JobID:   job.ID,
Speed:   matches[2],
Quality: job.Quality.Name,
}:
default:
}
}
}
}

func (s *Segmenter) countSegments(outputDir string) (int, error) {
files, err := filepath.Glob(filepath.Join(outputDir, "segment_*.ts"))
if err != nil {
return 0, err
}
return len(files), nil
}

func (s *Segmenter) GenerateMasterPlaylist(contentID string, qualities []string) (string, error) {
var playlist strings.Builder

playlist.WriteString("#EXTM3U\n")
playlist.WriteString("#EXT-X-VERSION:6\n")

for _, quality := range qualities {
profile, exists := DefaultProfiles[quality]
if !exists {
continue
}

playlist.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%s,FRAME-RATE=%d\n",
(profile.VideoBitrate+profile.AudioBitrate)*1000,
profile.Resolution,
profile.FPS))
playlist.WriteString(fmt.Sprintf("%s/playlist.m3u8\n", quality))
}

masterPath := filepath.Join(s.WorkDir, contentID, "master.m3u8")
if err := os.MkdirAll(filepath.Dir(masterPath), 0755); err != nil {
return "", err
}

file, err := os.Create(masterPath)
if err != nil {
return "", err
}
defer file.Close()

if _, err := file.WriteString(playlist.String()); err != nil {
return "", err
}

return masterPath, nil
}
