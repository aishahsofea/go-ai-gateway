package gateway

import (
	"fmt"
	"sync"
	"time"
)

type ServiceInstance struct {
	ID       string            `json:"id"`
	URL      string            `json:"url"`
	Health   string            `json:"health"`
	Route    string            `json:"route"`
	Metadata map[string]string `json:"metadata"`
	LastSeen time.Time         `json:"last_seen"`
}

type ServiceRegistry struct {
	mutex    sync.RWMutex
	services map[string]map[string]*ServiceInstance // route -> service_id -> instance
}

func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]map[string]*ServiceInstance),
	}
}

func (sr *ServiceRegistry) RegisterService(instance *ServiceInstance) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	if _, exists := sr.services[instance.Route]; !exists {
		sr.services[instance.Route] = make(map[string]*ServiceInstance)
	}

	instance.LastSeen = time.Now()
	sr.services[instance.Route][instance.ID] = instance
	return nil
}

func (sr *ServiceRegistry) DeregisterService(route, serviceID string) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	if _, exists := sr.services[route]; !exists {
		return fmt.Errorf("no services registered for route: %s", route)
	}

	if _, exists := sr.services[route][serviceID]; !exists {
		return fmt.Errorf("service ID %s not found for route: %s", serviceID, route)
	}

	delete(sr.services[route], serviceID)

	if len(sr.services[route]) == 0 {
		delete(sr.services, route)
	}
	return nil
}

func (sr *ServiceRegistry) GetServices(route string) []*ServiceInstance {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	if _, exists := sr.services[route]; !exists {
		return nil
	}

	instances := make([]*ServiceInstance, 0, len(sr.services[route]))
	for _, instance := range sr.services[route] {
		instances = append(instances, instance)
	}
	return instances
}

func (sr *ServiceRegistry) GetAllRoutes() []string {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	routes := make([]string, 0, len(sr.services))
	for route := range sr.services {
		routes = append(routes, route)
	}
	return routes
}

func (sr *ServiceRegistry) UpdateServiceHealth(route, serviceID, health string) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	if services, exists := sr.services[route]; exists {
		if instance, exists := services[serviceID]; exists {
			instance.Health = health
			instance.LastSeen = time.Now()
			return nil
		}
	}
	return fmt.Errorf("service %s not found for route %s", serviceID, route)
}
