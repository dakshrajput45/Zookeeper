package service

import (
	"errors"
	"sort"
	"sync"
	"time"
)

var (
	ErrNodeNotFound = errors.New("node not found")
)

const defaultHeartbeatTimeout = 15 * time.Second

type RegisterNodeRequest struct {
	NodeID  string `json:"node_id"`
	Address string `json:"address"`
}

type HeartbeatRequest struct {
	NodeID string `json:"node_id"`
}

type NodeStatus struct {
	NodeID            string `json:"node_id"`
	Address           string `json:"address"`
	IsLeader          bool   `json:"is_leader"`
	IsAlive           bool   `json:"is_alive"`
	LastHeartbeat     string `json:"last_heartbeat"`
	LastHeartbeatUnix int64 `json:"last_heartbeat_unix"`
}

type registeredNode struct {
	NodeID        string
	Address       string
	LastHeartbeat time.Time
}

type NodeService struct {
	mu               sync.RWMutex
	heartbeatTimeout time.Duration
	nodes            map[string]registeredNode
	leaderID         string
}

func NewNodeService() *NodeService {
	return &NodeService{
		heartbeatTimeout: defaultHeartbeatTimeout,
		nodes:            make(map[string]registeredNode),
	}
}

func (s *NodeService) RegisterNode(req RegisterNodeRequest) {
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nodes[req.NodeID] = registeredNode{
		NodeID:        req.NodeID,
		Address:       req.Address,
		LastHeartbeat: now,
	}

	// Bootstrap-only behavior for MVP; failover election comes later.
	if s.leaderID == "" {
		s.leaderID = req.NodeID
	}
}

func (s *NodeService) Heartbeat(req HeartbeatRequest) error {
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	node, ok := s.nodes[req.NodeID]
	if !ok {
		return ErrNodeNotFound
	}

	node.LastHeartbeat = now
	s.nodes[req.NodeID] = node
	return nil
}

func (s *NodeService) LeaderID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.leaderID
}

func (s *NodeService) SetLeader(nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.nodes[nodeID]; !ok {
		return ErrNodeNotFound
	}
	s.leaderID = nodeID
	return nil
}

func (s *NodeService) AliveNodes() []NodeStatus {
	now := time.Now().UTC()

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]NodeStatus, 0, len(s.nodes))
	for _, node := range s.nodes {
		isAlive := now.Sub(node.LastHeartbeat) <= s.heartbeatTimeout
		out = append(out, NodeStatus{
			NodeID:            node.NodeID,
			Address:           node.Address,
			IsLeader:          node.NodeID == s.leaderID,
			IsAlive:           isAlive,
			LastHeartbeat:     node.LastHeartbeat.Format(time.RFC3339),
			LastHeartbeatUnix: node.LastHeartbeat.Unix(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].NodeID < out[j].NodeID
	})
	return out
}

func (s *NodeService) AliveNodeIDs() []string {
	nodes := s.AliveNodes()
	out := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if node.IsAlive {
			out = append(out, node.NodeID)
		}
	}
	return out
}

func (s *NodeService) NodeByID(nodeID string) (registeredNode, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	node, ok := s.nodes[nodeID]
	return node, ok
}

func (s *NodeService) HeartbeatTimeout() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.heartbeatTimeout
}
