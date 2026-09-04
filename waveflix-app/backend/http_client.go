

package main

import (
"crypto/tls"
"log"
"net"
"net/http"
"sync"
"time"
)

type OptimizedHTTPClient struct {
client     *http.Client
transport  *http.Transport
stats      *HTTPClientStats
statsMu    sync.RWMutex
}

type HTTPClientStats struct {
TotalRequests     int64         `json:"total_requests"`
ActiveRequests    int64         `json:"active_requests"`
SuccessfulRequests int64        `json:"successful_requests"`
FailedRequests    int64         `json:"failed_requests"`
AverageLatency    time.Duration `json:"average_latency_ms"`
ConnectionsActive int           `json:"connections_active"`
ConnectionsIdle   int           `json:"connections_idle"`
}

func NewOptimizedHTTPClient() *OptimizedHTTPClient {

dialer := &net.Dialer{
Timeout:   10 * time.Second, 
KeepAlive: 30 * time.Second, 
DualStack: true,             
}

transport := &http.Transport{
DialContext:            dialer.DialContext,
MaxIdleConns:          200,             
MaxIdleConnsPerHost:   50,              
MaxConnsPerHost:       100,             
IdleConnTimeout:       90 * time.Second, 
TLSHandshakeTimeout:   10 * time.Second, 
ExpectContinueTimeout: 1 * time.Second,  

TLSClientConfig: &tls.Config{
InsecureSkipVerify: false,
ClientSessionCache: tls.NewLRUClientSessionCache(128), 
MinVersion:         tls.VersionTLS12,                  
},

ForceAttemptHTTP2: true,

DisableCompression: false, 
DisableKeepAlives:  false, 
}

client := &http.Client{
Transport: transport,
Timeout:   30 * time.Second, 

CheckRedirect: func(req *http.Request, via []*http.Request) error {
if len(via) >= 10 {
return http.ErrUseLastResponse
}
return nil
},
}

return &OptimizedHTTPClient{
client:    client,
transport: transport,
stats:     &HTTPClientStats{},
}
}

func (ohc *OptimizedHTTPClient) Get(url string) (*http.Response, error) {
return ohc.DoWithMetrics("GET", url, nil)
}

func (ohc *OptimizedHTTPClient) DoWithMetrics(method, url string, body interface{}) (*http.Response, error) {
start := time.Now()

ohc.statsMu.Lock()
ohc.stats.TotalRequests++
ohc.stats.ActiveRequests++
ohc.statsMu.Unlock()

defer func() {
ohc.statsMu.Lock()
ohc.stats.ActiveRequests--

latency := time.Since(start)
if ohc.stats.TotalRequests == 1 {
ohc.stats.AverageLatency = latency
} else {

alpha := 0.1
ohc.stats.AverageLatency = time.Duration(
float64(ohc.stats.AverageLatency)*(1-alpha) + float64(latency)*alpha,
)
}
ohc.statsMu.Unlock()
}()

req, err := http.NewRequest(method, url, nil)
if err != nil {
ohc.recordFailure()
return nil, err
}

req.Header.Set("User-Agent", "WaveFlixHub/2.0 (+https://github.com/hamin-baek/waveflix-hub)")
req.Header.Set("Accept-Encoding", "gzip, deflate")
req.Header.Set("Connection", "keep-alive")

resp, err := ohc.client.Do(req)
if err != nil {
ohc.recordFailure()
return nil, err
}

if resp.StatusCode >= 200 && resp.StatusCode < 400 {
ohc.recordSuccess()
} else {
ohc.recordFailure()
}

return resp, nil
}

func (ohc *OptimizedHTTPClient) GetStats() *HTTPClientStats {
ohc.statsMu.RLock()
defer ohc.statsMu.RUnlock()

statsCopy := *ohc.stats
return &statsCopy
}

func (ohc *OptimizedHTTPClient) GetConnectionStats() map[string]interface{} {

return map[string]interface{}{
"max_idle_conns":          ohc.transport.MaxIdleConns,
"max_idle_conns_per_host": ohc.transport.MaxIdleConnsPerHost,
"max_conns_per_host":      ohc.transport.MaxConnsPerHost,
"idle_conn_timeout":       ohc.transport.IdleConnTimeout,
"tls_handshake_timeout":   ohc.transport.TLSHandshakeTimeout,
}
}

func (ohc *OptimizedHTTPClient) recordSuccess() {
ohc.statsMu.Lock()
ohc.stats.SuccessfulRequests++
ohc.statsMu.Unlock()
}

func (ohc *OptimizedHTTPClient) recordFailure() {
ohc.statsMu.Lock()
ohc.stats.FailedRequests++
ohc.statsMu.Unlock()
}

func (ohc *OptimizedHTTPClient) Close() {
if ohc.transport != nil {
ohc.transport.CloseIdleConnections()
}
}

var optimizedHTTPClient *OptimizedHTTPClient
var httpClientOnce sync.Once

func GetOptimizedHTTPClient() *OptimizedHTTPClient {
httpClientOnce.Do(func() {
optimizedHTTPClient = NewOptimizedHTTPClient()
log.Println("[http] Optimized HTTP client initialized with connection pooling")
})
return optimizedHTTPClient
}
