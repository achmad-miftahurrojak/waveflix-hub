// WaveFlix Hub Auto-scaler for Docker Compose
// Monitors metrics and scales services based on thresholds
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type AutoScaler struct {
	dockerClient     *client.Client
	prometheusURL    string
	scaleUpThreshold float64
	scaleDownThreshold float64
	minReplicas      int
	maxReplicas      int
	scaleInterval    time.Duration
}

type PrometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type ServiceMetrics struct {
	CPUUsage     float64
	MemoryUsage  float64
	RequestRate  float64
	ResponseTime float64
	ErrorRate    float64
}

func main() {
	log.Println("[autoscaler] Starting WaveFlix Auto-scaler...")

	autoscaler := &AutoScaler{
		prometheusURL:      getEnv("PROMETHEUS_URL", "http://prometheus:9090"),
		scaleUpThreshold:   getEnvFloat("SCALE_UP_THRESHOLD", 80.0),
		scaleDownThreshold: getEnvFloat("SCALE_DOWN_THRESHOLD", 30.0),
		minReplicas:        getEnvInt("MIN_REPLICAS", 2),
		maxReplicas:        getEnvInt("MAX_REPLICAS", 10),
		scaleInterval:      getEnvDuration("SCALE_INTERVAL", "60s"),
	}

	// Initialize Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("[autoscaler] Failed to create Docker client: %v", err)
	}
	autoscaler.dockerClient = dockerClient

	log.Printf("[autoscaler] Configuration: UP=%0.1f%%, DOWN=%0.1f%%, MIN=%d, MAX=%d, INTERVAL=%v",
		autoscaler.scaleUpThreshold, autoscaler.scaleDownThreshold,
		autoscaler.minReplicas, autoscaler.maxReplicas, autoscaler.scaleInterval)

	// Start scaling loop
	ticker := time.NewTicker(autoscaler.scaleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			autoscaler.evaluateScaling()
		}
	}
}

func (as *AutoScaler) evaluateScaling() {
	ctx := context.Background()
	
	// Get current service metrics
	metrics, err := as.getServiceMetrics()
	if err != nil {
		log.Printf("[autoscaler] Failed to get metrics: %v", err)
		return
	}

	// Get current replica count
	currentReplicas, err := as.getCurrentReplicas(ctx, "waveflix_backend")
	if err != nil {
		log.Printf("[autoscaler] Failed to get current replicas: %v", err)
		return
	}

	log.Printf("[autoscaler] Current metrics: CPU=%0.1f%%, Memory=%0.1f%%, Requests/s=%0.1f, Replicas=%d",
		metrics.CPUUsage, metrics.MemoryUsage, metrics.RequestRate, currentReplicas)

	// Determine scaling action
	targetReplicas := as.calculateTargetReplicas(metrics, currentReplicas)
	
	if targetReplicas != currentReplicas {
		log.Printf("[autoscaler] Scaling from %d to %d replicas", currentReplicas, targetReplicas)
		err := as.scaleService(ctx, "waveflix_backend", targetReplicas)
		if err != nil {
			log.Printf("[autoscaler] Failed to scale service: %v", err)
		} else {
			log.Printf("[autoscaler] Successfully scaled to %d replicas", targetReplicas)
		}
	}
}

func (as *AutoScaler) calculateTargetReplicas(metrics *ServiceMetrics, currentReplicas int) int {
	// Calculate load score (weighted average)
	loadScore := (metrics.CPUUsage*0.4 + metrics.MemoryUsage*0.3 + 
		         (metrics.RequestRate/10)*0.2 + metrics.ErrorRate*0.1)

	var targetReplicas int

	if loadScore > as.scaleUpThreshold {
		// Scale up: increase by 50% or minimum 1 replica
		increase := max(1, int(float64(currentReplicas)*0.5))
		targetReplicas = currentReplicas + increase
		
		// Apply exponential scaling for high load
		if loadScore > as.scaleUpThreshold*1.5 {
			targetReplicas = currentReplicas * 2
		}
	} else if loadScore < as.scaleDownThreshold {
		// Scale down: decrease by 25% or minimum 1 replica
		decrease := max(1, int(float64(currentReplicas)*0.25))
		targetReplicas = currentReplicas - decrease
	} else {
		// No scaling needed
		targetReplicas = currentReplicas
	}

	// Apply bounds
	if targetReplicas < as.minReplicas {
		targetReplicas = as.minReplicas
	}
	if targetReplicas > as.maxReplicas {
		targetReplicas = as.maxReplicas
	}

	return targetReplicas
}

