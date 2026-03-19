package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/uttufy/loom/internal/beadsclient"
	"github.com/uttufy/loom/internal/config"
	"github.com/uttufy/loom/internal/coordinator"
	"github.com/uttufy/loom/internal/docgen"
	"github.com/uttufy/loom/internal/orchestrator"
	"github.com/uttufy/loom/internal/retrospective"
	"github.com/uttufy/loom/internal/scorer"
	"github.com/uttufy/loom/internal/tui"
)

// loadConfig loads the configuration file.
func loadConfig() (*config.Config, error) {
	path := ConfigPath
	if path == "" {
		var err error
		path, err = config.FindConfig()
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}

	if path == "" {
		return config.DefaultConfig(), nil
	}

	return config.Load(path)
}

// getClient creates a Beads client.
func getClient(cfg *config.Config) *beadsclient.Client {
	return beadsclient.NewClient(cfg.Beads.Path, ".", cfg.Beads.Timeout)
}

func runOrchestrator(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	orch, err := orchestrator.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create orchestrator: %w", err)
	}

	fmt.Println("Starting Loom orchestrator...")
	fmt.Println("Press Ctrl+C to stop")

	return orch.Run(context.Background())
}

func showReadyTasks(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	client := getClient(cfg)
	retroStore, err := retrospective.NewStore(cfg.Learning.PatternsFile)
	if err != nil {
		return fmt.Errorf("failed to create retro store: %w", err)
	}

	// Get ready tasks
	ready, err := client.Ready(context.Background(), beadsclient.WorkFilter{})
	if err != nil {
		return fmt.Errorf("failed to get ready tasks: %w", err)
	}

	if len(ready) == 0 {
		fmt.Println("No ready tasks available")
		return nil
	}

	// Score and rank
	taskScorer := scorer.New(client, retroStore, &cfg.Scoring)
	scored, err := taskScorer.Rank(context.Background(), ready)
	if err != nil {
		return fmt.Errorf("failed to rank tasks: %w", err)
	}

	if JSONOutput {
		return json.NewEncoder(os.Stdout).Encode(scored)
	}

	// Display tasks
	fmt.Printf("Found %d ready tasks (sorted by priority):\n\n", len(scored))
	for i, s := range scored {
		fmt.Printf("%d. [%s] %s (score: %d)\n", i+1, s.Issue.ID, s.Issue.Title, s.Score)
		if s.Issue.Priority <= 1 {
			fmt.Printf("   Priority: P%d | ", s.Issue.Priority)
		}
		if len(s.Issue.Blocks) > 0 {
			fmt.Printf("Blocks: %d | ", len(s.Issue.Blocks))
		}
		fmt.Printf("Age: %s\n", time.Since(s.Issue.CreatedAt).Round(time.Hour))
	}

	return nil
}

func showTaskScore(cmd *cobra.Command, args []string) error {
	issueID := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	client := getClient(cfg)
	retroStore, err := retrospective.NewStore(cfg.Learning.PatternsFile)
	if err != nil {
		return fmt.Errorf("failed to create retro store: %w", err)
	}

	// Get the issue
	issue, err := client.Show(context.Background(), issueID)
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	// Score with breakdown
	taskScorer := scorer.New(client, retroStore, &cfg.Scoring)
	breakdown, err := taskScorer.ScoreTaskWithBreakdown(context.Background(), issue)
	if err != nil {
		return fmt.Errorf("failed to score task: %w", err)
	}

	if JSONOutput {
		return json.NewEncoder(os.Stdout).Encode(breakdown)
	}

	fmt.Printf("Score breakdown for %s: %s\n\n", issue.ID, issue.Title)
	fmt.Printf("Total Score: %d\n\n", breakdown.Total)
	fmt.Printf("Components:\n")
	fmt.Printf("  Blocking bonus:  +%d (blocks %d tasks)\n", breakdown.BlockingBonus, breakdown.BlockedCount)
	fmt.Printf("  Priority bonus:  +%d\n", breakdown.PriorityBonus)
	fmt.Printf("  Staleness bonus: +%d (stale: %v)\n", breakdown.StalenessBonus, breakdown.IsStale)
	fmt.Printf("  Failure penalty: -%d (failed before: %v)\n", breakdown.FailurePenalty, breakdown.HasFailedBefore)

	return nil
}

func claimTask(cmd *cobra.Command, args []string) error {
	issueID := args[0]
	files := args[1:]

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	client := getClient(cfg)
	coord := coordinator.New(client, cfg.Coordination.LockTimeout)

	req := &coordinator.ClaimRequest{
		IssueID:       issueID,
		AgentID:       "loom-cli",
		ModifiedFiles: files,
	}

	if err := coord.ClaimTask(context.Background(), req); err != nil {
		return fmt.Errorf("failed to claim task: %w", err)
	}

	fmt.Printf("Claimed task %s\n", issueID)
	if len(files) > 0 {
		fmt.Printf("Locked files: %v\n", files)
	}

	return nil
}

