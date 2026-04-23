package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNoAliveNodes      = errors.New("no alive nodes")
	ErrNoValidCandidates = errors.New("no valid candidates")
)

type RunElectionRequest struct {
	CandidateIDs []string `json:"candidate_ids"`
}

type ElectionResult struct {
	Term            int64             `json:"term"`
	LeaderID        string            `json:"leader_id"`
	LeaderVotes     int               `json:"leader_votes"`
	AliveVoters     int               `json:"alive_voters"`
	Majority        int               `json:"majority"`
	MajorityReached bool              `json:"majority_reached"`
	Candidates      []string          `json:"candidates"`
	VoteCounts      map[string]int    `json:"vote_counts"`
	VoterDecisions  map[string]string `json:"voter_decisions"`
	ElectedAt       string            `json:"elected_at"`
}

type voteRequest struct {
	Term         int64    `json:"term"`
	CandidateIDs []string `json:"candidate_ids"`
}

type voteResponse struct {
	VotedFor string `json:"voted_for"`
}

type ElectionService struct {
	mu          sync.RWMutex
	currentTerm int64
	lastResult  ElectionResult
	nodeService *NodeService
	client      *http.Client
}

func NewElectionService(nodeService *NodeService) *ElectionService {
	return &ElectionService{
		nodeService: nodeService,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

func (s *ElectionService) RunElection(req RunElectionRequest) (ElectionResult, error) {
	aliveNodes := s.nodeService.AliveNodes()
	voters := make([]NodeStatus, 0, len(aliveNodes))
	for _, node := range aliveNodes {
		if node.IsAlive {
			voters = append(voters, node)
		}
	}
	if len(voters) == 0 {
		return ElectionResult{}, ErrNoAliveNodes
	}

	candidates := s.resolveCandidates(req.CandidateIDs, voters)
	if len(candidates) == 0 {
		return ElectionResult{}, ErrNoValidCandidates
	}

	term := s.nextTerm()
	counts := make(map[string]int, len(candidates))
	decisions := make(map[string]string, len(voters))
	for _, candidateID := range candidates {
		counts[candidateID] = 0
	}

	for _, voter := range voters {
		votedFor := s.requestVote(voter.Address, term, candidates)
		if !slices.Contains(candidates, votedFor) {
			continue
		}
		counts[votedFor]++
		decisions[voter.NodeID] = votedFor
	}

	leaderID, leaderVotes := selectWinner(counts, candidates)
	if err := s.nodeService.SetLeader(leaderID); err != nil {
		return ElectionResult{}, err
	}

	majority := (len(voters) / 2) + 1
	result := ElectionResult{
		Term:            term,
		LeaderID:        leaderID,
		LeaderVotes:     leaderVotes,
		AliveVoters:     len(voters),
		Majority:        majority,
		MajorityReached: leaderVotes >= majority,
		Candidates:      candidates,
		VoteCounts:      counts,
		VoterDecisions:  decisions,
		ElectedAt:       time.Now().UTC().Format(time.RFC3339),
	}

	s.mu.Lock()
	s.lastResult = result
	s.mu.Unlock()

	return result, nil
}

func (s *ElectionService) LastResult() ElectionResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastResult
}

func (s *ElectionService) nextTerm() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentTerm++
	return s.currentTerm
}

func (s *ElectionService) resolveCandidates(requested []string, voters []NodeStatus) []string {
	aliveSet := make(map[string]bool, len(voters))
	out := make([]string, 0, len(voters))
	for _, node := range voters {
		aliveSet[node.NodeID] = true
	}

	if len(requested) == 0 {
		for _, node := range voters {
			out = append(out, node.NodeID)
		}
		sort.Strings(out)
		return out
	}

	seen := make(map[string]bool, len(requested))
	for _, candidateID := range requested {
		id := strings.TrimSpace(candidateID)
		if id == "" || seen[id] || !aliveSet[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (s *ElectionService) requestVote(address string, term int64, candidates []string) string {
	url := strings.TrimRight(strings.TrimSpace(address), "/") + "/vote-request"
	body, _ := json.Marshal(voteRequest{
		Term:         term,
		CandidateIDs: candidates,
	})

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return ""
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var vote voteResponse
	if err := json.NewDecoder(resp.Body).Decode(&vote); err != nil {
		return ""
	}
	return strings.TrimSpace(vote.VotedFor)
}

func selectWinner(counts map[string]int, candidates []string) (string, int) {
	if len(candidates) == 0 {
		return "", 0
	}

	winner := candidates[0]
	maxVotes := counts[winner]
	for _, candidateID := range candidates[1:] {
		votes := counts[candidateID]
		if votes > maxVotes || (votes == maxVotes && candidateID < winner) {
			maxVotes = votes
			winner = candidateID
		}
	}
	return winner, maxVotes
}

func (s *ElectionService) CurrentTerm() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentTerm
}

func (s *ElectionService) DebugState() string {
	last := s.LastResult()
	return fmt.Sprintf("term=%d leader=%s", s.CurrentTerm(), last.LeaderID)
}
