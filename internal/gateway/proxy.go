package gateway

import (
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
	mutex         sync.RWMutex
}

func NewProxy(config *GatewayConfig) *Proxy {
	return &Proxy{
		config:        config,
		loadBalancers: make(map[string]LoadBalancer),
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Gateway received request: %s %s", r.Method, r.URL.Path)

	route, err := p.config.MatchRoute(r.URL.Path)
	if err != nil {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	lb := p.getLoadBalancer(route)

	backend, err := p.selectBackend(route, lb)
	if err != nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	log.Printf("Select backend: %s (strategy %s)", backend.URL, lb.String())

	targetURL, err := url.Parse(backend.URL)
	if err != nil {
		http.Error(w, "Invalid target URL", http.StatusInternalServerError)
		return
	}

	lcb, ok := lb.(*LeastConnectionsBalancer)
	if ok {
		lcb.IncrementConnections(backend.URL)
		defer lcb.DecrementConnections(backend.URL)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		p.customizeRequest(req, route, backend, lb)
	}

	proxy.ServeHTTP(w, r)
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
