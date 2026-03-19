// Package docgen generates documentation from CLI commands and config structs.
package docgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// CLIGenerator generates Mintlify documentation from Cobra commands.
type CLIGenerator struct {
	rootCmd *cobra.Command
}

// NewCLIGenerator creates a new CLI documentation generator.
func NewCLIGenerator(rootCmd *cobra.Command) *CLIGenerator {
	return &CLIGenerator{rootCmd: rootCmd}
}

// Generate generates CLI documentation to the specified output directory.
func (g *CLIGenerator) Generate(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate overview page
	if err := g.generateOverview(outputDir); err != nil {
		return err
	}

	// Generate pages for each command
	return g.generateCommands(g.rootCmd, outputDir, "")
}

// generateOverview creates the CLI overview page.
func (g *CLIGenerator) generateOverview(outputDir string) error {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString("title: CLI Reference\n")
	b.WriteString("description: Complete command reference for Loom CLI\n")
	b.WriteString("---\n\n")

	b.WriteString("# CLI Reference\n\n")
	b.WriteString("Complete reference for all Loom CLI commands.\n\n")

	b.WriteString("## Global Flags\n\n")
	b.WriteString("| Flag | Short | Description |\n")
	b.WriteString("|------|-------|-------------|\n")
	b.WriteString("| `--config` | `-c` | Path to config file (default is `loom.yaml`) |\n")
	b.WriteString("| `--json` | `-j` | Output in JSON format |\n")
	b.WriteString("| `--help` | `-h` | Show help for any command |\n")
	b.WriteString("| `--version` | `-v` | Show version information |\n\n")

	b.WriteString("## Commands Overview\n\n")
	b.WriteString(g.generateCommandsTable(g.rootCmd))

	return os.WriteFile(filepath.Join(outputDir, "overview.mdx"), []byte(b.String()), 0644)
}

// generateCommandsTable creates a table of all commands.
func (g *CLIGenerator) generateCommandsTable(cmd *cobra.Command) string {
	var b strings.Builder

	// Group commands by category
	categories := map[string][]*cobra.Command{
		"Core":         {},
		"Coordination": {},
		"Learning":     {},
		"Memory":       {},
		"Configuration": {},
	}

	for _, c := range cmd.Commands() {
		if c.IsAvailableCommand() && !c.IsAdditionalHelpTopicCommand() {
			name := c.Name()
			switch {
			case name == "run" || name == "ready" || name == "score" || name == "claim":
				categories["Core"] = append(categories["Core"], c)
			case name == "locks" || name == "conflicts":
				categories["Coordination"] = append(categories["Coordination"], c)
			case name == "retro" || name == "patterns":
				categories["Learning"] = append(categories["Learning"], c)
			case name == "status" || name == "compact":
				categories["Memory"] = append(categories["Memory"], c)
			case name == "config" || name == "hooks":
				categories["Configuration"] = append(categories["Configuration"], c)
			}
		}
	}

	for category, cmds := range categories {
		if len(cmds) == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("### %s\n\n", category))
		b.WriteString("| Command | Description |\n")
		b.WriteString("|---------|-------------|\n")
		for _, c := range cmds {
			slug := strings.ReplaceAll(c.CommandPath(), " ", "-")
			b.WriteString(fmt.Sprintf("| [`%s`](/%s) | %s |\n", c.CommandPath(), slug, c.Short))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// generateCommands recursively generates documentation for commands.
func (g *CLIGenerator) generateCommands(cmd *cobra.Command, outputDir, parentSlug string) error {
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}

		slug := c.Name()
		if parentSlug != "" {
			slug = parentSlug + "-" + slug
		}

		// Generate page for this command
		if err := g.generateCommandPage(c, outputDir, slug); err != nil {
			return err
		}

		// Recursively process subcommands
		if err := g.generateCommands(c, outputDir, slug); err != nil {
			return err
		}
	}

	return nil
}

