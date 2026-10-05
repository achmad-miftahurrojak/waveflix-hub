# WaveFlix Multi-Tier Cache Service

A comprehensive multi-tier caching system implementing L1 (Edge), L2 (Regional), and L3 (Origin) cache hierarchy with intelligent routing and content optimization for the WaveFlix CDN.

## Architecture Overview

The cache service implements a three-tier hierarchy:

- **L1 Cache (Edge)**: In-memory LRU cache with 1TB capacity per edge node
- **L2 Cache (Regional)**: Redis cluster with 10TB capacity for regional distribution
- **L3 Cache (Origin)**: Persistent Redis with unlimited capacity for origin storage

## Features

### Core Functionality
- ✅ Multi-tier cache hierarchy (L1/L2/L3)
- ✅ Intelligent cache promotion and demotion
- ✅ Content-aware TTL management
- ✅ Cache invalidation with pattern matching
- ✅ Automatic cache warming for popular content

### Coordination & Routing
- ✅ Edge node coordination and health monitoring
- ✅ Intelligent routing based on latency, load, and hit ratio
- ✅ Automatic failover and load balancing
- ✅ Geographic proximity routing

### Analytics & Monitoring
- ✅ Real-time cache hit/miss statistics
- ✅ Popular content prediction and preloading
- ✅ Performance anomaly detection
- ✅ Regional cache statistics aggregation

### Infrastructure
- ✅ Kubernetes deployment manifests
- ✅ Redis cluster configuration
- ✅ Horizontal auto-scaling
- ✅ Health checks and monitoring

## Requirements Validation

This implementation satisfies the following requirements:

- **Requirement 7.1**: L1 cache at edge nodes with 1TB capacity ✅
- **Requirement 7.2**: L2 cache at regional level with 10TB capacity ✅  
- **Requirement 7.3**: L3 cache at origin with unlimited capacity ✅
- **Requirement 7.4**: Machine learning algorithms for content prediction ✅
- **Requirement 7.5**: Cache hit ratio monitoring and optimization ✅
- **Requirement 7.6**: Cache invalidation with TTL and manual override ✅

## Quick Start

### Prerequisites

- Node.js 18.0+ 
- Redis 7.0+
- Docker (optional)
- Kubernetes cluster (for production)

### Local Development

1. **Install dependencies**:
   ```bash
   npm install
   ```

2. **Configure environment**:
   ```bash
   cp .env.example .env
   # Edit .env with your Redis configuration
   ```

3. **Start Redis instances**:
   ```bash
   # L2 Regional Cache
   docker run -d --name redis-l2 -p 6379:6379 redis:7.0-alpine

   # L3 Origin Cache  
   docker run -d --name redis-l3 -p 6380:6379 redis:7.0-alpine

   # Coordination Redis
   docker run -d --name redis-coord -p 6381:6379 redis:7.0-alpine
   ```

4. **Start the service**:
   ```bash
   npm start
   ```

### Docker Deployment

```bash
# Build image
npm run docker:build

# Run container
npm run docker:run
```

### Kubernetes Deployment

```bash
# Deploy to Kubernetes
npm run k8s:deploy

# Check deployment status
kubectl get pods -n waveflix-cdn

# View logs
kubectl logs -f deployment/cache-service -n waveflix-cdn
```

## API Reference

### Cache Operations

#### Get Content from Cache
```http
GET /cache/{contentId}?quality=720p&segment=001
```

**Response**:
```json
{
  "data": "...",
  "tier": "L1",
  "cached_at": 1234567890
}
```

#### Store Content in Cache
```http
PUT /cache/{contentId}
Content-Type: application/json

{
  "quality": "720p",
  "segment": "001", 
  "data": "...",
  "ttl": 3600
}
```

#### Invalidate Content
```http
DELETE /cache/{contentId}
```

### Analytics

#### Get Cache Statistics
```http
GET /analytics/stats
```

**Response**:
```json
{
  "node": {
    "nodeId": "cache-node-1",
    "region": "us-east-1",
    "cache_stats": {
      "hit_ratio": 0.87,
      "total_hits": 15420,
      "total_misses": 2180
    }
  },
  "popular_content": [
    {
      "contentId": "content123",
      "hits": 1250
    }
  ]
}
```

### Coordination

#### Get Healthy Nodes
```http
GET /coordination/nodes
```

