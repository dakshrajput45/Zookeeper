package controller

import (
	"encoding/json"
	"net/http"

	"zookeeper/internal/service"
)

type HealthController struct {
	healthService *service.HealthService
}

func NewHealthController(healthService *service.HealthService) *HealthController {
	return &HealthController{healthService: healthService}
}

func (c *HealthController) GetHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := c.healthService.Status()
	_ = json.NewEncoder(w).Encode(response)
}
