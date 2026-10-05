package hls

import (
"fmt"
"os"
"path/filepath"
"sort"
"strconv"
"strings"
)

type PlaylistManager struct {
WorkDir     string
MaxSegments int
}

type Segment struct {
Index    int
Duration float64
Filename string
URI      string
}

type Playlist struct {
Version     int
Sequence    int
TargetDur   int
Segments    []Segment
IsLive      bool
HasEndList  bool
}

func NewPlaylistManager(workDir string) *PlaylistManager {
return &PlaylistManager{
WorkDir:     workDir,
MaxSegments: 10,
}
}

func (pm *PlaylistManager) CreatePlaylist(contentID, quality string, segments []Segment) error {
playlist := &Playlist{
Version:    3,
Sequence:   0,
TargetDur:  6,
Segments:   segments,
IsLive:     false,
HasEndList: true,
}

return pm.WritePlaylist(contentID, quality, playlist)
}

func (pm *PlaylistManager) WritePlaylist(contentID, quality string, playlist *Playlist) error {
playlistDir := filepath.Join(pm.WorkDir, contentID, quality)
if err := os.MkdirAll(playlistDir, 0755); err != nil {
return err
}

playlistPath := filepath.Join(playlistDir, "playlist.m3u8")
file, err := os.Create(playlistPath)
if err != nil {
return err
}
defer file.Close()

content := pm.generateM3U8Content(playlist)
_, err = file.WriteString(content)
return err
}

func (pm *PlaylistManager) generateM3U8Content(playlist *Playlist) string {
var builder strings.Builder

builder.WriteString("#EXTM3U\n")
builder.WriteString(fmt.Sprintf("#EXT-X-VERSION:%d\n", playlist.Version))
builder.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", playlist.TargetDur))

if !playlist.IsLive {
builder.WriteString(fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n", playlist.Sequence))
}

for _, segment := range playlist.Segments {
builder.WriteString(fmt.Sprintf("#EXTINF:%.6f,\n", segment.Duration))
builder.WriteString(fmt.Sprintf("%s\n", segment.Filename))
}

if playlist.HasEndList {
builder.WriteString("#EXT-X-ENDLIST\n")
}

return builder.String()
}

func (pm *PlaylistManager) LoadSegmentsFromDirectory(segmentDir string) ([]Segment, error) {
pattern := filepath.Join(segmentDir, "segment_*.ts")
files, err := filepath.Glob(pattern)
if err != nil {
return nil, err
}

sort.Strings(files)

var segments []Segment
for i, file := range files {
filename := filepath.Base(file)

duration, err := pm.getSegmentDuration(file)
if err != nil {
duration = 6.0
}

segment := Segment{
Index:    i,
Duration: duration,
Filename: filename,
URI:      filename,
}
segments = append(segments, segment)
}

return segments, nil
}

func (pm *PlaylistManager) getSegmentDuration(segmentPath string) (float64, error) {
return 6.0, nil
}

func (pm *PlaylistManager) UpdateLivePlaylist(contentID, quality string, newSegment Segment) error {
playlistPath := filepath.Join(pm.WorkDir, contentID, quality, "playlist.m3u8")

playlist, err := pm.parseExistingPlaylist(playlistPath)
if err != nil {
playlist = &Playlist{
Version:   3,
Sequence:  0,
TargetDur: 6,
IsLive:    true,
}
}

playlist.Segments = append(playlist.Segments, newSegment)

if len(playlist.Segments) > pm.MaxSegments {
playlist.Segments = playlist.Segments[1:]
playlist.Sequence++
}

return pm.WritePlaylist(contentID, quality, playlist)
}

func (pm *PlaylistManager) parseExistingPlaylist(playlistPath string) (*Playlist, error) {
content, err := os.ReadFile(playlistPath)
if err != nil {
return nil, err
}

lines := strings.Split(string(content), "\n")
playlist := &Playlist{
Version:   3,
Sequence:  0,
TargetDur: 6,
IsLive:    true,
}

var currentDuration float64
for _, line := range lines {
line = strings.TrimSpace(line)
if line == "" {
continue
}

if strings.HasPrefix(line, "#EXT-X-VERSION:") {
version, _ := strconv.Atoi(strings.TrimPrefix(line, "#EXT-X-VERSION:"))
playlist.Version = version
} else if strings.HasPrefix(line, "#EXT-X-TARGETDURATION:") {
targetDur, _ := strconv.Atoi(strings.TrimPrefix(line, "#EXT-X-TARGETDURATION:"))
playlist.TargetDur = targetDur
} else if strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:") {
sequence, _ := strconv.Atoi(strings.TrimPrefix(line, "#EXT-X-MEDIA-SEQUENCE:"))
playlist.Sequence = sequence
} else if strings.HasPrefix(line, "#EXTINF:") {
durStr := strings.TrimPrefix(line, "#EXTINF:")
durStr = strings.TrimSuffix(durStr, ",")
currentDuration, _ = strconv.ParseFloat(durStr, 64)
} else if strings.HasPrefix(line, "#EXT-X-ENDLIST") {
playlist.HasEndList = true
playlist.IsLive = false
} else if !strings.HasPrefix(line, "#") && line != "" {
segment := Segment{
Index:    len(playlist.Segments),
Duration: currentDuration,
Filename: line,
URI:      line,
}
playlist.Segments = append(playlist.Segments, segment)
}
}

return playlist, nil
}

func (pm *PlaylistManager) GenerateVariantPlaylist(contentID string, qualities []QualityProfile) error {
masterDir := filepath.Join(pm.WorkDir, contentID)
if err := os.MkdirAll(masterDir, 0755); err != nil {
return err
}

masterPath := filepath.Join(masterDir, "master.m3u8")
file, err := os.Create(masterPath)
if err != nil {
return err
}
defer file.Close()

var builder strings.Builder
builder.WriteString("#EXTM3U\n")
builder.WriteString("#EXT-X-VERSION:6\n")

for _, quality := range qualities {
bandwidth := (quality.VideoBitrate + quality.AudioBitrate) * 1000

builder.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%s,FRAME-RATE=%d\n",
bandwidth, quality.Resolution, quality.FPS))
builder.WriteString(fmt.Sprintf("%s/playlist.m3u8\n", quality.Name))
}

_, err = file.WriteString(builder.String())
return err
}

func (pm *PlaylistManager) CleanupOldSegments(contentID, quality string, keepCount int) error {
segmentDir := filepath.Join(pm.WorkDir, contentID, quality)
pattern := filepath.Join(segmentDir, "segment_*.ts")

files, err := filepath.Glob(pattern)
if err != nil {
return err
}

if len(files) <= keepCount {
return nil
}

sort.Strings(files)
toDelete := files[:len(files)-keepCount]

for _, file := range toDelete {
if err := os.Remove(file); err != nil {
fmt.Printf("Failed to remove old segment %s: %v\n", file, err)
}
}

return nil
}

