#!/bin/bash
# Production Environment Initialization Script for WaveFlix Hub

set -e
set -o pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ENV_FILE="$PROJECT_ROOT/.env.production"
DEPLOYMENT_TYPE="${DEPLOYMENT_TYPE:-docker}" # docker, kubernetes

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Error handling
trap 'log_error "Script failed at line $LINENO"' ERR

main() {
    log_info "🚀 Initializing WaveFlix Hub Production Environment"
    echo "============================================="
    
    # Check prerequisites
    check_prerequisites
    
    # Load and validate environment
    load_environment
    validate_environment
    
    # Setup based on deployment type
    case $DEPLOYMENT_TYPE in
        "docker")
            setup_docker_production
            ;;
        "kubernetes")
            setup_kubernetes_production
            ;;
        *)
            log_error "Invalid DEPLOYMENT_TYPE: $DEPLOYMENT_TYPE. Use 'docker' or 'kubernetes'"
            exit 1
            ;;
    esac
    
    # Post-setup verification
    verify_deployment
    
    log_success "🎉 Production environment initialized successfully!"
    show_access_information
}

check_prerequisites() {
    log_info "📋 Checking prerequisites..."
    
    local missing_tools=()
    
    # Common tools
    command -v docker >/dev/null 2>&1 || missing_tools+=("docker")
    command -v docker-compose >/dev/null 2>&1 || missing_tools+=("docker-compose")
    command -v curl >/dev/null 2>&1 || missing_tools+=("curl")
    command -v jq >/dev/null 2>&1 || missing_tools+=("jq")
    
    # Kubernetes-specific tools
    if [ "$DEPLOYMENT_TYPE" = "kubernetes" ]; then
        command -v kubectl >/dev/null 2>&1 || missing_tools+=("kubectl")
        command -v helm >/dev/null 2>&1 || missing_tools+=("helm")
    fi
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_info "Please install the missing tools and try again."
        exit 1
    fi
    
    log_success "All prerequisites met"
}

load_environment() {
    log_info "🔧 Loading environment configuration..."
    
    if [ ! -f "$ENV_FILE" ]; then
        log_warning "Environment file not found: $ENV_FILE"
        log_info "Creating from template..."
        
        cp "$PROJECT_ROOT/.env.example" "$ENV_FILE"
        log_warning "Please edit $ENV_FILE with your production values"
        log_info "Required variables: DOMAIN_NAME, POSTGRES_PASSWORD, REDIS_PASSWORD, TMDB_API_KEY, JWT_SECRET"
        exit 1
    fi
    
    # Load environment variables
    set -a  # Export all variables
    source "$ENV_FILE"
    set +a
    
    log_success "Environment loaded from $ENV_FILE"
}

