package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"zookeeper/internal/service"
)

type ReplicationController struct {
	replicationService *service.ReplicationService
}

func NewReplicationController(replicationService *service.ReplicationService) *ReplicationController {
	return &ReplicationController{replicationService: replicationService}
}

func (c *ReplicationController) Write(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req service.WriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	result, err := c.replicationService.ProposeWrite(req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWriteKeyRequired),
			errors.Is(err, service.ErrWriteValueRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		case errors.Is(err, service.ErrLeaderNotAvailable),
			errors.Is(err, service.ErrLeaderAddressMissing):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		case errors.Is(err, service.ErrQuorumNotReached):
			writeJSON(w, http.StatusServiceUnavailable, result)
			return
		default:
			http.Error(w, "write failed", http.StatusInternalServerError)
			return
		}
	}

	writeJSON(w, http.StatusOK, result)
}

func (c *ReplicationController) Read(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	result, err := c.replicationService.Read(key)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWriteKeyRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		case errors.Is(err, service.ErrLeaderNotAvailable),
			errors.Is(err, service.ErrLeaderAddressMissing),
			errors.Is(err, service.ErrReadFailed):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		default:
			http.Error(w, "read failed", http.StatusInternalServerError)
			return
		}
	}

	writeJSON(w, http.StatusOK, result)
}

func (c *ReplicationController) GetState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, c.replicationService.State())
}

