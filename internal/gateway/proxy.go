package gateway

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type Proxy struct {
	config *GatewayConfig
}

func NewProxy(config *GatewayConfig) *Proxy {
	return &Proxy{
		config: config,
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, err := p.config.MatchRoute(r.URL.Path)
	if err != nil {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	targetURL, err := url.Parse(route.Target)
	if err != nil {
		http.Error(w, "Invalid target URL", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		p.customizeRequest(req, route)
	}

	proxy.ServeHTTP(w, r)
}

func (p *Proxy) customizeRequest(req *http.Request, route *Route) {
	if route.StripPrefix != "" && strings.HasPrefix(req.URL.Path, route.StripPrefix) {
		req.URL.Path = strings.TrimPrefix(req.URL.Path, route.StripPrefix)
	}

	// Gateway headers
	req.Header.Set("X-Forwarded-By", "go-ai-gateway")
	req.Header.Set("X-Gateway-Version", "1.0")
}