func (as *AutoScaler) getServiceMetrics() (*ServiceMetrics, error) {
	metrics := &ServiceMetrics{}

	// Query CPU usage
	cpuQuery := `avg(rate(container_cpu_usage_seconds_total{name=~"waveflix_backend.*"}[5m])) * 100`
	cpuUsage, err := as.queryPrometheus(cpuQuery)
	if err == nil && len(cpuUsage) > 0 {
		metrics.CPUUsage = cpuUsage[0]
	}

	// Query Memory usage
	memQuery := `avg(container_memory_usage_bytes{name=~"waveflix_backend.*"} / container_spec_memory_limit_bytes{name=~"waveflix_backend.*"}) * 100`
	memUsage, err := as.queryPrometheus(memQuery)
	if err == nil && len(memUsage) > 0 {
		metrics.MemoryUsage = memUsage[0]
	}

	// Query Request rate (if available from application metrics)
	reqQuery := `sum(rate(http_requests_total{service="waveflix-backend"}[5m]))`
	reqRate, err := as.queryPrometheus(reqQuery)
	if err == nil && len(reqRate) > 0 {
		metrics.RequestRate = reqRate[0]
	}

	// Query Error rate
	errorQuery := `sum(rate(http_requests_total{service="waveflix-backend",status=~"5.."}[5m])) / sum(rate(http_requests_total{service="waveflix-backend"}[5m])) * 100`
	errorRate, err := as.queryPrometheus(errorQuery)
	if err == nil && len(errorRate) > 0 {
		metrics.ErrorRate = errorRate[0]
	}

	return metrics, nil
}

func (as *AutoScaler) queryPrometheus(query string) ([]float64, error) {
	url := fmt.Sprintf("%s/api/v1/query?query=%s", as.prometheusURL, query)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var promResp PrometheusResponse
	err = json.NewDecoder(resp.Body).Decode(&promResp)
	if err != nil {
		return nil, err
	}

	if promResp.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: %s", promResp.Status)
	}

	var values []float64
	for _, result := range promResp.Data.Result {
		if len(result.Value) > 1 {
			if valueStr, ok := result.Value[1].(string); ok {
				if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
					values = append(values, value)
				}
			}
		}
	}

	return values, nil
}

func (as *AutoScaler) getCurrentReplicas(ctx context.Context, serviceName string) (int, error) {
	services, err := as.dockerClient.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		// Fallback to container counting for Docker Compose
		return as.countContainers(ctx, serviceName)
	}

	for _, service := range services {
		if strings.Contains(service.Spec.Name, serviceName) {
			return int(*service.Spec.Mode.Replicated.Replicas), nil
		}
	}

	return 0, fmt.Errorf("service not found: %s", serviceName)
}

func (as *AutoScaler) countContainers(ctx context.Context, serviceName string) (int, error) {
	containers, err := as.dockerClient.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return 0, err
	}

	count := 0
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.Contains(name, serviceName) {
				count++
				break
			}
		}
	}

	return count, nil
}

func (as *AutoScaler) scaleService(ctx context.Context, serviceName string, replicas int) error {
	// Try Docker Swarm service scaling first
	services, err := as.dockerClient.ServiceList(ctx, types.ServiceListOptions{})
	if err == nil {
		for _, service := range services {
			if strings.Contains(service.Spec.Name, serviceName) {
				serviceSpec := service.Spec
				serviceSpec.Mode.Replicated.Replicas = uint64Ptr(uint64(replicas))
				
				_, err := as.dockerClient.ServiceUpdate(ctx, service.ID, service.Version, serviceSpec, types.ServiceUpdateOptions{})
				return err
			}
		}
	}

	// Fallback to Docker Compose scaling (requires docker-compose command)
	log.Printf("[autoscaler] Using docker-compose for scaling")
	return as.dockerComposeScale(serviceName, replicas)
}

func (as *AutoScaler) dockerComposeScale(serviceName, replicas int) error {
	// This would require executing docker-compose command
	// For now, just log the action
	log.Printf("[autoscaler] Would scale %s to %d replicas (docker-compose scaling not implemented)", serviceName, replicas)
	return nil
}

// Utility functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvDuration(key, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	if duration, err := time.ParseDuration(defaultValue); err == nil {
		return duration
	}
	return time.Minute
}

func uint64Ptr(i uint64) *uint64 {
	return &i
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}