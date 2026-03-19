// Package beadsclient provides a client for interacting with the Beads CLI.
package beadsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// Client wraps the Beads CLI for programmatic access.
type Client struct {
	bdPath  string        // Path to bd executable
	dir     string        // Working directory
	timeout time.Duration // Command timeout
}

// NewClient creates a new Beads client.
func NewClient(bdPath, dir string, timeout time.Duration) *Client {
	if bdPath == "" {
		bdPath = "bd"
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		bdPath:  bdPath,
		dir:     dir,
		timeout: timeout,
	}
}

// runCommand executes a bd command and returns the output.
func (c *Client) runCommand(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.bdPath, args...)
	cmd.Dir = c.dir

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd command failed: %s: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd command failed: %w", err)
	}

	return output, nil
}

// Ready returns unblocked tasks ready for work.
func (c *Client) Ready(ctx context.Context, filter WorkFilter) ([]*Issue, error) {
	args := []string{"ready", "--json"}

	if len(filter.Types) > 0 {
		for _, t := range filter.Types {
			args = append(args, "--type", string(t))
		}
	}
	if filter.Priority != nil {
		args = append(args, "--priority", fmt.Sprintf("%d", *filter.Priority))
	}
	if filter.Assignee != "" {
		args = append(args, "--assignee", filter.Assignee)
	}
	if filter.Label != "" {
		args = append(args, "--label", filter.Label)
	}

	output, err := c.runCommand(ctx, args...)
	if err != nil {
		return nil, err
	}

	var issues []*Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse ready output: %w", err)
	}

	return issues, nil
}

// Show returns details for a specific issue.
func (c *Client) Show(ctx context.Context, issueID string) (*Issue, error) {
	output, err := c.runCommand(ctx, "show", issueID, "--json")
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(output, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse show output: %w", err)
	}

	return &issue, nil
}

// Create creates a new issue.
func (c *Client) Create(ctx context.Context, issue *Issue) error {
	args := []string{"create", issue.Title}

	if issue.Description != "" {
		args = append(args, "--description", issue.Description)
	}
	if issue.IssueType != "" {
		args = append(args, "--type", string(issue.IssueType))
	}
	if issue.Priority > 0 {
		args = append(args, "--priority", fmt.Sprintf("%d", issue.Priority))
	}
	for _, label := range issue.Labels {
		args = append(args, "--label", label)
	}
	if issue.ParentID != "" {
		args = append(args, "--parent", issue.ParentID)
	}

	_, err := c.runCommand(ctx, args...)
	return err
}

// Update updates an existing issue.
func (c *Client) Update(ctx context.Context, issueID string, updates map[string]interface{}) error {
	args := []string{"update", issueID}

	if status, ok := updates["status"].(Status); ok {
		args = append(args, "--status", string(status))
	}
	if assignee, ok := updates["assignee"].(string); ok {
		args = append(args, "--assignee", assignee)
	}
	if priority, ok := updates["priority"].(int); ok {
		args = append(args, "--priority", fmt.Sprintf("%d", priority))
	}
	if description, ok := updates["description"].(string); ok {
		args = append(args, "--description", description)
	}

	_, err := c.runCommand(ctx, args...)
	return err
}

// Claim claims an issue for an agent.
func (c *Client) Claim(ctx context.Context, issueID, agentID string) error {
	updates := map[string]interface{}{
		"status":   StatusInProgress,
		"assignee": agentID,
	}
	return c.Update(ctx, issueID, updates)
}

// Close closes an issue with a summary.
func (c *Client) Close(ctx context.Context, issueID, summary string) error {
	args := []string{"close", issueID}
	if summary != "" {
		args = append(args, summary)
	}
	_, err := c.runCommand(ctx, args...)
	return err
}

// AddDependency creates a dependency between issues.
func (c *Client) AddDependency(ctx context.Context, blockerID, blockedID string) error {
	_, err := c.runCommand(ctx, "dep", "add", blockerID, blockedID)
	return err
}

// RemoveDependency removes a dependency between issues.
func (c *Client) RemoveDependency(ctx context.Context, blockerID, blockedID string) error {
	_, err := c.runCommand(ctx, "dep", "remove", blockerID, blockedID)
	return err
}

// GetDependents returns issues that depend on the given issue.
func (c *Client) GetDependents(ctx context.Context, issueID string) ([]*Issue, error) {
	output, err := c.runCommand(ctx, "dep", "list", issueID, "--dependents", "--json")
	if err != nil {
		return nil, err
	}

	var issues []*Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse dependents output: %w", err)
	}

	return issues, nil
}

// GetBlockers returns issues that block the given issue.
func (c *Client) GetBlockers(ctx context.Context, issueID string) ([]*Issue, error) {
	output, err := c.runCommand(ctx, "dep", "list", issueID, "--blockers", "--json")
	if err != nil {
		return nil, err
	}

	var issues []*Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse blockers output: %w", err)
	}

	return issues, nil
}

// Compact runs the compaction routine.
func (c *Client) Compact(ctx context.Context) error {
	_, err := c.runCommand(ctx, "compact")
	return err
}

// List lists all issues with optional filters.
func (c *Client) List(ctx context.Context, filter WorkFilter) ([]*Issue, error) {
	args := []string{"list", "--json"}

	if len(filter.Status) > 0 {
		for _, s := range filter.Status {
			args = append(args, "--status", string(s))
		}
	}
	if len(filter.Types) > 0 {
		for _, t := range filter.Types {
			args = append(args, "--type", string(t))
		}
	}
	if filter.Priority != nil {
		args = append(args, "--priority", fmt.Sprintf("%d", *filter.Priority))
	}
	if filter.Assignee != "" {
		args = append(args, "--assignee", filter.Assignee)
	}
	if filter.Label != "" {
		args = append(args, "--label", filter.Label)
	}

	output, err := c.runCommand(ctx, args...)
	if err != nil {
		return nil, err
	}

	var issues []*Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse list output: %w", err)
	}

	return issues, nil
}
