# WaveFlix HLS Service

The HLS service is a core component of the WaveFlix CDN that handles video segmentation and HLS playlist generation for adaptive bitrate streaming.

## Features

- **FFmpeg Integration**: Hardware-accelerated video transcoding with multiple quality profiles
- **HLS Segmentation**: Generates 6-second video segments with proper keyframes
- **Adaptive Bitrate**: Supports multiple quality levels (240p-4K) with automatic playlist generation
- **Storage Abstraction**: Pluggable storage backends (local filesystem, S3)
- **HTTP API**: RESTful API for processing jobs and serving HLS content
- **Kubernetes Ready**: Production-ready deployment with auto-scaling

## Architecture

```
+-----------------+    +------------------+    +-----------------+
¦   Client API    ¦---?¦   HLS Service    ¦---?¦  Storage Layer  ¦
+-----------------+    +------------------+    +-----------------+
                              ¦
                              ?
                       +------------------+
                       ¦  FFmpeg Worker   ¦
                       +------------------+
                              ¦
                              ?
                       +------------------+
                       ¦ Playlist Manager ¦
                       +------------------+
```

## Quality Profiles

| Profile | Resolution | Video Bitrate | Audio Bitrate | FPS | Codec   |
|---------|------------|---------------|---------------|-----|---------|
| 240p    | 426x240    | 400 kbps      | 64 kbps       | 24  | H.264   |
| 360p    | 640x360    | 800 kbps      | 96 kbps       | 30  | H.264   |
| 480p    | 854x480    | 1.2 Mbps      | 128 kbps      | 30  | H.264   |
| 720p    | 1280x720   | 2.5 Mbps      | 128 kbps      | 30  | H.264   |
| 1080p   | 1920x1080  | 5 Mbps        | 192 kbps      | 30  | H.264   |
| 4K      | 3840x2160  | 15 Mbps       | 256 kbps      | 30  | H.265   |

## API Endpoints

### Process Video
```http
POST /api/v1/process
Content-Type: application/json

{
  "content_id": "movie_123",
  "input_path": "/content/movie.mp4",
  "qualities": ["720p", "480p", "360p"],
  "priority": 1
}
```

### Get Job Status
```http
GET /api/v1/status/{job_id}
```

### Stream Content
```http
GET /api/v1/content/{content_id}/master.m3u8
GET /api/v1/content/{content_id}/{quality}/playlist.m3u8
GET /api/v1/content/{content_id}/{quality}/segment_{number}.ts
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8080 | HTTP server port |
| HLS_WORK_DIR | /tmp/hls | Working directory for segments |
| FFMPEG_PATH | ffmpeg | Path to FFmpeg binary |

## Deployment

### Local Development
```bash
go mod download
go run ./cmd
```

### Docker
```bash
docker build -t waveflix/hls-service:latest .
docker run -p 8080:8080 -v /tmp/hls:/tmp/hls waveflix/hls-service:latest
```

### Kubernetes
```bash
kubectl apply -f deployments/k8s.yaml
```

## Integration

The HLS service integrates with:
- **Processing Service**: For FFmpeg transcoding jobs
- **Storage Service**: For persistent segment storage
- **Content Service**: For metadata and content management
- **Streaming Service**: For content delivery optimization

## Performance

- **Concurrent Jobs**: Up to 5 parallel transcoding jobs per instance
- **Segment Duration**: 6 seconds for optimal streaming performance
- **Hardware Acceleration**: Supports NVENC, Intel Quick Sync, and AMD VCE
- **Auto-scaling**: Kubernetes HPA based on CPU/Memory usage (70%/80%)

## Monitoring

Health check endpoint: `GET /health`

Metrics are exposed on port 9090 (when enabled):
- Transcoding job metrics
- Segment serving statistics
- Error rates and latency

## File Structure

```
hls-service/
+-- cmd/                    # Application entrypoint
+-- internal/
¦   +-- hls/               # HLS segmentation and playlist logic
¦   +-- storage/           # Storage abstraction layer
¦   +-- server/            # HTTP API server
¦   +-- integration/       # External service clients
+-- configs/               # Configuration files
+-- deployments/           # Kubernetes manifests
+-- Dockerfile            # Container image definition
```

## HLS Compliance

The service generates standard-compliant HLS streams:
- HTTP Live Streaming (HLS) version 3+
- Apple-compatible playlists and segments
- Proper segment timing and keyframe alignment
- Support for seeking and adaptive bitrate switching

## Security

- CORS support for cross-origin requests
- Content-Type validation for uploaded files
- Path traversal protection in file operations
- Resource limits to prevent DoS attacks
