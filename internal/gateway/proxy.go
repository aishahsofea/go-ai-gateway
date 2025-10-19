package gateway

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
)

type Proxy struct {
	config        *GatewayConfig
	loadBalancers map[string]LoadBalancer // Cache load balancers per route
	retryConfig   RetryConfig
	mutex         sync.RWMutex
}

func NewProxy(config *GatewayConfig) *Proxy {
	return &Proxy{
		config:        config,
		loadBalancers: make(map[string]LoadBalancer),
		retryConfig:   DefaultRetryConfig(),
	}
}

type statusTracker struct {
	http.ResponseWriter
	status int
}

func (st *statusTracker) WriteHeader(status int) {
	st.status = status
	st.ResponseWriter.WriteHeader(status)
}

type bufferingResponseWriter struct {
	status int
	header http.Header
	body   *bytes.Buffer
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Gateway received request: %s %s", r.Method, r.URL.Path)

	route, err := p.config.MatchRoute(r.URL.Path)
	if err != nil {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	lb := p.getLoadBalancer(route)

	var finalBackend *Backend
	var finalStatus int

	err = p.retryConfig.ExecuteWithRetry(r.Context(), func() (int, error) {

		backend, err := p.selectBackend(route, lb)
		if err != nil {
			return 503, err
		}

		lcb, ok := lb.(*LeastConnectionsBalancer)
		if ok {
			lcb.IncrementConnections(backend.URL)
			defer lcb.DecrementConnections(backend.URL)
		}

		var bufferedResponse = &bufferingResponseWriter{}

		status, requestErr := p.executeRequest(bufferedResponse, r, backend, route, lb)
		log.Printf("🎯 Request completed: status=%d, error=%v", status, requestErr)
		finalBackend = backend
		finalStatus = status

		if status >= 500 {
			return status, fmt.Errorf("server error: %d", status)
		}

		if status < 500 {
			bufferedResponse.replayTo(w)
		}
		return status, requestErr
	})

	if err != nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	if finalStatus >= 500 {
		finalBackend.CircuitBreaker.RecordFailure()
		log.Printf("🔴 Circuit breaker recorded failure for %s (status: %d, failures %v)", finalBackend.URL, finalStatus, finalBackend.CircuitBreaker.GetStats())
	} else {
		finalBackend.CircuitBreaker.RecordSuccess()
		log.Printf("🟢 Circuit breaker recorded success for %s (status: %d)", finalBackend.URL, finalStatus)
	}

}

func (p *Proxy) customizeRequest(req *http.Request, route *Route, backend *Backend, lb LoadBalancer) {
	if route.StripPrefix != "" && strings.HasPrefix(req.URL.Path, route.StripPrefix) {
		req.URL.Path = strings.TrimPrefix(req.URL.Path, route.StripPrefix)
	}

	// Gateway headers
	req.Header.Set("X-Forwarded-By", "go-ai-gateway")
	req.Header.Set("X-Gateway-Version", "1.0")
	req.Header.Set("X-Backend-URL", backend.URL)
	req.Header.Set("X-Load-Balancer", lb.String())
}

// returns cached or new load balancer for route
func (p *Proxy) getLoadBalancer(route *Route) LoadBalancer {
	p.mutex.RLock()
	lb, exists := p.loadBalancers[route.Pattern]
	p.mutex.RUnlock()

	if exists {
		return lb
	}

	// Create new load balancer
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Double-check after acquiring write lock
	lb, exists = p.loadBalancers[route.Pattern]

	if exists {
		return lb
	}

	lb = NewLoadBalancer(route.LoadBalancer)
	p.loadBalancers[route.Pattern] = lb
	log.Printf("Created %s load balancer for route : %s", lb.String(), route.Pattern)
	return lb
}

func (p *Proxy) selectBackend(route *Route, lb LoadBalancer) (*Backend, error) {
	backends := route.GetBackends()
	if len(backends) == 0 {
		return nil, fmt.Errorf("no backends configured for route: %s", route.Pattern)
	}

	return lb.SelectBackend(backends)
}

func (p *Proxy) executeRequest(w http.ResponseWriter, r *http.Request, backend *Backend, route *Route, lb LoadBalancer) (int, error) {

	log.Printf("Select backend: %s (strategy %s)", backend.URL, lb.String())

	targetURL, err := url.Parse(backend.URL)
	if err != nil {
		return 500, fmt.Errorf("invalid backend URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		p.customizeRequest(req, route, backend, lb)
	}

	statusTracker := &statusTracker{
		ResponseWriter: w,
		status:         200,
	}

	proxy.ServeHTTP(statusTracker, r)

	if statusTracker.status >= 500 {
		return statusTracker.status, fmt.Errorf("server error: %d", statusTracker.status)
	}

	return statusTracker.status, nil
}

func (b *bufferingResponseWriter) Header() http.Header {
	if b.header == nil {
		b.header = make(http.Header)
	}
	return b.header
}

func (b *bufferingResponseWriter) Write(data []byte) (int, error) {
	if b.body == nil {
		b.body = &bytes.Buffer{}
	}
	return b.body.Write(data)
}

func (b *bufferingResponseWriter) WriteHeader(status int) {
	b.status = status
}

func (b *bufferingResponseWriter) replayTo(w http.ResponseWriter) {
	for key, values := range b.header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	if b.status != 0 {
		w.WriteHeader(b.status)
	}

	if b.body != nil {
		w.Write(b.body.Bytes())
	}
}
