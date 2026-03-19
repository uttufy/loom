// Package beadsclient provides types for interacting with the Beads CLI.
package beadsclient

import "time"

// Status represents the status of an issue.
type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusClosed     Status = "closed"
)

// IssueType represents the type of an issue.
type IssueType string

const (
	IssueTypeTask  IssueType = "task"
	IssueTypeBug   IssueType = "bug"
	IssueTypeFeat  IssueType = "feat"
	IssueTypeChore IssueType = "chore"
)

// Issue represents a task or work item from Beads.
type Issue struct {
	// Standard Beads fields
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      Status     `json:"status"`
	Priority    int        `json:"priority"`
	IssueType   IssueType  `json:"issue_type"`
	Assignee    string     `json:"assignee,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Labels      []string   `json:"labels,omitempty"`
	ParentID    string     `json:"parent_id,omitempty"`

	// Dependency tracking
	Blocks      []string `json:"blocks,omitempty"`       // IDs of issues this blocks
	BlockedBy   []string `json:"blocked_by,omitempty"`   // IDs of issues blocking this

	// Loom extensions
	ModifiedFiles []string `json:"modified_files,omitempty"`
	LoomScore     int      `json:"loom_score,omitempty"`
	FailureCount  int      `json:"failure_count,omitempty"`
	LastAgent     string   `json:"last_agent,omitempty"`
	SessionID     string   `json:"session_id,omitempty"`
}

// WorkFilter is used to filter ready work.
type WorkFilter struct {
	Status    []Status `json:"status,omitempty"`
	Types     []IssueType `json:"types,omitempty"`
	Priority  *int     `json:"priority,omitempty"`
	Assignee  string   `json:"assignee,omitempty"`
	Label     string   `json:"label,omitempty"`
}

// Dependency represents a dependency between two issues.
type Dependency struct {
	ID          string    `json:"id"`
	BlockerID   string    `json:"blocker_id"`
	BlockedID   string    `json:"blocked_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// AuditEntry represents an entry in the audit trail.
type AuditEntry struct {
	ID        string    `json:"id"`
	IssueID   string    `json:"issue_id"`
	Action    string    `json:"action"`
	Agent     string    `json:"agent"`
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details,omitempty"`
}
