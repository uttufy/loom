// Package scorer implements AI-driven task prioritization.
package scorer

import (
	"context"
	"time"

	"github.com/uttufy/loom/internal/beadsclient"
	"github.com/uttufy/loom/internal/config"
)

// Scorer scores tasks based on priority, dependencies, and history.
type Scorer struct {
	client     *beadsclient.Client
	retroStore RetroStore
	config     *config.ScoringConfig
}

// RetroStore provides access to retrospective data for failure tracking.
type RetroStore interface {
	HasFailure(issueID string) bool
	GetFailureCount(issueID string) int
}

// New creates a new Scorer.
func New(client *beadsclient.Client, retroStore RetroStore, cfg *config.ScoringConfig) *Scorer {
	return &Scorer{
		client:     client,
		retroStore: retroStore,
		config:     cfg,
	}
}

// ScoreTask calculates a priority score for a task.
// Higher scores indicate higher priority.
func (s *Scorer) ScoreTask(ctx context.Context, issue *beadsclient.Issue) (int, error) {
	score := 0

	// +N for tasks blocking other tasks
	dependents, err := s.client.GetDependents(ctx, issue.ID)
	if err == nil && len(dependents) >= 2 {
		score += s.config.BlockingMultiplier
	} else if err == nil && len(dependents) > 0 {
		// Half bonus for single blocker
		score += s.config.BlockingMultiplier / 2
	}

	// +N for P0/P1 priority
	if issue.Priority <= 1 {
		score += s.config.PriorityBoost
	}

	// +N for tasks open > N days (staleness)
	stalenessThreshold := time.Duration(s.config.StalenessDays) * 24 * time.Hour
	if time.Since(issue.CreatedAt) > stalenessThreshold {
		score += s.config.StalenessBonus
	}

	// -N for previously failed tasks
	if s.retroStore != nil {
		if s.retroStore.HasFailure(issue.ID) {
			score -= s.config.FailurePenalty
		}
	}

	return score, nil
}

// ScoreBreakdown provides a detailed breakdown of a task's score.
type ScoreBreakdown struct {
	Total             int  `json:"total"`
	BlockingBonus     int  `json:"blocking_bonus"`
	PriorityBonus     int  `json:"priority_bonus"`
	StalenessBonus    int  `json:"staleness_bonus"`
	FailurePenalty    int  `json:"failure_penalty"`
	BlockedCount      int  `json:"blocked_count"`
	IsStale           bool `json:"is_stale"`
	HasFailedBefore   bool `json:"has_failed_before"`
}

// ScoreTaskWithBreakdown calculates score with detailed breakdown.
func (s *Scorer) ScoreTaskWithBreakdown(ctx context.Context, issue *beadsclient.Issue) (*ScoreBreakdown, error) {
	breakdown := &ScoreBreakdown{}

	// Calculate blocking bonus
	dependents, err := s.client.GetDependents(ctx, issue.ID)
	if err == nil {
		breakdown.BlockedCount = len(dependents)
		if len(dependents) >= 2 {
			breakdown.BlockingBonus = s.config.BlockingMultiplier
		} else if len(dependents) > 0 {
			breakdown.BlockingBonus = s.config.BlockingMultiplier / 2
		}
	}

	// Calculate priority bonus
	if issue.Priority <= 1 {
		breakdown.PriorityBonus = s.config.PriorityBoost
	}

	// Calculate staleness bonus
	stalenessThreshold := time.Duration(s.config.StalenessDays) * 24 * time.Hour
	if time.Since(issue.CreatedAt) > stalenessThreshold {
		breakdown.StalenessBonus = s.config.StalenessBonus
		breakdown.IsStale = true
	}

	// Calculate failure penalty
	if s.retroStore != nil {
		if s.retroStore.HasFailure(issue.ID) {
			breakdown.FailurePenalty = s.config.FailurePenalty
			breakdown.HasFailedBefore = true
		}
	}

	breakdown.Total = breakdown.BlockingBonus + breakdown.PriorityBonus +
		breakdown.StalenessBonus - breakdown.FailurePenalty

	return breakdown, nil
}

// ScoredIssue pairs an issue with its score.
type ScoredIssue struct {
	Issue *beadsclient.Issue
	Score int
}

// RankOrders issues by score (highest first).
func (s *Scorer) Rank(ctx context.Context, issues []*beadsclient.Issue) ([]*ScoredIssue, error) {
	scored := make([]*ScoredIssue, len(issues))

	for i, issue := range issues {
		score, err := s.ScoreTask(ctx, issue)
		if err != nil {
			score = 0 // Default score on error
		}
		scored[i] = &ScoredIssue{
			Issue: issue,
			Score: score,
		}
	}

	// Sort by score descending
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].Score > scored[i].Score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	return scored, nil
}

// Top returns the highest-scoring issue.
func (s *Scorer) Top(ctx context.Context, issues []*beadsclient.Issue) (*ScoredIssue, error) {
	if len(issues) == 0 {
		return nil, nil
	}

	scored, err := s.Rank(ctx, issues)
	if err != nil {
		return nil, err
	}

	return scored[0], nil
}