// generateCommandPage generates a documentation page for a single command.
func (g *CLIGenerator) generateCommandPage(cmd *cobra.Command, outputDir, slug string) error {
	var b strings.Builder

	// Frontmatter
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("title: %s\n", cmd.CommandPath()))
	b.WriteString(fmt.Sprintf("description: %s\n", cmd.Short))
	b.WriteString("---\n\n")

	// Title
	b.WriteString(fmt.Sprintf("# %s\n\n", cmd.CommandPath()))
	b.WriteString(fmt.Sprintf("%s\n\n", cmd.Short))

	// Long description
	if cmd.Long != "" {
		b.WriteString("## Overview\n\n")
		b.WriteString(cmd.Long + "\n\n")
	}

	// Usage
	b.WriteString("## Usage\n\n")
	b.WriteString("```bash\n")
	b.WriteString(cmd.CommandPath())
	if cmd.HasFlags() {
		b.WriteString(" [flags]")
	}
	if cmd.Args != nil {
		b.WriteString(" [args]")
	}
	b.WriteString("\n```\n\n")

	// Arguments
	if cmd.Args != nil {
		b.WriteString("## Arguments\n\n")
		if cmd.Name() == "score" {
			b.WriteString("| Argument | Required | Description |\n")
			b.WriteString("|----------|----------|-------------|\n")
			b.WriteString("| `<issue-id>` | Yes | The issue identifier to score |\n")
		} else if cmd.Name() == "claim" {
			b.WriteString("| Argument | Required | Description |\n")
			b.WriteString("|----------|----------|-------------|\n")
			b.WriteString("| `<issue-id>` | Yes | The issue identifier to claim |\n")
			b.WriteString("| `[files...]` | No | Files to lock for coordination |\n")
		} else if strings.HasPrefix(slug, "retro-show") {
			b.WriteString("| Argument | Required | Description |\n")
			b.WriteString("|----------|----------|-------------|\n")
			b.WriteString("| `<retro-id>` | Yes | The retrospective identifier |\n")
		} else if strings.HasPrefix(slug, "hooks-test") {
			b.WriteString("| Argument | Required | Description |\n")
			b.WriteString("|----------|----------|-------------|\n")
			b.WriteString("| `<event>` | Yes | The hook event to test |\n")
		}
		b.WriteString("\n")
	}

	// Flags
	if cmd.HasFlags() || cmd.HasPersistentFlags() {
		b.WriteString("## Flags\n\n")
		b.WriteString("| Flag | Short | Default | Description |\n")
		b.WriteString("|------|-------|---------|-------------|\n")

		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			b.WriteString(g.formatFlag(f))
		})
		cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			b.WriteString(g.formatFlag(f))
		})
		b.WriteString("\n")
	}

	// Examples
	b.WriteString("## Examples\n\n")
	b.WriteString(g.generateExamples(cmd, slug))

	// Related commands
	b.WriteString("## See Also\n\n")
	b.WriteString(g.generateSeeAlso(cmd))

	filename := slug + ".mdx"
	return os.WriteFile(filepath.Join(outputDir, filename), []byte(b.String()), 0644)
}

// formatFlag formats a flag for the documentation table.
func (g *CLIGenerator) formatFlag(f *pflag.Flag) string {
	shorthand := ""
	if f.Shorthand != "" {
		shorthand = "`-" + f.Shorthand + "`"
	}
	return fmt.Sprintf("| `--%s` | %s | `%s` | %s |\n", f.Name, shorthand, f.DefValue, f.Usage)
}

