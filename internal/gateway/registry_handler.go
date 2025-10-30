package gateway

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aishahsofea/go-ai-gateway/internal/utils"
)

func (sr *ServiceRegistry) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var instance ServiceInstance
	err := json.NewDecoder(r.Body).Decode(&instance)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if instance.ID == "" || instance.Route == "" || instance.URL == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	err = sr.RegisterService(&instance)
	if err != nil {
		http.Error(w, "failed to register service", http.StatusInternalServerError)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "service registered successfully"})
}

func (sr *ServiceRegistry) DeregisterHandler(w http.ResponseWriter, r *http.Request) {

	serviceID := r.PathValue("id")
	if serviceID == "" {
		http.Error(w, "missing service ID", http.StatusBadRequest)
		return
	}

	route := r.URL.Query().Get("route")
	if route == "" {
		http.Error(w, "missing route", http.StatusBadRequest)
		return
	}

	err := sr.DeregisterService(route, serviceID)
	if err != nil {

		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, "failed to deregister service", http.StatusInternalServerError)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "service deregistered successfully"})

}

func (sr *ServiceRegistry) GetAllServicesHandler(w http.ResponseWriter, r *http.Request) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	// group services by route
	servicesByRoute := make(map[string][]*ServiceInstance)

	for route, services := range sr.services {
		serviceList := make([]*ServiceInstance, 0, len(services))
		for _, service := range services {
			serviceList = append(serviceList, service)
		}
		servicesByRoute[route] = serviceList
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": servicesByRoute})
}

func (sr *ServiceRegistry) GetServicesByRouteHandler(w http.ResponseWriter, r *http.Request) {
	route := r.URL.Query().Get("route")

	if route == "" {
		http.Error(w, "missing route parameter", http.StatusBadRequest)
		return
	}

	services := sr.GetServices(route)
	if services == nil {
		http.Error(w, "no services found for the specified route", http.StatusNotFound)
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": services})
}
