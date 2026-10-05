# WaveFlix Storage Service

The Storage Service provides distributed file storage with intelligent tiering, automated migration, and comprehensive analytics for the WaveFlix CDN. It implements a three-tier storage hierarchy optimized for access patterns and cost efficiency.

## Features

- **Three-Tier Storage**: Hot, Warm, and Cold tiers with different performance characteristics
- **Automated Migration**: Rule-based migration between tiers based on access patterns and age
- **Content Deduplication**: MD5-based deduplication to optimize storage efficiency
- **Access Analytics**: Detailed analytics on file access patterns and frequency
- **Replication Management**: Configurable cross-tier replication for high availability
- **RESTful API**: Complete HTTP API for file operations and storage management

## Architecture

```
+-----------------+    +------------------+    +-----------------+
¦   Client API    ¦---?¦ Tiered Storage   ¦---?¦  Storage Tiers  ¦
+-----------------+    +------------------+    +-----------------+
                              ¦                          ¦
                              ?                          ?
                       +------------------+    +-----------------+
                       ¦ Metadata Store   ¦    ¦ Migration Svc   ¦
                       +------------------+    +-----------------+
                              ¦
                              ?
                       +------------------+
                       ¦ Access Analytics ¦
                       +------------------+
```

## Storage Tiers

| Tier | Performance | Capacity | Use Case | Auto-Migration |
|------|-------------|----------|----------|----------------|
| **Hot** | <1ms latency | 1TB | Frequently accessed content | Files accessed <24h ago |
| **Warm** | <10ms latency | 10TB | Moderately accessed content | Files with moderate access |
| **Cold** | <100ms latency | 100TB | Rarely accessed content | Files accessed <7 days ago |

## Migration Rules

### Default Migration Policies
1. **Hot ? Warm**: Files older than 24 hours with <5 accesses
2. **Warm ? Cold**: Files older than 7 days with <2 accesses  
3. **Cold ? Hot**: Files with >50 accesses (promotion for viral content)

### Custom Rules Support
- Access threshold-based migration
- Age-based migration policies
- Size-based migration rules
- Configurable migration intervals

## API Endpoints

### File Operations
```http
POST /api/v1/files
GET /api/v1/files/{key}
DELETE /api/v1/files/{key}
POST /api/v1/files/{key}/move
GET /api/v1/files/{key}/info
```

### Storage Management
```http
GET /api/v1/files?prefix={prefix}&tier={tier}
GET /api/v1/storage/stats
GET /api/v1/storage/tiers
POST /api/v1/storage/migration/start
GET /api/v1/storage/migration/stats
```

### Analytics
```http
GET /api/v1/analytics/access
GET /api/v1/analytics/top?limit={n}
```

### Health & Monitoring
```http
GET /health
GET /ready
```

## Usage Examples

### Upload File
```bash
curl -X POST http://storage-service/api/v1/files \
  -H "X-File-Key: content/movie_123/video.mp4" \
  -H "Content-Type: video/mp4" \
  --data-binary @video.mp4
```

### Download File
```bash
curl -X GET http://storage-service/api/v1/files/content/movie_123/video.mp4 \
  -o downloaded_video.mp4
```

### Move File Between Tiers
```bash
curl -X POST http://storage-service/api/v1/files/content/movie_123/video.mp4/move \
  -H "Content-Type: application/json" \
  -d '{"to_tier": "cold"}'
```

### Get Storage Statistics
```bash
curl -X GET http://storage-service/api/v1/storage/stats
```

## Configuration

### Environment Variables
| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8082 | HTTP server port |
| HOT_TIER_PATH | /storage/hot | Hot tier storage path |
| WARM_TIER_PATH | /storage/warm | Warm tier storage path |
| COLD_TIER_PATH | /storage/cold | Cold tier storage path |

### Storage Tier Configuration
```go
config := &storage.TieredConfig{
    HotTierTTL:  24 * time.Hour,
    WarmTierTTL: 7 * 24 * time.Hour,
    MigrationRules: []storage.MigrationRule{
        {
            FromTier:        storage.HotTier,
            ToTier:          storage.WarmTier,
            AccessThreshold: 5,
            AgeThreshold:    24 * time.Hour,
            Enabled:         true,
        },
    },
    Replication: storage.ReplicationConfig{
        Enabled:     true,
        MinReplicas: 2,
        MaxReplicas: 3,
        CrossTier:   true,
    },
}
```

## Replication & Availability

