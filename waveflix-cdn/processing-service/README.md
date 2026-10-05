# WaveFlix Processing Service

The Processing Service is a core component of the WaveFlix CDN that manages job queues for video transcoding, thumbnail generation, and content analysis with priority-based scheduling and automatic retry mechanisms.

## Features

- **Priority Queue System**: Priority-based job scheduling with configurable retry logic
- **Multi-Worker Architecture**: Concurrent processing with transcoding and thumbnail workers
- **Auto-Retry Mechanism**: Exponential backoff retry with configurable max attempts
- **Batch Processing**: Support for batch job submission and processing
- **Progress Monitoring**: Real-time job progress tracking and status updates
- **Redis Persistence**: Optional Redis backend for job persistence and recovery
- **Metrics & Monitoring**: Comprehensive metrics and health endpoints

## Architecture

```
+-----------------+    +------------------+    +-----------------+
¦    HTTP API     ¦---?¦  Queue Manager   ¦---?¦  Worker Pool    ¦
+-----------------+    +------------------+    +-----------------+
                              ¦                          ¦
                              ?                          ?
                       +------------------+    +-----------------+
                       ¦  Redis Store     ¦    ¦  FFmpeg Engine  ¦
                       +------------------+    +-----------------+
```

## Job Types

| Type | Description | Worker | Priority Support |
|------|-------------|--------|------------------|
| transcode | Video transcoding to multiple formats | TranscodingWorker | ? |
| segment | HLS segmentation | TranscodingWorker | ? |
| thumbnail | Thumbnail generation | ThumbnailWorker | ? |
| analyze | Video analysis | AnalyzeWorker | ? |

## Priority Levels

- **Critical (3)**: Urgent processing, highest priority
- **High (2)**: Important jobs, processed before normal
- **Normal (1)**: Default priority for most jobs  
- **Low (0)**: Background jobs, lowest priority

## API Endpoints

### Job Management
```http
POST /api/v1/jobs
POST /api/v1/jobs/batch
GET /api/v1/jobs/{job_id}
GET /api/v1/jobs/{job_id}/status
```

### Queue Management
```http
GET /api/v1/queue/status
GET /api/v1/queue/metrics
```

### Specialized Endpoints
```http
POST /api/v1/transcode
POST /api/v1/thumbnail
```

### Health & Monitoring
```http
GET /health
GET /ready
```

## Job Submission Examples

### Transcoding Job
```json
{
  "type": "transcode",
  "priority": 1,
  "payload": {
    "input_path": "/content/movie.mp4",
    "output_path": "/processed/movie",
    "content_id": "movie_123",
    "profile": "720p",
    "format": "hls"
  },
  "max_retries": 3,
  "retry_delay": "30s"
}
```

### Batch Job Submission
```json
{
  "jobs": [
    {
      "type": "transcode",
      "priority": 2,
      "payload": {
        "input_path": "/content/movie1.mp4",
        "profile": "1080p"
      }
    },
    {
      "type": "thumbnail",
      "priority": 1,
      "payload": {
        "input_path": "/content/movie1.mp4",
        "content_id": "movie1"
      }
    }
  ]
}
```

## Configuration

### Environment Variables
| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8081 | HTTP server port |
| WORK_DIR | /tmp/processing | Working directory for processing |
| FFMPEG_PATH | ffmpeg | Path to FFmpeg binary |
| MAX_CONCURRENT_JOBS | 5 | Max concurrent jobs per service |
| REDIS_URL | localhost:6379 | Redis connection string |

### Retry Configuration
- **Max Retries**: 3 attempts by default
- **Initial Delay**: 30 seconds
- **Max Delay**: 300 seconds  
- **Backoff Factor**: 2.0 (exponential)

## Queue Management

### Priority Scheduling
Jobs are processed based on:
1. Priority level (Critical ? High ? Normal ? Low)
2. Creation time (FIFO within same priority)

### Retry Logic
Failed jobs are automatically retried with:
- Exponential backoff delay
- Configurable max retry attempts
- Permanent failure after max retries exceeded

### Concurrency Control
- Max concurrent jobs per worker type
- Global concurrency limits
- Worker availability checking

## Workers

### Transcoding Worker
- **Capabilities**: Video transcoding, HLS segmentation
- **Formats**: HLS, MP4, WebM
- **Quality Profiles**: 240p-4K with hardware acceleration
- **Progress Monitoring**: Real-time FFmpeg progress parsing

### Thumbnail Worker  
- **Capabilities**: Thumbnail extraction at multiple timestamps
- **Formats**: JPEG thumbnails
- **Timestamps**: Configurable extraction points
- **Batch Processing**: Multiple thumbnails per video

## Monitoring

### Health Endpoints
- `/health`: Basic service health check
- `/ready`: Readiness check with queue status validation

### Metrics
```json
{
  "queue_length": 15,
  "running_jobs": 5,
  "total_jobs": 1250,
  "active_workers": 2,
  "jobs_completed": 1200,
  "jobs_failed": 35,
  "avg_duration_ms": 45000
}
```

## Deployment

### Local Development
```bash
export WORK_DIR="/tmp/processing"
export FFMPEG_PATH="ffmpeg"
go run ./cmd
```

### Docker
```bash
docker build -t waveflix/processing-service:latest .
docker run -p 8081:8081 -v /tmp/processing:/tmp/processing waveflix/processing-service:latest
```

### Kubernetes
```bash
kubectl apply -f deployments/k8s.yaml
```

## Performance

- **Throughput**: 5-50 concurrent jobs (auto-scaling)
- **Auto-scaling**: CPU/Memory based HPA (75%/85% thresholds)
- **Pod Distribution**: Anti-affinity for optimal resource usage
- **Disruption Budget**: Minimum 3 replicas for availability

## Integration

Integrates with:
- **HLS Service**: For segment processing and playlist generation
- **Storage Service**: For input/output file management  
- **Content Service**: For metadata updates and status tracking
- **Monitoring**: Prometheus metrics and alerting

## File Structure

```
processing-service/
+-- cmd/                    # Application entrypoint
+-- internal/
¦   +-- queue/             # Queue management and job scheduling
¦   +-- workers/           # Worker implementations
¦   +-- server/            # HTTP API server
+-- deployments/           # Kubernetes manifests
+-- Dockerfile            # Container image definition
```

## Error Handling

- **Validation Errors**: Invalid job payloads return 400 Bad Request
- **Resource Limits**: Worker capacity limits prevent overload
- **Retry Logic**: Automatic retry with exponential backoff
- **Failure Tracking**: Permanent failure after max retries
- **Dead Letter Queue**: Failed jobs stored for analysis
