// Package api provides the public API for Loom.
package api

import (
	"context"

	"github.com/uttufy/loom/internal/beadsclient"
	"github.com/uttufy/loom/internal/config"
	"github.com/uttufy/loom/internal/coordinator"
	"github.com/uttufy/loom/internal/hooks"
	"github.com/uttufy/loom/internal/memory"
	"github.com/uttufy/loom/internal/orchestrator"
	"github.com/uttufy/loom/internal/retrospective"
	"github.com/uttufy/loom/internal/safety"
	"github.com/uttufy/loom/internal/scorer"
)

// Loom provides the main API for the orchestration system.
type Loom struct {
	orchestrator *orchestrator.Orchestrator
	client       *beadsclient.Client
	scorer       *scorer.Scorer
	hooks        *hooks.Engine
	coordinator  *coordinator.Coordinator
	retros       *retrospective.Store
	memory       *memory.Manager
	safety       *safety.Guard
	config       *config.Config
}

// New creates a new Loom instance with the given configuration.
func New(cfg *config.Config) (*Loom, error) {
	orch, err := orchestrator.New(cfg)
	if err != nil {
		return nil, err
	}

	client := beadsclient.NewClient(cfg.Beads.Path, ".", cfg.Beads.Timeout)
	retroStore, err := retrospective.NewStore(cfg.Learning.PatternsFile)
	if err != nil {
		return nil, err
	}

	return &Loom{
		orchestrator: orch,
		client:       client,
		scorer:       scorer.New(client, retroStore, &cfg.Scoring),
		hooks:        hooks.NewEngine(),
		coordinator:  coordinator.New(client, cfg.Coordination.LockTimeout),
		retros:       retroStore,
		memory:       memory.NewManager(client, &cfg.Memory),
		safety:       safety.NewGuard(),
		config:       cfg,
	}, nil
}

// NewFromConfigFile creates a Loom instance from a config file.
func NewFromConfigFile(path string) (*Loom, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	return New(cfg)
}

// Run starts the orchestration loop.
func (l *Loom) Run(ctx context.Context) error {
	return l.orchestrator.Run(ctx)
}

// Stop stops the orchestrator.
func (l *Loom) Stop() {
	l.orchestrator.Stop()
}

// GetReadyTasks returns prioritized ready tasks.
func (l *Loom) GetReadyTasks(ctx context.Context) ([]*scorer.ScoredIssue, error) {
	return l.orchestrator.GetReadyTasks(ctx)
}

// ClaimTask claims a task with file declarations.
func (l *Loom) ClaimTask(ctx context.Context, issueID, agentID string, files []string) error {
	return l.coordinator.ClaimTask(ctx, &coordinator.ClaimRequest{
		IssueID:       issueID,
		AgentID:       agentID,
		ModifiedFiles: files,
	})
}

// CompleteTask marks a task as completed.
func (l *Loom) CompleteTask(ctx context.Context, issueID, summary string) error {
	return l.orchestrator.CompleteTask(ctx, issueID, summary)
}

// FailTask records a task failure.
func (l *Loom) FailTask(ctx context.Context, issueID string, err error) error {
	return l.orchestrator.FailTask(ctx, issueID, err)
}

// RegisterHook registers a hook handler.
func (l *Loom) RegisterHook(event hooks.Event, handler hooks.Handler) {
	l.hooks.Register(event, handler)
}

// ValidateCommand validates a command for safety.
func (l *Loom) ValidateCommand(toolName, command string) error {
	return l.safety.Validate(toolName, command)
}

// ApproveCommand approves a command.
func (l *Loom) ApproveCommand(command string) {
	l.safety.Approve(command)
}

// GetLocks returns current file locks.
func (l *Loom) GetLocks() []*coordinator.FileLock {
	return l.coordinator.GetLocks()
}

// DetectConflicts detects potential conflicts.
func (l *Loom) DetectConflicts(ctx context.Context) ([]*coordinator.Conflict, error) {
	return l.coordinator.DetectConflicts(ctx)
}

// GetPatterns returns learned patterns.
func (l *Loom) GetPatterns() []*retrospective.Pattern {
	return l.retros.GetPatterns()
}

// SaveRetrospective saves a retrospective.
func (l *Loom) SaveRetrospective(retro *retrospective.Retrospective) error {
	return l.retros.Save(retro)
}

// GetMemoryStats returns memory statistics.
func (l *Loom) GetMemoryStats() *memory.Stats {
	return l.memory.GetStats()
}

// ShouldCompact returns true if compaction is needed.
func (l *Loom) ShouldCompact(contextUsage float64) bool {
	return l.memory.ShouldCompact(contextUsage)
}

// Compact runs memory compaction.
func (l *Loom) Compact(ctx context.Context) error {
	return l.client.Compact(ctx)
}

// ScoreTask scores a single task.
func (l *Loom) ScoreTask(ctx context.Context, issue *beadsclient.Issue) (int, error) {
	return l.scorer.ScoreTask(ctx, issue)
}

// GetClient returns the underlying Beads client.
func (l *Loom) GetClient() *beadsclient.Client {
	return l.client
}

// GetConfig returns the current configuration.
func (l *Loom) GetConfig() *config.Config {
	return l.config
}