// generateExamples generates example commands.
func (g *CLIGenerator) generateExamples(cmd *cobra.Command, slug string) string {
	var b strings.Builder

	switch cmd.Name() {
	case "run":
		b.WriteString("```bash\n")
		b.WriteString("# Start the orchestrator\n")
		b.WriteString("loom run\n\n")
		b.WriteString("# With a specific config file\n")
		b.WriteString("loom run --config /path/to/loom.yaml\n")
		b.WriteString("```\n")

	case "ready":
		b.WriteString("```bash\n")
		b.WriteString("# Show prioritized ready tasks\n")
		b.WriteString("loom ready\n\n")
		b.WriteString("# Output in JSON format\n")
		b.WriteString("loom ready --json\n")
		b.WriteString("```\n")

	case "score":
		b.WriteString("```bash\n")
		b.WriteString("# Show score breakdown for a task\n")
		b.WriteString("loom score issue-42\n\n")
		b.WriteString("# Output in JSON format\n")
		b.WriteString("loom score issue-42 --json\n")
		b.WriteString("```\n")

	case "claim":
		b.WriteString("```bash\n")
		b.WriteString("# Claim a task\n")
		b.WriteString("loom claim issue-42\n\n")
		b.WriteString("# Claim with file declarations\n")
		b.WriteString("loom claim issue-42 src/auth.go src/auth_test.go\n")
		b.WriteString("```\n")

	case "locks":
		b.WriteString("```bash\n")
		b.WriteString("# Show all active file locks\n")
		b.WriteString("loom locks\n\n")
		b.WriteString("# Output in JSON format\n")
		b.WriteString("loom locks --json\n")
		b.WriteString("```\n")

	case "conflicts":
		b.WriteString("```bash\n")
		b.WriteString("# Detect potential conflicts\n")
		b.WriteString("loom conflicts\n")
		b.WriteString("```\n")

	case "status":
		b.WriteString("```bash\n")
		b.WriteString("# Show orchestrator status\n")
		b.WriteString("loom status\n\n")
		b.WriteString("# Output in JSON format\n")
		b.WriteString("loom status --json\n")
		b.WriteString("```\n")

	case "compact":
		b.WriteString("```bash\n")
		b.WriteString("# Run memory compaction\n")
		b.WriteString("loom compact\n")
		b.WriteString("```\n")

	case "patterns":
		b.WriteString("```bash\n")
		b.WriteString("# List learned patterns\n")
		b.WriteString("loom patterns\n\n")
		b.WriteString("# Output in JSON format\n")
		b.WriteString("loom patterns --json\n")
		b.WriteString("```\n")

	case "list":
		if strings.HasPrefix(slug, "retro") {
			b.WriteString("```bash\n")
			b.WriteString("# List recent retrospectives\n")
			b.WriteString("loom retro list\n")
			b.WriteString("```\n")
		} else if strings.HasPrefix(slug, "hooks") {
			b.WriteString("```bash\n")
			b.WriteString("# List all registered hooks\n")
			b.WriteString("loom hooks list\n\n")
			b.WriteString("# Output in JSON format\n")
			b.WriteString("loom hooks list --json\n")
			b.WriteString("```\n")
		}

	case "show":
		if strings.HasPrefix(slug, "retro") {
			b.WriteString("```bash\n")
			b.WriteString("# Show retrospective details\n")
			b.WriteString("loom retro show retro-123\n")
			b.WriteString("```\n")
		}

	case "create":
		if strings.HasPrefix(slug, "retro") {
			b.WriteString("```bash\n")
			b.WriteString("# Create a new retrospective\n")
			b.WriteString("loom retro create\n")
			b.WriteString("```\n")
		}

	case "test":
		if strings.HasPrefix(slug, "hooks") {
			b.WriteString("```bash\n")
			b.WriteString("# Test a hook event\n")
			b.WriteString("loom hooks test pre-tool-call\n")
			b.WriteString("```\n")
		}

	case "init":
		if strings.HasPrefix(slug, "config") {
			b.WriteString("```bash\n")
			b.WriteString("# Initialize config in current directory\n")
			b.WriteString("loom config init\n\n")
			b.WriteString("# Initialize with a specific path\n")
			b.WriteString("loom config init --config /path/to/loom.yaml\n")
			b.WriteString("```\n")
		}

	case "mcp":
		b.WriteString("```bash\n")
		b.WriteString("# Start the MCP server\n")
		b.WriteString("loom mcp\n")
		b.WriteString("```\n")

	default:
		b.WriteString("```bash\n")
		b.WriteString(fmt.Sprintf("# %s\n", cmd.Short))
		b.WriteString(cmd.CommandPath() + "\n")
		b.WriteString("```\n")
	}

	return b.String()
}

// generateSeeAlso generates related commands section.
func (g *CLIGenerator) generateSeeAlso(cmd *cobra.Command) string {
	var b strings.Builder

	// Add parent command if not root
	if cmd.Parent() != nil && cmd.Parent() != g.rootCmd {
		parent := cmd.Parent()
		slug := strings.ReplaceAll(parent.CommandPath(), " ", "-")
		b.WriteString(fmt.Sprintf("- [`%s`](/cli-reference/%s) - %s\n", parent.CommandPath(), slug, parent.Short))
	}

	// Add sibling commands
	if cmd.Parent() != nil {
		for _, sibling := range cmd.Parent().Commands() {
			if sibling != cmd && sibling.IsAvailableCommand() && !sibling.IsAdditionalHelpTopicCommand() {
				slug := strings.ReplaceAll(sibling.CommandPath(), " ", "-")
				b.WriteString(fmt.Sprintf("- [`%s`](/cli-reference/%s) - %s\n", sibling.CommandPath(), slug, sibling.Short))
			}
		}
	}

	return b.String()
}