### Replication Strategy
- **Cross-Tier Replication**: Files replicated across different storage tiers
- **Configurable Replicas**: 2-3 replicas per file for high availability
- **Automatic Recovery**: Missing replicas automatically recreated
- **Consistency**: Strong consistency for metadata, eventual consistency for replicas

### Fault Tolerance
- **Multi-Tier Availability**: Service continues if single tier fails
- **Automatic Failover**: Requests automatically routed to available replicas
- **Data Recovery**: Built-in repair mechanisms for corrupted files
- **Graceful Degradation**: Performance degrades gracefully under load

## Analytics & Insights

### Access Pattern Analytics
- **Hourly Access Tracking**: Detailed hourly access statistics
- **Frequency Analysis**: Access frequency calculations for migration decisions
- **Top Files Reporting**: Most frequently accessed content identification
- **Usage Trends**: Historical usage pattern analysis

### Storage Metrics
```json
{
  "hot": {
    "capacity": 1099511627776,
    "used": 524288000000,
    "available": 575223627776,
    "access_cost": 0.10,
    "latency": "1ms"
  },
  "analytics": {
    "total_files": 15240,
    "total_accesses": 1204500,
    "avg_accesses": 79.1
  }
}
```

## Performance Optimization

### Caching Strategy
- **Tier-Based Cache Headers**: Different TTL based on storage tier
  - Hot Tier: 1 hour cache
  - Warm Tier: 24 hour cache  
  - Cold Tier: 7 day cache

### Access Optimization
- **Predictive Migration**: Promote files based on access patterns
- **Intelligent Prefetching**: Preload likely-to-be-accessed content
- **Load Balancing**: Distribute requests across multiple storage nodes
- **Compression**: On-the-fly compression for bandwidth optimization

## Deployment

### Local Development
```bash
export HOT_TIER_PATH="/tmp/storage/hot"
export WARM_TIER_PATH="/tmp/storage/warm"
export COLD_TIER_PATH="/tmp/storage/cold"
go run ./cmd
```

### Docker
```bash
docker build -t waveflix/storage-service:latest .
docker run -p 8082:8082 \
  -v /storage/hot:/storage/hot \
  -v /storage/warm:/storage/warm \
  -v /storage/cold:/storage/cold \
  waveflix/storage-service:latest
```

### Kubernetes
```bash
kubectl apply -f deployments/k8s.yaml
```

## Storage Classes

### Production Storage Configuration
- **Hot Tier**: NVMe SSD storage class (`fast-ssd`)
- **Warm Tier**: Standard SSD storage class (`standard-ssd`)  
- **Cold Tier**: High-capacity HDD storage class (`cold-hdd`)

### Capacity Planning
- **Hot Tier**: 1TB per node (frequently accessed content)
- **Warm Tier**: 10TB per node (moderately accessed content)
- **Cold Tier**: 100TB per node (archive and backup content)

## Monitoring & Alerting

### Health Checks
- **Service Health**: Basic connectivity and functionality
- **Storage Health**: Tier capacity and availability monitoring
- **Migration Health**: Migration service status and performance

### Metrics Collection
- **Request Metrics**: Request rate, latency, error rate
- **Storage Metrics**: Capacity utilization, tier distribution
- **Migration Metrics**: Files migrated, migration success rate
- **Access Metrics**: Popular content, access patterns

## Integration

Integrates with:
- **HLS Service**: Stores video segments and playlists
- **Processing Service**: Stores transcoded content and thumbnails
- **Content Service**: Provides file metadata and management
- **CDN Edge Nodes**: Serves content with optimized caching

## File Structure

```
storage-service/
+-- cmd/                    # Application entrypoint
+-- internal/
¦   +-- storage/           # Tiered storage implementation
¦   ¦   +-- tiered.go      # Main storage orchestration
¦   ¦   +-- metadata.go    # Metadata store and analytics
¦   ¦   +-- migration.go   # Migration and replication
¦   +-- server/            # HTTP API server
+-- deployments/           # Kubernetes manifests
+-- Dockerfile            # Container image definition
```

## Security

### Access Control
- **API Key Authentication**: Secure access to storage operations
- **Path Traversal Protection**: Prevents unauthorized file access
- **Content Validation**: File type and size validation
- **Audit Logging**: Comprehensive access and operation logging

### Data Protection
- **Encryption at Rest**: Optional encryption for sensitive content
- **Secure Transport**: TLS encryption for all API communications
- **Integrity Checks**: MD5 checksums for data integrity validation
- **Access Logging**: Detailed audit trail for compliance
