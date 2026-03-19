// Package orchestrator implements the core orchestration loop.
package orchestrator

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/uttufy/loom/internal/beadsclient"
	"github.com/uttufy/loom/internal/config"
	"github.com/uttufy/loom/internal/coordinator"
	"github.com/uttufy/loom/internal/hooks"
	"github.com/uttufy/loom/internal/memory"
	"github.com/uttufy/loom/internal/retrospective"
	"github.com/uttufy/loom/internal/safety"
	"github.com/uttufy/loom/internal/scorer"
)

// Orchestrator manages the main orchestration loop.
type Orchestrator struct {
	client      *beadsclient.Client
	scorer      *scorer.Scorer
	hooks       *hooks.Engine
	coordinator *coordinator.Coordinator
	retros      *retrospective.Store
	memory      *memory.Manager
	safety      *safety.Guard
	config      *config.Config

	agentID   string
	sessionID string

	mu       sync.Mutex
	running  bool
	stopChan chan struct{}
}

// New creates a new Orchestrator.
func New(cfg *config.Config) (*Orchestrator, error) {
	client := beadsclient.NewClient(cfg.Beads.Path, ".", cfg.Beads.Timeout)

	retroStore, err := retrospective.NewStore(cfg.Learning.PatternsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create retro store: %w", err)
	}

	return &Orchestrator{
		client:      client,
		scorer:      scorer.New(client, retroStore, &cfg.Scoring),
		hooks:       hooks.NewEngine(),
		coordinator: coordinator.New(client, cfg.Coordination.LockTimeout),
		retros:      retroStore,
		memory:      memory.NewManager(client, &cfg.Memory),
		safety:      safety.NewGuard(),
		config:      cfg,
		agentID:     "loom-agent",
		sessionID:   fmt.Sprintf("session-%d", time.Now().Unix()),
		stopChan:    make(chan struct{}),
	}, nil
}

// SetAgentID sets the agent identifier.
func (o *Orchestrator) SetAgentID(id string) {
	o.agentID = id
}

// SetSessionID sets the session identifier.
func (o *Orchestrator) SetSessionID(id string) {
	o.sessionID = id
}

// RegisterHook registers a hook handler.
func (o *Orchestrator) RegisterHook(event hooks.Event, handler hooks.Handler) {
	o.hooks.Register(event, handler)
}

// Run starts the orchestration loop.
func (o *Orchestrator) Run(ctx context.Context) error {
	o.mu.Lock()
	if o.running {
		o.mu.Unlock()
		return fmt.Errorf("orchestrator already running")
	}
	o.running = true
	o.mu.Unlock()

	defer func() {
		o.mu.Lock()
		o.running = false
		o.mu.Unlock()
	}()

	// Set up signal handling
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			log.Println("Received shutdown signal")
			cancel()
		case <-o.stopChan:
			cancel()
		}
	}()

	// Main loop
	for {
		select {
		case <-ctx.Done():
			return o.shutdown(ctx)
		default:
			if err := o.tick(ctx); err != nil {
				if err == ErrNoWork {
					// No work available, wait before retrying
					time.Sleep(5 * time.Second)
					continue
				}
				log.Printf("Orchestration error: %v", err)
			}
		}
	}
}

// Stop stops the orchestrator.
func (o *Orchestrator) Stop() {
	close(o.stopChan)
}

// tick performs one iteration of the orchestration loop.
func (o *Orchestrator) tick(ctx context.Context) error {
	// 1. Get ready tasks
	ready, err := o.client.Ready(ctx, beadsclient.WorkFilter{})
	if err != nil {
		return fmt.Errorf("failed to get ready tasks: %w", err)
	}

	if len(ready) == 0 {
		return ErrNoWork
	}

	// 2. Score and rank tasks
	scored, err := o.scorer.Rank(ctx, ready)
	if err != nil {
		return fmt.Errorf("failed to rank tasks: %w", err)
	}

	// 3. Try to claim the top task
	topTask := scored[0]

	// Execute pre-prompt hooks
	hookCtx := &hooks.Context{
		Event:   hooks.EventPrePrompt,
		IssueID: topTask.Issue.ID,
		AgentID: o.agentID,
		Metadata: map[string]any{
			"score":    topTask.Score,
			"issue":    topTask.Issue,
			"ready":    ready,
		},
	}

	if result, err := o.hooks.Execute(ctx, hookCtx); err != nil {
		return fmt.Errorf("pre-prompt hook failed: %w", err)
	} else if result.Block {
		return fmt.Errorf("blocked by hook: %s", result.Reason)
	}

	// 4. Claim the task
	if err := o.client.Claim(ctx, topTask.Issue.ID, o.agentID); err != nil {
		return fmt.Errorf("failed to claim task %s: %w", topTask.Issue.ID, err)
	}

	// Execute on-claim hooks
	claimCtx := &hooks.Context{
		Event:   hooks.EventOnClaim,
		IssueID: topTask.Issue.ID,
		AgentID: o.agentID,
	}
	o.hooks.Execute(ctx, claimCtx)

	log.Printf("Claimed task %s (score: %d): %s", topTask.Issue.ID, topTask.Score, topTask.Issue.Title)

	// 5. Return task for execution (delegated to agent)
	// The actual work execution is handled externally
	// This orchestrator manages the loop, not the execution

	return nil
}