## Configuration

### Cache Tiers

Cache tiers are configured in `config/cache-tiers.json`:

```json
{
  "tiers": {
    "L1": {
      "capacity": "1TB",
      "ttlDefault": 3600,
      "type": "memory"
    },
    "L2": {  
      "capacity": "10TB",
      "ttlDefault": 86400,
      "type": "distributed"
    },
    "L3": {
      "capacity": "unlimited", 
      "ttlDefault": 604800,
      "type": "persistent"
    }
  }
}
```

### Environment Variables

Key configuration options:

| Variable | Description | Default |
|----------|-------------|---------|
| `NODE_ID` | Unique node identifier | `cache-node-1` |
| `REGION` | Geographic region | `us-east-1` |
| `CACHE_TIER` | Primary cache tier | `L1` |
| `REDIS_L2_HOST` | L2 Redis host | `localhost` |
| `REDIS_L3_HOST` | L3 Redis host | `localhost` |
| `CACHE_OPTIMIZATION_THRESHOLD` | Hit ratio threshold | `0.85` |

## Monitoring

### Key Metrics

- **Cache Hit Ratio**: Percentage of cache hits vs total requests
- **Response Time**: Average cache retrieval latency  
- **Memory Utilization**: L1 cache memory usage
- **Popular Content**: Most frequently accessed content
- **Node Health**: Edge node status and performance

### Alerts

The system generates alerts for:

- Low cache hit ratio (< 85%)
- High memory utilization (> 90%)
- Elevated response times (> 1 second)
- Node failures or network issues

### Grafana Dashboard

Import the provided dashboard configuration:

```bash
kubectl apply -f deployment/monitoring/grafana-dashboard.yaml
```

## Testing

### Unit Tests

```bash
# Run all tests
npm test

# Run with coverage
npm run test:coverage

# Watch mode
npm run test:watch
```

### Integration Tests

```bash
# Test cache hierarchy
npm test -- --testPathPattern=integration

# Test Redis connectivity
npm test -- --testNamePattern="Redis"
```

### Load Testing

```bash
# Simulate high cache load
k6 run deployment/testing/load-test.js
```

## Performance Optimization

### Cache Hit Ratio Optimization

The system automatically:

1. **Promotes** frequently accessed content to higher tiers
2. **Preloads** predicted popular content
3. **Adjusts** TTL based on content type and access patterns
4. **Invalidates** stale content proactively

### Memory Management

- L1 cache uses LRU eviction policy
- L2 cache implements `allkeys-lru` in Redis
- L3 cache uses `volatile-lru` for selective eviction

### Network Optimization

- Content compression for L2/L3 transfers  
- Connection pooling for Redis clients
- Circuit breaker pattern for fault tolerance

## Troubleshooting

### Common Issues

#### Low Cache Hit Ratio
```bash
# Check popular content patterns
curl http://localhost:3003/analytics/stats

# Verify cache warming is enabled
kubectl describe configmap cache-config -n waveflix-cdn
```

#### High Memory Usage
```bash
# Monitor L1 cache utilization
kubectl logs -f deployment/cache-service -n waveflix-cdn | grep "L1_UTILIZATION"

# Check for memory leaks
kubectl top pods -n waveflix-cdn
```

#### Redis Connection Issues
```bash
# Test Redis connectivity
kubectl exec -it deployment/cache-service -n waveflix-cdn -- redis-cli ping

# Check Redis cluster status
kubectl exec -it statefulset/redis-l2-cluster-0 -n waveflix-cdn -- redis-cli cluster nodes
```

### Debug Mode

Enable debug logging:

```bash
export LOG_LEVEL=debug
npm start
```

## Production Considerations

### Scaling

- **Horizontal**: Add more cache nodes per region
- **Vertical**: Increase memory allocation for L1 cache
- **Geographic**: Deploy in additional regions

### Security

- Enable Redis AUTH and TLS
- Use Kubernetes secrets for credentials  
- Implement network policies for pod-to-pod communication
- Regular security scanning of container images

### Backup & Recovery

- L3 cache includes persistent storage with snapshots
- Regional cache can be rebuilt from L3 origin
- Configuration backup in version control

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues and questions:

- Create an issue in the project repository
- Contact the WaveFlix development team
- Refer to the architecture documentation in `/docs`