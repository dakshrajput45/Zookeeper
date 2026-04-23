package controller

import (
	"net/http"

	"zookeeper/internal/service"
)

type ElectionController struct {
	electionService *service.ElectionService
}

func NewElectionController(electionService *service.ElectionService) *ElectionController {
	return &ElectionController{electionService: electionService}
}

func (c *ElectionController) GetElectionState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, c.electionService.LastResult())
}

