// Package main provides the CLI entry point for Loom.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time.
	Version = "dev"

	// ConfigPath is the path to the config file.
	ConfigPath string

	// JSONOutput controls JSON output format.
	JSONOutput bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "loom",
	Short: "Loom - AI Agent Orchestrator for Beads",
	Long: `Loom is an orchestration layer built on top of Beads (bd) that enhances
AI coding agents with intelligent task prioritization, lifecycle hooks,
multi-agent coordination, and learning capabilities.

Beads are the task, Loom is what weaves them together.`,
	Version: Version,
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the orchestrator loop",
	Long: `Start the main orchestration loop that:
1. Gets ready tasks from Beads
2. Scores and prioritizes them
3. Claims the top task
4. Executes work
5. Closes completed tasks
6. Repeats until no work or interrupted`,
	RunE: runOrchestrator,
}

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "Show prioritized ready tasks",
	Long: `Display all unblocked tasks ranked by priority score.

The scoring formula considers:
- Blocking impact (tasks blocking others)
- Priority level (P0/P1 boost)
- Staleness (tasks open too long)
- Failure history (previously failed tasks)`,
	RunE: showReadyTasks,
}

var scoreCmd = &cobra.Command{
	Use:   "score <issue-id>",
	Short: "Show task score breakdown",
	Long: `Display a detailed breakdown of how a task was scored.

Shows the contribution of each scoring factor:
- Blocking bonus
- Priority bonus
- Staleness bonus
- Failure penalty`,
	Args: cobra.ExactArgs(1),
	RunE: showTaskScore,
}

var claimCmd = &cobra.Command{
	Use:   "claim <issue-id> [files...]",
	Short: "Claim a task with file declarations",
	Long: `Claim a task and optionally declare which files you intend to modify.

File declarations enable multi-agent coordination by preventing
conflicts when multiple agents work on the same codebase.`,
	Args: cobra.MinimumNArgs(1),
	RunE: claimTask,
}

var retroCmd = &cobra.Command{
	Use:   "retro",
	Short: "Create and view retrospectives",
	Long: `Manage session retrospectives for learning.

Retrospectives capture:
- What tasks were completed/failed
- What strategies worked
- Lessons learned for future sessions`,
}

var patternsCmd = &cobra.Command{
	Use:   "patterns",
	Short: "List learned patterns",
	Long: `Display patterns learned from past sessions.

Patterns capture reusable strategies and approaches
that can be applied to similar tasks.`,
	RunE: listPatterns,
}

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage lifecycle hooks",
	Long: `List, add, remove, and test lifecycle hooks.

Available hook events:
- pre-prompt: Inject context before prompt processing
- pre-tool-call: Validate tool calls before execution
- post-tool-call: Process tool call results
- post-response: Create follow-ups from responses
- on-error: Handle errors during execution
- on-claim: Setup/linting on task claim
- pre-close: Verify tests before closing
- on-block: Reprioritization notification`,
}

var locksCmd = &cobra.Command{
	Use:   "locks",
	Short: "Show current file locks",
	Long: `Display all active file locks for multi-agent coordination.

File locks prevent conflicts when multiple agents
are working on the same codebase.`,
	RunE: showLocks,
}

var conflictsCmd = &cobra.Command{
	Use:   "conflicts",
	Short: "Detect potential conflicts",
	Long: `Analyze current locks and detect potential conflicts
between agents working on overlapping files.`,
	RunE: detectConflicts,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show context usage and stats",
	Long: `Display current orchestrator status including:
- Context window usage
- Compaction statistics
- Active locks
- Session progress`,
	RunE: showStatus,
}

var compactCmd = &cobra.Command{
	Use:   "compact",
	Short: "Run importance-weighted compaction",
	Long: `Run memory compaction to reduce context size.

Uses importance-weighted compaction that retains
more detail for high-dependency and failed tasks.`,
	RunE: runCompact,
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long: `View and manage Loom configuration.

Configuration is loaded from loom.yaml in the current
directory or parent directories.`,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&ConfigPath, "config", "c", "", "config file (default is loom.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&JSONOutput, "json", "j", false, "output in JSON format")

	// Add commands
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(readyCmd)
	rootCmd.AddCommand(scoreCmd)
	rootCmd.AddCommand(claimCmd)

	// Retro commands
	retroCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List recent retrospectives",
		RunE:  listRetrospectives,
	})
	retroCmd.AddCommand(&cobra.Command{
		Use:   "show <retro-id>",
		Short: "Show retrospective details",
		Args:  cobra.ExactArgs(1),
		RunE:  showRetrospective,
	})
	retroCmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create a new retrospective",
		RunE:  createRetrospective,
	})
	rootCmd.AddCommand(retroCmd)

	rootCmd.AddCommand(patternsCmd)

	// Hook commands
	hooksCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List registered hooks",
		RunE:  listHooks,
	})
	hooksCmd.AddCommand(&cobra.Command{
		Use:   "test <event>",
		Short: "Test hook execution",
		Args:  cobra.ExactArgs(1),
		RunE:  testHook,
	})
	rootCmd.AddCommand(hooksCmd)

	// Coordination commands
	rootCmd.AddCommand(locksCmd)
	rootCmd.AddCommand(conflictsCmd)

	// Memory commands
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(compactCmd)

	// Config commands
	configCmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Initialize loom config",
		RunE:  initConfig,
	})
	configCmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE:  showConfig,
	})
	rootCmd.AddCommand(configCmd)

	// MCP command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server for Claude Code",
		Long: `Start the Model Context Protocol (MCP) server
for integration with Claude Code.

Add to .mcp.json:
{
  "mcpServers": {
    "loom": {
      "command": "loom",
      "args": ["mcp"]
    }
  }
}`,
		RunE: startMCPServer,
	})

	// Docs command
	docsCmd := &cobra.Command{
		Use:   "docs [output-dir]",
		Short: "Generate CLI documentation",
		Long: `Generate CLI documentation in Mintlify format.

The generated documentation includes:
- Command overview page
- Individual pages for each command
- Usage examples and flags`,
		RunE: generateCLIDocs,
	}
	rootCmd.AddCommand(docsCmd)

	// Config docs command
	configDocsCmd := &cobra.Command{
		Use:   "config-docs [output-dir]",
		Short: "Generate config documentation",
		Long: `Generate configuration documentation in Mintlify format.

The generated documentation includes:
- Config overview page
- Individual pages for each config section
- Option tables with types and defaults`,
		RunE: generateConfigDocs,
	}
	rootCmd.AddCommand(configDocsCmd)

	// TUI command - Beautiful interactive terminal interface
	tuiCmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI dashboard",
		Long: `Launch a beautiful, interactive terminal user interface for Loom.

The TUI provides:
- Visual task list with keyboard navigation
- Real-time task prioritization and scoring
- Interactive task claiming and management
- File lock and pattern viewing
- Keyboard shortcuts for all operations

Keyboard shortcuts:
  ↑/k     Move up       ↓/j     Move down
  Enter   Claim task    s       Show score
  d       Details       r       Refresh
  l       View locks    p       View patterns
  ?       Help          q       Quit`,
		RunE: runTUI,
	}
	rootCmd.AddCommand(tuiCmd)
}