// CompleteTask marks a task as completed.
func (o *Orchestrator) CompleteTask(ctx context.Context, issueID, summary string) error {
	// Execute pre-close hooks
	hookCtx := &hooks.Context{
		Event:   hooks.EventPreClose,
		IssueID: issueID,
		AgentID: o.agentID,
	}

	if result, err := o.hooks.Execute(ctx, hookCtx); err != nil {
		return fmt.Errorf("pre-close hook failed: %w", err)
	} else if result.Block {
		return fmt.Errorf("blocked by hook: %s", result.Reason)
	}

	// Close the task
	if err := o.client.Close(ctx, issueID, summary); err != nil {
		return fmt.Errorf("failed to close task: %w", err)
	}

	// Release file locks
	o.coordinator.ReleaseTask(ctx, issueID)

	// Execute post-response hooks
	postCtx := &hooks.Context{
		Event:   hooks.EventPostResponse,
		IssueID: issueID,
		AgentID: o.agentID,
		Response: summary,
	}
	o.hooks.Execute(ctx, postCtx)

	return nil
}

// FailTask records a task failure.
func (o *Orchestrator) FailTask(ctx context.Context, issueID string, err error) error {
	// Execute error hooks
	hookCtx := &hooks.Context{
		Event:   hooks.EventOnError,
		IssueID: issueID,
		AgentID: o.agentID,
		Error:   err,
	}
	o.hooks.Execute(ctx, hookCtx)

	// Record failure in retrospective store
	failure := retrospective.FailedTask{
		IssueID:   issueID,
		Error:     err.Error(),
		Timestamp: time.Now(),
	}

	// Release the task back to open status
	updates := map[string]interface{}{
		"status": beadsclient.StatusOpen,
	}
	if updateErr := o.client.Update(ctx, issueID, updates); updateErr != nil {
		return fmt.Errorf("failed to release task: %w", updateErr)
	}

	// Release file locks
	o.coordinator.ReleaseTask(ctx, issueID)

	// Save failure
	retro := &retrospective.Retrospective{
		ID:        fmt.Sprintf("failure-%s-%d", issueID, time.Now().Unix()),
		SessionID: o.sessionID,
		AgentID:   o.agentID,
		CreatedAt: time.Now(),
		Failed:    []retrospective.FailedTask{failure},
	}
	o.retros.Save(retro)

	return nil
}

// GetReadyTasks returns prioritized ready tasks.
func (o *Orchestrator) GetReadyTasks(ctx context.Context) ([]*scorer.ScoredIssue, error) {
	ready, err := o.client.Ready(ctx, beadsclient.WorkFilter{})
	if err != nil {
		return nil, err
	}

	return o.scorer.Rank(ctx, ready)
}

// shutdown performs cleanup on shutdown.
func (o *Orchestrator) shutdown(ctx context.Context) error {
	log.Println("Shutting down orchestrator...")

	// Create final retrospective
	retro := &retrospective.Retrospective{
		ID:        fmt.Sprintf("session-%s", o.sessionID),
		SessionID: o.sessionID,
		AgentID:   o.agentID,
		CreatedAt: time.Now(),
	}

	if err := o.retros.Save(retro); err != nil {
		log.Printf("Failed to save retrospective: %v", err)
	}

	// Clean up expired locks
	o.coordinator.CleanupExpired()

	return nil
}

// ErrNoWork indicates no work is available.
var ErrNoWork = fmt.Errorf("no work available")
