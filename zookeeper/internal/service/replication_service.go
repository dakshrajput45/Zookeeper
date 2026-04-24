package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	ErrWriteKeyRequired     = errors.New("key is required")
	ErrWriteValueRequired   = errors.New("value is required")
	ErrReadFailed           = errors.New("leader read failed")
	ErrLeaderNotAvailable   = errors.New("leader not available")
	ErrLeaderAddressMissing = errors.New("leader address missing")
	ErrQuorumNotReached     = errors.New("replication quorum not reached")
)

type WriteRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type WriteResult struct {
	Index        int64    `json:"index"`
	Term         int64    `json:"term"`
	Key          string   `json:"key"`
	Value        string   `json:"value"`
	LeaderID     string   `json:"leader_id"`
	Quorum       int      `json:"quorum"`
	AckedBy      []string `json:"acked_by"`
	Committed    bool     `json:"committed"`
	CreatedAt    string   `json:"created_at"`
	CommittedAt  string   `json:"committed_at,omitempty"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

type ReadResult struct {
	Key      string `json:"key"`
	Value    string `json:"value,omitempty"`
	Found    bool   `json:"found"`
	LeaderID string `json:"leader_id"`
}

type ReplicationState struct {
	CommittedIndex int64         `json:"committed_index"`
	Entries        []WriteResult `json:"entries"`
}

type appendRequest struct {
	Index    int64  `json:"index"`
	Term     int64  `json:"term"`
	LeaderID string `json:"leader_id"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

type leaderReadResponse struct {
	Value string `json:"value"`
	Found bool   `json:"found"`
}

type ReplicationService struct {
	mu             sync.RWMutex
	nextIndex      int64
	committedIndex int64
	entries        []WriteResult
	nodeService    *NodeService
	election       *ElectionService
	client         *http.Client
}

func NewReplicationService(nodeService *NodeService, election *ElectionService) *ReplicationService {
	return &ReplicationService{
		nextIndex:   1,
		entries:     make([]WriteResult, 0),
		nodeService: nodeService,
		election:    election,
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

func (s *ReplicationService) ProposeWrite(req WriteRequest) (WriteResult, error) {
	if strings.TrimSpace(req.Key) == "" {
		return WriteResult{}, ErrWriteKeyRequired
	}
	if strings.TrimSpace(req.Value) == "" {
		return WriteResult{}, ErrWriteValueRequired
	}

	leaderID := strings.TrimSpace(s.nodeService.LeaderID())
	if leaderID == "" {
		return WriteResult{}, ErrLeaderNotAvailable
	}

	leaderNode, ok := s.nodeService.NodeByID(leaderID)
	if !ok {
		return WriteResult{}, ErrLeaderNotAvailable
	}
	if strings.TrimSpace(leaderNode.Address) == "" {
		return WriteResult{}, ErrLeaderAddressMissing
	}

	aliveNodes := s.aliveNodes()
	if len(aliveNodes) == 0 {
		return WriteResult{}, ErrLeaderNotAvailable
	}

	quorum := (len(aliveNodes) / 2) + 1
	entry := s.newEntry(req, leaderID, quorum)

	for _, node := range aliveNodes {
		if s.appendToNode(node.Address, entry) {
			entry.AckedBy = append(entry.AckedBy, node.NodeID)
		}
	}

	if len(entry.AckedBy) >= quorum {
		entry.Committed = true
		entry.CommittedAt = time.Now().UTC().Format(time.RFC3339)
		s.appendEntry(entry)
		s.setCommittedIndex(entry.Index)
		return entry, nil
	}

	entry.ErrorMessage = ErrQuorumNotReached.Error()
	s.appendEntry(entry)
	return entry, ErrQuorumNotReached
}

func (s *ReplicationService) Read(key string) (ReadResult, error) {
	if strings.TrimSpace(key) == "" {
		return ReadResult{}, ErrWriteKeyRequired
	}

	leaderID := strings.TrimSpace(s.nodeService.LeaderID())
	if leaderID == "" {
		return ReadResult{}, ErrLeaderNotAvailable
	}

	leaderNode, ok := s.nodeService.NodeByID(leaderID)
	if !ok {
		return ReadResult{}, ErrLeaderNotAvailable
	}
	if strings.TrimSpace(leaderNode.Address) == "" {
		return ReadResult{}, ErrLeaderAddressMissing
	}

	u := strings.TrimRight(strings.TrimSpace(leaderNode.Address), "/") + "/internal/read?key=" + url.QueryEscape(key)
	request, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return ReadResult{}, ErrReadFailed
	}

	response, err := s.client.Do(request)
	if err != nil {
		return ReadResult{}, ErrReadFailed
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return ReadResult{}, ErrReadFailed
	}

	var payload leaderReadResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return ReadResult{}, ErrReadFailed
	}

	return ReadResult{
		Key:      key,
		Value:    payload.Value,
		Found:    payload.Found,
		LeaderID: leaderID,
	}, nil
}

func (s *ReplicationService) State() ReplicationState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entriesCopy := make([]WriteResult, len(s.entries))
	copy(entriesCopy, s.entries)
	return ReplicationState{
		CommittedIndex: s.committedIndex,
		Entries:        entriesCopy,
	}
}

func (s *ReplicationService) aliveNodes() []NodeStatus {
	nodes := s.nodeService.AliveNodes()
	out := make([]NodeStatus, 0, len(nodes))
	for _, node := range nodes {
		if node.IsAlive {
			out = append(out, node)
		}
	}
	return out
}

func (s *ReplicationService) newEntry(req WriteRequest, leaderID string, quorum int) WriteResult {
	term := s.election.LastResult().Term
	if term <= 0 {
		term = 1
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry := WriteResult{
		Index:     s.nextIndex,
		Term:      term,
		Key:       req.Key,
		Value:     req.Value,
		LeaderID:  leaderID,
		Quorum:    quorum,
		AckedBy:   make([]string, 0),
		Committed: false,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	s.nextIndex++
	return entry
}

func (s *ReplicationService) appendToNode(address string, entry WriteResult) bool {
	url := strings.TrimRight(strings.TrimSpace(address), "/") + "/replication/append"
	body, _ := json.Marshal(appendRequest{
		Index:    entry.Index,
		Term:     entry.Term,
		LeaderID: entry.LeaderID,
		Key:      entry.Key,
		Value:    entry.Value,
	})

	request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return false
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := s.client.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()

	return response.StatusCode == http.StatusOK
}

func (s *ReplicationService) appendEntry(entry WriteResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entry)
}

func (s *ReplicationService) setCommittedIndex(index int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.committedIndex = index
}

