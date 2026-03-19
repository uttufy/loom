// Package retrospective provides learning and retrospective storage.
package retrospective

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/uttufy/loom/internal/beadsclient"
)

// Store manages retrospectives and learned patterns.
type Store struct {
	patternsFile string
	patterns     []*Pattern
	failures     map[string][]FailedTask // issueID -> failures
	mu           sync.RWMutex
}

// Retrospective represents a session retrospective.
type Retrospective struct {
	ID          string        `json:"id"`
	SessionID   string        `json:"session_id"`
	AgentID     string        `json:"agent_id"`
	CreatedAt   time.Time     `json:"created_at"`
	Completed   []string      `json:"completed"`   // Issue IDs completed
	Failed      []FailedTask  `json:"failed"`
	WhatWorked  []string      `json:"what_worked"`
	WhatDidnt   []string      `json:"what_didnt_work"`
	Strategies  []string      `json:"strategies"`
	Patterns    []Pattern     `json:"patterns"`
	Duration    time.Duration `json:"duration"`
}

// FailedTask represents a failed task attempt.
type FailedTask struct {
	IssueID    string    `json:"issue_id"`
	Error      string    `json:"error"`
	Retries    int       `json:"retries"`
	Timestamp  time.Time `json:"timestamp"`
}

// Pattern represents a learned pattern.
type Pattern struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Applicability string    `json:"applicability"`
	SuccessRate   float64   `json:"success_rate"`
	Examples      []string  `json:"examples"`
	CreatedAt     time.Time `json:"created_at"`
	LastUsed      time.Time `json:"last_used"`
	UseCount      int       `json:"use_count"`
}

// NewStore creates a new retrospective store.
func NewStore(patternsFile string) (*Store, error) {
	s := &Store{
		patternsFile: patternsFile,
		patterns:     make([]*Pattern, 0),
		failures:     make(map[string][]FailedTask),
	}

	// Load existing patterns
	if err := s.loadPatterns(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return s, nil
}

// Save stores a retrospective.
func (s *Store) Save(retro *Retrospective) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Track failures for future reference
	for _, failed := range retro.Failed {
		s.failures[failed.IssueID] = append(s.failures[failed.IssueID], failed)
	}

	// Add new patterns
	for _, pattern := range retro.Patterns {
		s.patterns = append(s.patterns, &pattern)
	}

	// Persist patterns to file
	return s.savePatterns()
}

// HasFailure checks if an issue has previous failures.
func (s *Store) HasFailure(issueID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.failures[issueID]) > 0
}

// GetFailureCount returns the number of times an issue has failed.
func (s *Store) GetFailureCount(issueID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.failures[issueID])
}

// GetFailures returns all failures for an issue.
func (s *Store) GetFailures(issueID string) []FailedTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.failures[issueID]
}

// GetPatterns returns all learned patterns.
func (s *Store) GetPatterns() []*Pattern {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.patterns
}

// FindPattern finds a pattern by name or ID.
func (s *Store) FindPattern(query string) *Pattern {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.patterns {
		if p.ID == query || p.Name == query {
			return p
		}
	}
	return nil
}

// loadPatterns loads patterns from the patterns file.
func (s *Store) loadPatterns() error {
	if s.patternsFile == "" {
		return nil
	}

	data, err := os.ReadFile(s.patternsFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s.patterns)
}

// savePatterns saves patterns to the patterns file.
func (s *Store) savePatterns() error {
	if s.patternsFile == "" {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(s.patternsFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.patterns, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.patternsFile, data, 0644)
}

// CreateFromSession creates a retrospective from session data.
func (s *Store) CreateFromSession(sessionID, agentID string, completed []*beadsclient.Issue, failed []FailedTask, strategies []string) *Retrospective {
	return &Retrospective{
		ID:         generateID(),
		SessionID:  sessionID,
		AgentID:    agentID,
		CreatedAt:  time.Now(),
		Completed:  extractIDs(completed),
		Failed:     failed,
		Strategies: strategies,
	}
}

func extractIDs(issues []*beadsclient.Issue) []string {
	ids := make([]string, len(issues))
	for i, issue := range issues {
		ids[i] = issue.ID
	}
	return ids
}

func generateID() string {
	return fmt.Sprintf("retro-%d", time.Now().UnixNano())
}
