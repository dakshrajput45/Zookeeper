package service

import "time"

type HealthService struct{}

type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

func NewHealthService() *HealthService {
	return &HealthService{}
}

func (s *HealthService) Status() HealthResponse {
	return HealthResponse{
		Status:    "ok",
		Service:   "zookeeper",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}