func listPatterns(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	store, err := retrospective.NewStore(cfg.Learning.PatternsFile)
	if err != nil {
		return fmt.Errorf("failed to create retro store: %w", err)
	}

	patterns := store.GetPatterns()
	if len(patterns) == 0 {
		fmt.Println("No patterns learned yet")
		return nil
	}

	if JSONOutput {
		return json.NewEncoder(os.Stdout).Encode(patterns)
	}

	fmt.Printf("Learned patterns (%d):\n\n", len(patterns))
	for i, p := range patterns {
		fmt.Printf("%d. %s\n", i+1, p.Name)
		fmt.Printf("   %s\n", p.Description)
		fmt.Printf("   Success rate: %.0f%% | Used %d times\n\n", p.SuccessRate*100, p.UseCount)
	}

	return nil
}

func listRetrospectives(cmd *cobra.Command, args []string) error {
	fmt.Println("Listing retrospectives...")
	// TODO: Implement retro listing
	return nil
}

func showRetrospective(cmd *cobra.Command, args []string) error {
	retroID := args[0]
	fmt.Printf("Showing retrospective %s...\n", retroID)
	// TODO: Implement retro show
	return nil
}

func createRetrospective(cmd *cobra.Command, args []string) error {
	fmt.Println("Creating retrospective...")
	// TODO: Implement retro creation
	return nil
}

func listHooks(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if JSONOutput {
		return json.NewEncoder(os.Stdout).Encode(cfg.Hooks)
	}

	fmt.Println("Registered hooks:")

	events := []struct {
		name string
		defs []config.HookDefinition
	}{
		{"pre-prompt", cfg.Hooks.PrePrompt},
		{"pre-tool-call", cfg.Hooks.PreToolCall},
		{"post-tool-call", cfg.Hooks.PostToolCall},
		{"post-response", cfg.Hooks.PostResponse},
		{"on-error", cfg.Hooks.OnError},
		{"on-claim", cfg.Hooks.OnClaim},
		{"pre-close", cfg.Hooks.PreClose},
		{"on-block", cfg.Hooks.OnBlock},
	}

	for _, event := range events {
		if len(event.defs) > 0 {
			fmt.Printf("  %s:\n", event.name)
			for _, def := range event.defs {
				if def.Script != "" {
					fmt.Printf("    - %s (script: %s)\n", def.Name, def.Script)
				} else {
					fmt.Printf("    - %s (builtin: %s)\n", def.Name, def.Builtin)
				}
			}
		}
	}

	return nil
}

func testHook(cmd *cobra.Command, args []string) error {
	event := args[0]
	fmt.Printf("Testing hook for event: %s\n", event)
	// TODO: Implement hook testing
	return nil
}

func showLocks(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	client := getClient(cfg)
	coord := coordinator.New(client, cfg.Coordination.LockTimeout)

	locks := coord.GetLocks()
	if len(locks) == 0 {
		fmt.Println("No active file locks")
		return nil
	}

	if JSONOutput {
		return json.NewEncoder(os.Stdout).Encode(locks)
	}

	fmt.Printf("Active file locks (%d):\n\n", len(locks))
	for _, lock := range locks {
		fmt.Printf("  %s\n", lock.FilePath)
		fmt.Printf("    Issue: %s | Agent: %s\n", lock.IssueID, lock.AgentID)
		fmt.Printf("    Claimed: %s | Expires: %s\n\n",
			lock.ClaimedAt.Format(time.RFC3339),
			lock.ExpiresAt.Format(time.RFC3339))
	}

	return nil
}

func detectConflicts(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	client := getClient(cfg)
	coord := coordinator.New(client, cfg.Coordination.LockTimeout)

	conflicts, err := coord.DetectConflicts(context.Background())
	if err != nil {
		return fmt.Errorf("failed to detect conflicts: %w", err)
	}

	if len(conflicts) == 0 {
		fmt.Println("No conflicts detected")
		return nil
	}

	if JSONOutput {
		return json.NewEncoder(os.Stdout).Encode(conflicts)
	}

	fmt.Printf("Detected conflicts (%d):\n\n", len(conflicts))
	for i, c := range conflicts {
		fmt.Printf("%d. File: %s\n", i+1, c.File)
		fmt.Printf("   Agent 1: %s (issue %s)\n", c.Agent1, c.Issue1)
		fmt.Printf("   Agent 2: %s (issue %s)\n\n", c.Agent2, c.Issue2)
	}

	return nil
}

