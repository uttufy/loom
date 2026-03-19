// Package memory provides context and memory management.
package memory

import (
	"context"

	"github.com/uttufy/loom/internal/beadsclient"
	"github.com/uttufy/loom/internal/config"
)

// Manager handles memory compaction and context management.
type Manager struct {
	client     *beadsclient.Client
	config     *config.MemoryConfig
	stats      *Stats
}

// Stats tracks memory statistics.
type Stats struct {
	ContextUsage    float64 `json:"context_usage"`
	CompactionCount int     `json:"compaction_count"`
	TotalIssues     int     `json:"total_issues"`
	RetainedIssues  int     `json:"retained_issues"`
}

// NewManager creates a new memory manager.
func NewManager(client *beadsclient.Client, cfg *config.MemoryConfig) *Manager {
	return &Manager{
		client: client,
		config: cfg,
		stats:  &Stats{},
	}
}

// CheckAndCompact checks if compaction is needed and runs it.
func (m *Manager) CheckAndCompact(ctx context.Context, contextUsage float64) error {
	m.stats.ContextUsage = contextUsage

	if contextUsage < m.config.CompactThreshold {
		return nil
	}

	// Run importance-weighted compaction
	if err := m.client.Compact(ctx); err != nil {
		return err
	}

	m.stats.CompactionCount++
	return nil
}

// GetStats returns current memory statistics.
func (m *Manager) GetStats() *Stats {
	return m.stats
}

// ShouldCompact returns true if compaction should be run.
func (m *Manager) ShouldCompact(contextUsage float64) bool {
	return contextUsage >= m.config.CompactThreshold
}

// CalculateImportance calculates the importance score for an issue.
// Higher importance = retain more detail during compaction.
func (m *Manager) CalculateImportance(issue *beadsclient.Issue, dependencyCount int, hasFailed bool) float64 {
	importance := 0.5 // Base importance

	// High-dependency tasks are more important
	if dependencyCount >= 3 {
		importance += 0.3
	} else if dependencyCount >= 1 {
		importance += 0.15
	}

	// Failed tasks are important for learning
	if hasFailed {
		importance += 0.2
	}

	// Recent tasks are more important
	// (This would be calculated based on age)

	// Cap at 1.0
	if importance > 1.0 {
		importance = 1.0
	}

	return importance
}

// GetRetentionPercent returns the retention percentage for an issue.
func (m *Manager) GetRetentionPercent(importance float64) int {
	// Map importance to retention based on config
	// High importance -> high retention
	if importance >= 0.8 {
		return m.config.Retention.HighDependency
	} else if importance >= 0.6 {
		return m.config.Retention.FailedTasks
	}
	return 50 // Default retention
}
