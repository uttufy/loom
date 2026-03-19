// Package coordinator provides multi-agent coordination.
package coordinator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/uttufy/loom/internal/beadsclient"
)

// Coordinator manages multi-agent coordination and file locking.
type Coordinator struct {
	client   *beadsclient.Client
	locks    map[string]*FileLock
	mu       sync.RWMutex
	timeout  time.Duration
}

// FileLock represents a lock on a file.
type FileLock struct {
	FilePath  string    `json:"file_path"`
	IssueID   string    `json:"issue_id"`
	AgentID   string    `json:"agent_id"`
	ClaimedAt time.Time `json:"claimed_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ClaimRequest represents a request to claim a task.
type ClaimRequest struct {
	IssueID       string   `json:"issue_id"`
	AgentID       string   `json:"agent_id"`
	ModifiedFiles []string `json:"modified_files"`
}

// New creates a new Coordinator.
func New(client *beadsclient.Client, lockTimeout time.Duration) *Coordinator {
	return &Coordinator{
		client:  client,
		locks:   make(map[string]*FileLock),
		timeout: lockTimeout,
	}
}

// ClaimTask attempts to claim a task with file declarations.
func (c *Coordinator) ClaimTask(ctx context.Context, req *ClaimRequest) error {
	// Check for file-level conflicts first
	for _, file := range req.ModifiedFiles {
		if conflict := c.checkFileLock(ctx, file, req.AgentID); conflict != nil {
			return &ConflictError{
				File:     file,
				AgentID:  conflict.AgentID,
				IssueID:  conflict.IssueID,
			}
		}
	}

	// Claim the task via Beads
	if err := c.client.Claim(ctx, req.IssueID, req.AgentID); err != nil {
		return fmt.Errorf("failed to claim issue: %w", err)
	}

	// Create file locks
	now := time.Now()
	for _, file := range req.ModifiedFiles {
		c.mu.Lock()
		c.locks[file] = &FileLock{
			FilePath:  file,
			IssueID:   req.IssueID,
			AgentID:   req.AgentID,
			ClaimedAt: now,
			ExpiresAt: now.Add(c.timeout),
		}
		c.mu.Unlock()
	}

	return nil
}

// ReleaseTask releases a task and its file locks.
func (c *Coordinator) ReleaseTask(ctx context.Context, issueID string) error {
	// Remove all locks for this issue
	c.mu.Lock()
	for file, lock := range c.locks {
		if lock.IssueID == issueID {
			delete(c.locks, file)
		}
	}
	c.mu.Unlock()

	return nil
}

// checkFileLock checks if a file is locked by another agent.
func (c *Coordinator) checkFileLock(ctx context.Context, file, agentID string) *FileLock {
	c.mu.RLock()
	lock, exists := c.locks[file]
	c.mu.RUnlock()

	if !exists {
		return nil
	}

	// Check if lock has expired
	if time.Now().After(lock.ExpiresAt) {
		c.mu.Lock()
		delete(c.locks, file)
		c.mu.Unlock()
		return nil
	}

	// Check if it's the same agent
	if lock.AgentID == agentID {
		return nil
	}

	return lock
}

// GetLocks returns all current file locks.
func (c *Coordinator) GetLocks() []*FileLock {
	c.mu.RLock()
	defer c.mu.RUnlock()

	locks := make([]*FileLock, 0, len(c.locks))
	for _, lock := range c.locks {
		locks = append(locks, lock)
	}
	return locks
}

// GetLock returns the lock for a specific file.
func (c *Coordinator) GetLock(file string) *FileLock {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.locks[file]
}

// DetectConflicts checks for potential conflicts between agents.
func (c *Coordinator) DetectConflicts(ctx context.Context) ([]*Conflict, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Group locks by agent
	agentFiles := make(map[string][]*FileLock)
	for _, lock := range c.locks {
		agentFiles[lock.AgentID] = append(agentFiles[lock.AgentID], lock)
	}

	// Check for overlapping files
	var conflicts []*Conflict
	for agent1, files1 := range agentFiles {
		for agent2, files2 := range agentFiles {
			if agent1 >= agent2 {
				continue
			}

			for _, f1 := range files1 {
				for _, f2 := range files2 {
					if c.filesOverlap(f1.FilePath, f2.FilePath) {
						conflicts = append(conflicts, &Conflict{
							File:     f1.FilePath,
							Agent1:   agent1,
							Issue1:   f1.IssueID,
							Agent2:   agent2,
							Issue2:   f2.IssueID,
						})
					}
				}
			}
		}
	}

	return conflicts, nil
}

// filesOverlap checks if two file paths might conflict.
func (c *Coordinator) filesOverlap(file1, file2 string) bool {
	if file1 == file2 {
		return true
	}
	// Could add more sophisticated overlap detection
	// e.g., directory containment, related files
	return false
}

// Conflict represents a detected conflict.
type Conflict struct {
	File   string `json:"file"`
	Agent1 string `json:"agent1"`
	Issue1 string `json:"issue1"`
	Agent2 string `json:"agent2"`
	Issue2 string `json:"issue2"`
}

// ConflictError indicates a conflict prevented claiming.
type ConflictError struct {
	File    string
	AgentID string
	IssueID string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("file %s locked by agent %s (issue %s)", e.File, e.AgentID, e.IssueID)
}

// CleanupExpired removes expired locks.
func (c *Coordinator) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	now := time.Now()
	for file, lock := range c.locks {
		if now.After(lock.ExpiresAt) {
			delete(c.locks, file)
			count++
		}
	}
	return count
}