func showStatus(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	client := getClient(cfg)

	// Get task counts
	ready, _ := client.Ready(context.Background(), beadsclient.WorkFilter{})
	inProgress, _ := client.List(context.Background(), beadsclient.WorkFilter{
		Status: []beadsclient.Status{beadsclient.StatusInProgress},
	})

	status := map[string]interface{}{
		"ready_tasks":      len(ready),
		"in_progress":      len(inProgress),
		"compact_threshold": cfg.Memory.CompactThreshold,
		"hooks_enabled":    cfg.Hooks.Enabled,
		"coordination":     cfg.Coordination.Enabled,
		"learning":         cfg.Learning.Enabled,
	}

	if JSONOutput {
		return json.NewEncoder(os.Stdout).Encode(status)
	}

	fmt.Println("Loom Status")
	fmt.Printf("  Ready tasks:     %d\n", len(ready))
	fmt.Printf("  In progress:     %d\n", len(inProgress))
	fmt.Printf("  Compact threshold: %.0f%%\n", cfg.Memory.CompactThreshold*100)
	fmt.Printf("  Hooks enabled:   %v\n", cfg.Hooks.Enabled)
	fmt.Printf("  Coordination:    %v\n", cfg.Coordination.Enabled)
	fmt.Printf("  Learning:        %v\n", cfg.Learning.Enabled)

	return nil
}

func runCompact(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	client := getClient(cfg)

	fmt.Println("Running importance-weighted compaction...")
	if err := client.Compact(context.Background()); err != nil {
		return fmt.Errorf("compaction failed: %w", err)
	}

	fmt.Println("Compaction complete")
	return nil
}

func initConfig(cmd *cobra.Command, args []string) error {
	cfg := config.DefaultConfig()

	path := ConfigPath
	if path == "" {
		path = "loom.yaml"
	}

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists: %s", path)
	}

	if err := cfg.Save(path); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Created config file: %s\n", path)
	return nil
}

func showConfig(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if JSONOutput {
		return json.NewEncoder(os.Stdout).Encode(cfg)
	}

	// Show config in a readable format
	fmt.Println("Loom Configuration")
	fmt.Printf("Beads path:     %s\n", cfg.Beads.Path)
	fmt.Printf("Beads timeout:  %s\n", cfg.Beads.Timeout)
	fmt.Println()
	fmt.Println("Scoring weights:")
	fmt.Printf("  Blocking multiplier: %d\n", cfg.Scoring.BlockingMultiplier)
	fmt.Printf("  Priority boost:      %d\n", cfg.Scoring.PriorityBoost)
	fmt.Printf("  Staleness days:      %d\n", cfg.Scoring.StalenessDays)
	fmt.Printf("  Staleness bonus:     %d\n", cfg.Scoring.StalenessBonus)
	fmt.Printf("  Failure penalty:     %d\n", cfg.Scoring.FailurePenalty)
	fmt.Println()
	fmt.Printf("Memory threshold: %.0f%%\n", cfg.Memory.CompactThreshold*100)
	fmt.Printf("Hooks enabled:    %v\n", cfg.Hooks.Enabled)
	fmt.Printf("Coordination:     %v\n", cfg.Coordination.Enabled)
	fmt.Printf("Learning:         %v\n", cfg.Learning.Enabled)

	return nil
}

func startMCPServer(cmd *cobra.Command, args []string) error {
	fmt.Println("Starting MCP server...")
	// TODO: Implement MCP server
	return fmt.Errorf("MCP server not yet implemented")
}

func generateCLIDocs(cmd *cobra.Command, args []string) error {
	outputDir := "./docs-site/cli-reference"
	if len(args) > 0 {
		outputDir = args[0]
	}

	gen := docgen.NewCLIGenerator(rootCmd)
	if err := gen.Generate(outputDir); err != nil {
		return fmt.Errorf("failed to generate CLI docs: %w", err)
	}

	fmt.Printf("Generated CLI documentation in %s\n", outputDir)
	return nil
}

func generateConfigDocs(cmd *cobra.Command, args []string) error {
	outputDir := "./docs-site/config-reference"
	if len(args) > 0 {
		outputDir = args[0]
	}

	gen := docgen.NewConfigGenerator()
	sections := docgen.GetDefaultSections()
	if err := gen.Generate(outputDir, sections); err != nil {
		return fmt.Errorf("failed to generate config docs: %w", err)
	}

	fmt.Printf("Generated config documentation in %s\n", outputDir)
	return nil
}

func runTUI(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	client := getClient(cfg)
	retroStore, err := retrospective.NewStore(cfg.Learning.PatternsFile)
	if err != nil {
		return fmt.Errorf("failed to create retro store: %w", err)
	}

	taskScorer := scorer.New(client, retroStore, &cfg.Scoring)

	// Create and run TUI
	model := tui.New(client, taskScorer, cfg)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