validate_environment() {
    log_info "✅ Validating environment variables..."
    
    local required_vars=(
        "DOMAIN_NAME"
        "POSTGRES_PASSWORD" 
        "REDIS_PASSWORD"
        "TMDB_API_KEY"
        "JWT_SECRET"
    )
    
    local missing_vars=()
    
    for var in "${required_vars[@]}"; do
        if [ -z "${!var}" ]; then
            missing_vars+=("$var")
        fi
    done
    
    if [ ${#missing_vars[@]} -ne 0 ]; then
        log_error "Missing required environment variables: ${missing_vars[*]}"
        log_info "Please set these variables in $ENV_FILE"
        exit 1
    fi
    
    # Validate specific formats
    if [ ${#JWT_SECRET} -lt 32 ]; then
        log_error "JWT_SECRET must be at least 32 characters long"
        exit 1
    fi
    
    if [ ${#POSTGRES_PASSWORD} -lt 12 ]; then
        log_warning "POSTGRES_PASSWORD should be at least 12 characters for security"
    fi
    
    log_success "Environment validation passed"
}

setup_docker_production() {
    log_info "🐳 Setting up Docker production environment..."
    
    # Create necessary directories
    mkdir -p "$PROJECT_ROOT/logs"
    mkdir -p "$PROJECT_ROOT/backups"
    mkdir -p "$PROJECT_ROOT/ssl"
    
    # Generate SSL certificates if not exists
    if [ ! -f "$PROJECT_ROOT/nginx/ssl/waveflix.crt" ]; then
        log_info "Generating SSL certificates..."
        generate_ssl_certificates
    fi
    
    # Build and start services
    log_info "Building Docker images..."
    cd "$PROJECT_ROOT"
    
    # Build images with production target
    docker-compose -f docker-compose.prod.yml build --no-cache
    
    # Start services
    log_info "Starting production services..."
    docker-compose -f docker-compose.prod.yml up -d
    
    # Wait for services to be ready
    wait_for_services_docker
    
    log_success "Docker production environment ready"
}

setup_kubernetes_production() {
    log_info "☸️  Setting up Kubernetes production environment..."
    
    # Check kubectl connection
    if ! kubectl cluster-info >/dev/null 2>&1; then
        log_error "Cannot connect to Kubernetes cluster"
        log_info "Please ensure kubectl is configured correctly"
        exit 1
    fi
    
    # Create namespace
    log_info "Creating namespace..."
    kubectl apply -f "$PROJECT_ROOT/k8s/namespace.yaml"
    
    # Create secrets (prompt for values)
    log_info "Setting up secrets..."
    setup_kubernetes_secrets
    
    # Deploy applications in order
    log_info "Deploying applications..."
    
    # Database and cache
    kubectl apply -f "$PROJECT_ROOT/k8s/postgres-deployment.yaml"
    kubectl apply -f "$PROJECT_ROOT/k8s/redis-deployment.yaml"
    
    # Wait for database to be ready
    wait_for_kubernetes_deployment "postgres"
    wait_for_kubernetes_deployment "redis"
    
    # Application services
    kubectl apply -f "$PROJECT_ROOT/k8s/backend-deployment.yaml"
    kubectl apply -f "$PROJECT_ROOT/k8s/frontend-deployment.yaml"
    
    # Load balancer and ingress
    kubectl apply -f "$PROJECT_ROOT/k8s/nginx-configmap.yaml"
    kubectl apply -f "$PROJECT_ROOT/k8s/ingress.yaml"
    
    # Monitoring
    kubectl apply -f "$PROJECT_ROOT/k8s/monitoring-deployment.yaml"
    
    # Auto-scaling
    kubectl apply -f "$PROJECT_ROOT/k8s/hpa.yaml"
    kubectl apply -f "$PROJECT_ROOT/k8s/vpa.yaml"
    
    # Wait for all deployments
    wait_for_kubernetes_deployment "waveflix-backend"
    wait_for_kubernetes_deployment "waveflix-frontend"
    
    log_success "Kubernetes production environment ready"
}

generate_ssl_certificates() {
    local ssl_dir="$PROJECT_ROOT/nginx/ssl"
    mkdir -p "$ssl_dir"
    
    if command -v openssl >/dev/null 2>&1; then
        log_info "Generating self-signed SSL certificate for $DOMAIN_NAME..."
        
        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout "$ssl_dir/waveflix.key" \
            -out "$ssl_dir/waveflix.crt" \
            -subj "/C=US/ST=State/L=City/O=WaveFlix/CN=$DOMAIN_NAME" \
            -addext "subjectAltName=DNS:$DOMAIN_NAME,DNS:www.$DOMAIN_NAME,DNS:api.$DOMAIN_NAME"
        
        log_success "SSL certificate generated"
        log_warning "Remember to replace with proper SSL certificates in production!"
    else
        log_error "OpenSSL not found. Please install OpenSSL or provide SSL certificates manually"
        exit 1
    fi
}

setup_kubernetes_secrets() {
    # Check if secrets already exist
    if kubectl get secret waveflix-secrets -n waveflix >/dev/null 2>&1; then
        log_info "Secrets already exist, skipping creation"
        return
    fi
    
    log_info "Creating Kubernetes secrets..."
    
    # Encode values to base64
    local postgres_password_b64=$(echo -n "$POSTGRES_PASSWORD" | base64 -w 0)
    local redis_password_b64=$(echo -n "$REDIS_PASSWORD" | base64 -w 0)
    local tmdb_api_key_b64=$(echo -n "$TMDB_API_KEY" | base64 -w 0)
    local jwt_secret_b64=$(echo -n "$JWT_SECRET" | base64 -w 0)
    local database_url_b64=$(echo -n "postgres://waveflix:$POSTGRES_PASSWORD@postgres-service:5432/waveflix?sslmode=require" | base64 -w 0)
    
    # Create secret from template
    cat > /tmp/waveflix-secrets.yaml << EOF
apiVersion: v1
kind: Secret
metadata:
  name: waveflix-secrets
  namespace: waveflix
type: Opaque
data:
  database-url: $database_url_b64
  postgres-password: $postgres_password_b64
  redis-password: $redis_password_b64
  tmdb-api-key: $tmdb_api_key_b64
  jwt-secret: $jwt_secret_b64
EOF
    
    kubectl apply -f /tmp/waveflix-secrets.yaml
    rm /tmp/waveflix-secrets.yaml
    
    log_success "Secrets created"
}

wait_for_services_docker() {
    log_info "⏳ Waiting for services to be ready..."
    
    local max_attempts=60
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if docker-compose -f docker-compose.prod.yml exec -T backend /waveflix-backend -health-check >/dev/null 2>&1; then
            log_success "Backend service is ready"
            break
        fi
        
        attempt=$((attempt + 1))
        echo -n "."
        sleep 5
    done
    
    if [ $attempt -eq $max_attempts ]; then
        log_error "Services failed to start within timeout"
        log_info "Check logs with: docker-compose -f docker-compose.prod.yml logs"
        exit 1
    fi
}

wait_for_kubernetes_deployment() {
    local deployment_name=$1
    log_info "Waiting for deployment $deployment_name to be ready..."
    
    kubectl wait --for=condition=available --timeout=300s deployment/$deployment_name -n waveflix
    
    if [ $? -eq 0 ]; then
        log_success "Deployment $deployment_name is ready"
    else
        log_error "Deployment $deployment_name failed to become ready"
        exit 1
    fi
}

verify_deployment() {
    log_info "🔍 Verifying deployment..."
    
    local health_url
    
    if [ "$DEPLOYMENT_TYPE" = "docker" ]; then
        health_url="https://$DOMAIN_NAME/health"
    else
        # For Kubernetes, we might need to use port-forward or ingress
        health_url="http://localhost:8080/health"  # Adjust based on your setup
    fi
    
    # Test health endpoint
    local attempt=0
    local max_attempts=12
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -f -s "$health_url" >/dev/null 2>&1; then
            log_success "Health check passed"
            break
        fi
        
        attempt=$((attempt + 1))
        log_info "Health check attempt $attempt/$max_attempts..."
        sleep 10
    done
    
    if [ $attempt -eq $max_attempts ]; then
        log_warning "Health check failed, but deployment may still be initializing"
    fi
}

show_access_information() {
    echo ""
    echo "🌐 Access Information:"
    echo "===================="
    echo "Application: https://$DOMAIN_NAME"
    echo "API: https://api.$DOMAIN_NAME"
    
    if [ "$DEPLOYMENT_TYPE" = "docker" ]; then
        echo "Grafana: https://$DOMAIN_NAME:3001 (admin / $GRAFANA_PASSWORD)"
        echo "Prometheus: https://$DOMAIN_NAME:9090"
    else
        echo "Grafana: Use kubectl port-forward service/grafana 3000:3000 -n waveflix"
        echo "Prometheus: Use kubectl port-forward service/prometheus 9090:9090 -n waveflix"
    fi
    
    echo ""
    echo "📖 Useful Commands:"
    echo "==================="
    
    if [ "$DEPLOYMENT_TYPE" = "docker" ]; then
        echo "View logs: docker-compose -f docker-compose.prod.yml logs -f"
        echo "Scale services: docker-compose -f docker-compose.prod.yml up -d --scale backend=5"
        echo "Stop services: docker-compose -f docker-compose.prod.yml down"
    else
        echo "View logs: kubectl logs -f deployment/waveflix-backend -n waveflix"
        echo "Scale services: kubectl scale deployment waveflix-backend --replicas=5 -n waveflix"
        echo "Delete deployment: kubectl delete namespace waveflix"
    fi
    
    echo "Test deployment: $SCRIPT_DIR/test-production.sh"
}

# Run main function
main "$@"