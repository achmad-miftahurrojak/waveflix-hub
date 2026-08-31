#!/bin/bash
# Manual scaling script for WaveFlix Hub services

set -e

# Default values
SERVICE="backend"
REPLICAS=3
ACTION="info"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--service)
            SERVICE="$2"
            shift 2
            ;;
        -r|--replicas)
            REPLICAS="$2"
            shift 2
            ;;
        -a|--action)
            ACTION="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  -s, --service    Service to scale (default: backend)"
            echo "  -r, --replicas   Number of replicas (default: 3)"
            echo "  -a, --action     Action: scale|info|status (default: info)"
            echo "  -h, --help       Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option $1"
            exit 1
            ;;
    esac
done

echo "🎯 WaveFlix Hub Scaling Management"
echo "================================="

case $ACTION in
    "info")
        echo "📊 Current service status:"
        docker-compose ps
        
        echo ""
        echo "📈 Resource usage:"
        docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}"
        ;;
        
    "scale")
        echo "⚡ Scaling $SERVICE to $REPLICAS replicas..."
        
        # Validate service exists
        if ! docker-compose config --services | grep -q "^$SERVICE$"; then
            echo "❌ Service '$SERVICE' not found in docker-compose.yml"
            echo "Available services:"
            docker-compose config --services
            exit 1
        fi
        
        # Perform scaling
        docker-compose up -d --scale "$SERVICE=$REPLICAS" --no-recreate
        
        echo "✅ Scaling completed!"
        echo "📊 Updated status:"
        docker-compose ps "$SERVICE"
        ;;
        
    "status")
        echo "🩺 Health check status:"
        
        # Check backend health
        backend_containers=$(docker-compose ps -q backend)
        healthy_count=0
        total_count=0
        
        for container in $backend_containers; do
            total_count=$((total_count + 1))
            if docker exec "$container" wget --quiet --tries=1 --spider http://localhost:8080/health 2>/dev/null; then
                healthy_count=$((healthy_count + 1))
                echo "✅ Backend container $container: healthy"
            else
                echo "❌ Backend container $container: unhealthy"
            fi
        done
        
        echo "📊 Health Summary: $healthy_count/$total_count containers healthy"
        
        # Check load distribution
        echo ""
        echo "🌐 Load balancer status:"
        if docker-compose ps nginx | grep -q "Up"; then
            echo "✅ Nginx load balancer is running"
            # Show nginx status if available
            if docker-compose exec nginx nginx -T 2>/dev/null | grep -q upstream; then
                echo "📋 Upstream configuration detected"
            fi
        else
            echo "❌ Nginx load balancer is not running"
        fi
        ;;
        
    *)
        echo "❌ Unknown action: $ACTION"
        echo "Available actions: scale, info, status"
        exit 1
        ;;
esac

echo ""
echo "💡 Quick scaling commands:"
echo "  Scale up:   $0 --action scale --service backend --replicas 5"
echo "  Scale down: $0 --action scale --service backend --replicas 2"
echo "  Status:     $0 --action status"