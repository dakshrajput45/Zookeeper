package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"zookeeper/internal/service"
)

type NodeController struct {
	nodeService     *service.NodeService
	electionService *service.ElectionService
}

func NewNodeController(nodeService *service.NodeService, electionService *service.ElectionService) *NodeController {
	return &NodeController{
		nodeService:     nodeService,
		electionService: electionService,
	}
}

func (c *NodeController) RegisterNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req service.RegisterNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.NodeID) == "" || strings.TrimSpace(req.Address) == "" {
		http.Error(w, "node_id and address are required", http.StatusBadRequest)
		return
	}

	c.nodeService.RegisterNode(req)
	c.electionService.ObserveHeartbeat(req.NodeID)
	writeJSON(w, http.StatusCreated, map[string]any{
		"accepted":  true,
		"node_id":   req.NodeID,
		"leader_id": c.nodeService.LeaderID(),
	})
}

func (c *NodeController) Heartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req service.HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.NodeID) == "" {
		http.Error(w, "node_id is required", http.StatusBadRequest)
		return
	}

	if err := c.nodeService.Heartbeat(req); err != nil {
		if errors.Is(err, service.ErrNodeNotFound) {
			http.Error(w, "node not registered", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to process heartbeat", http.StatusInternalServerError)
		return
	}
	c.electionService.ObserveHeartbeat(req.NodeID)

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"node_id":   req.NodeID,
		"leader_id": c.nodeService.LeaderID(),
	})
}

func (c *NodeController) GetLeader(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	leaderID := c.nodeService.LeaderID()
	writeJSON(w, http.StatusOK, map[string]any{
		"leader_id": leaderID,
		"found":     leaderID != "",
	})
}

func (c *NodeController) GetAliveNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nodes := c.nodeService.AliveNodes()
	aliveCount := 0
	for _, node := range nodes {
		if node.IsAlive {
			aliveCount++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count":       len(nodes),
		"alive_count": aliveCount,
		"dead_count":  len(nodes) - aliveCount,
		"nodes":       nodes,
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
