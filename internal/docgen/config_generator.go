package docgen

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// ConfigGenerator generates Mintlify documentation from config structs.
type ConfigGenerator struct{}

// NewConfigGenerator creates a new config documentation generator.
func NewConfigGenerator() *ConfigGenerator {
	return &ConfigGenerator{}
}

// ConfigSection represents a documentation section for a config type.
type ConfigSection struct {
	Name        string
	Slug        string
	Description string
	Fields      []ConfigField
}

// ConfigField represents a field in a config section.
type ConfigField struct {
	Name        string
	Type        string
	Default     string
	Description string
	YAMLName    string
}

// Generate generates config documentation to the specified output directory.
func (g *ConfigGenerator) Generate(outputDir string, sections []ConfigSection) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate overview page
	if err := g.generateOverview(outputDir, sections); err != nil {
		return err
	}

	// Generate pages for each section
	for _, section := range sections {
		if err := g.generateSectionPage(outputDir, section); err != nil {
			return err
		}
	}

	return nil
}

// generateOverview creates the config overview page.
func (g *ConfigGenerator) generateOverview(outputDir string, sections []ConfigSection) error {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString("title: Configuration Reference\n")
	b.WriteString("description: Complete configuration reference for Loom\n")
	b.WriteString("---\n\n")

	b.WriteString("# Configuration Reference\n\n")
	b.WriteString("Loom is configured via `loom.yaml` in your project directory. ")
	b.WriteString("Run `loom config init` to create a default configuration.\n\n")

	b.WriteString("## Configuration File Location\n\n")
	b.WriteString("Loom searches for configuration in this order:\n\n")
	b.WriteString("1. `--config` flag value\n")
	b.WriteString("2. `loom.yaml` in the current directory\n")
	b.WriteString("3. `loom.yaml` in parent directories\n\n")

	b.WriteString("## Configuration Sections\n\n")
	b.WriteString("| Section | Description |\n")
	b.WriteString("|---------|-------------|\n")

	for _, section := range sections {
		b.WriteString(fmt.Sprintf("| [%s](/config-reference/%s) | %s |\n", section.Name, section.Slug, section.Description))
	}

	b.WriteString("\n## Example Configuration\n\n")
	b.WriteString("```yaml\n")
	b.WriteString("# Beads integration\n")
	b.WriteString("beads:\n")
	b.WriteString("  path: bd\n")
	b.WriteString("  timeout: 30s\n\n")
	b.WriteString("# Scoring weights\n")
	b.WriteString("scoring:\n")
	b.WriteString("  blocking_multiplier: 3\n")
	b.WriteString("  priority_boost: 2\n")
	b.WriteString("  staleness_days: 3\n")
	b.WriteString("  staleness_bonus: 1\n")
	b.WriteString("  failure_penalty: 1\n\n")
	b.WriteString("# Hooks configuration\n")
	b.WriteString("hooks:\n")
	b.WriteString("  enabled: true\n\n")
	b.WriteString("# Safety configuration\n")
	b.WriteString("safety:\n")
	b.WriteString("  block_destructive: true\n\n")
	b.WriteString("# Memory management\n")
	b.WriteString("memory:\n")
	b.WriteString("  compact_threshold: 0.70\n\n")
	b.WriteString("# Coordination\n")
	b.WriteString("coordination:\n")
	b.WriteString("  enabled: true\n")
	b.WriteString("  lock_timeout: 1h\n\n")
	b.WriteString("# Learning\n")
	b.WriteString("learning:\n")
	b.WriteString("  enabled: true\n")
	b.WriteString("```\n")

	return os.WriteFile(filepath.Join(outputDir, "overview.mdx"), []byte(b.String()), 0644)
}

// generateSectionPage generates documentation for a single config section.
func (g *ConfigGenerator) generateSectionPage(outputDir string, section ConfigSection) error {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("title: %s Configuration\n", section.Name))
	b.WriteString(fmt.Sprintf("description: %s\n", section.Description))
	b.WriteString("---\n\n")

	b.WriteString(fmt.Sprintf("# %s\n\n", section.Name))
	b.WriteString(fmt.Sprintf("%s\n\n", section.Description))

	b.WriteString("## Options\n\n")
	b.WriteString("| Option | Type | Default | Description |\n")
	b.WriteString("|--------|------|---------|-------------|\n")

	for _, field := range section.Fields {
		b.WriteString(fmt.Sprintf("| `%s` | %s | `%s` | %s |\n",
			field.YAMLName, field.Type, field.Default, field.Description))
	}

	b.WriteString("\n## Example\n\n")
	b.WriteString("```yaml\n")
	b.WriteString(fmt.Sprintf("%s:\n", strings.ToLower(section.Name)))
	for _, field := range section.Fields {
		b.WriteString(fmt.Sprintf("  %s: %s\n", field.YAMLName, field.Default))
	}
	b.WriteString("```\n")

	return os.WriteFile(filepath.Join(outputDir, section.Slug+".mdx"), []byte(b.String()), 0644)
}

// GetDefaultSections returns the default config sections for Loom.
func GetDefaultSections() []ConfigSection {
	return []ConfigSection{
		{
			Name:        "Beads",
			Slug:        "beads",
			Description: "Configure the Beads CLI integration for task management.",
			Fields: []ConfigField{
				{
					YAMLName:    "path",
					Type:        "string",
					Default:     "bd",
					Description: "Path to the Beads CLI executable",
				},
				{
					YAMLName:    "timeout",
					Type:        "duration",
					Default:     "30s",
					Description: "Command timeout duration",
				},
			},
		},
		{
			Name:        "Scoring",
			Slug:        "scoring",
			Description: "Configure task prioritization weights for intelligent ranking.",
			Fields: []ConfigField{
				{
					YAMLName:    "blocking_multiplier",
					Type:        "int",
					Default:     "3",
					Description: "Score multiplier for each blocked task (+N per blocked task)",
				},
				{
					YAMLName:    "priority_boost",
					Type:        "int",
					Default:     "2",
					Description: "Score bonus for P0/P1 priority tasks",
				},
				{
					YAMLName:    "staleness_days",
					Type:        "int",
					Default:     "3",
					Description: "Days before a task is considered stale",
				},
				{
					YAMLName:    "staleness_bonus",
					Type:        "int",
					Default:     "1",
					Description: "Score bonus for stale tasks",
				},
				{
					YAMLName:    "failure_penalty",
					Type:        "int",
					Default:     "1",
					Description: "Score penalty for previously failed tasks",
				},
			},
		},
		{
			Name:        "Hooks",
			Slug:        "hooks",
			Description: "Configure lifecycle hooks for injecting custom behavior.",
			Fields: []ConfigField{
				{
					YAMLName:    "enabled",
					Type:        "bool",
					Default:     "true",
					Description: "Enable or disable hook execution",
				},
				{
					YAMLName:    "pre_prompt",
					Type:        "[]HookDefinition",
					Default:     "[]",
					Description: "Hooks to run before prompt processing",
				},
				{
					YAMLName:    "pre_tool_call",
					Type:        "[]HookDefinition",
					Default:     "[]",
					Description: "Hooks to run before tool execution",
				},
				{
					YAMLName:    "post_tool_call",
					Type:        "[]HookDefinition",
					Default:     "[]",
					Description: "Hooks to run after tool execution",
				},
				{
					YAMLName:    "post_response",
					Type:        "[]HookDefinition",
					Default:     "[]",
					Description: "Hooks to run after agent response",
				},
				{
					YAMLName:    "on_error",
					Type:        "[]HookDefinition",
					Default:     "[]",
					Description: "Hooks to run on task failure",
				},
				{
					YAMLName:    "on_claim",
					Type:        "[]HookDefinition",
					Default:     "[]",
					Description: "Hooks to run on task claim",
				},
				{
					YAMLName:    "pre_close",
					Type:        "[]HookDefinition",
					Default:     "[]",
					Description: "Hooks to run before task close",
				},
				{
					YAMLName:    "on_block",
					Type:        "[]HookDefinition",
					Default:     "[]",
					Description: "Hooks to run when task becomes blocked",
				},
			},
		},
		{
			Name:        "Safety",
			Slug:        "safety",
			Description: "Configure safety guards to protect against dangerous operations.",
			Fields: []ConfigField{
				{
					YAMLName:    "block_destructive",
					Type:        "bool",
					Default:     "true",
					Description: "Block destructive commands like rm -rf, git push --force",
				},
				{
					YAMLName:    "require_confirmation",
					Type:        "[]string",
					Default:     "[git push, npm publish, docker push]",
					Description: "Commands that require user confirmation",
				},
				{
					YAMLName:    "allowed_without_confirmation",
					Type:        "[]string",
					Default:     "[git status, git diff, cat, ls]",
					Description: "Commands always allowed without confirmation",
				},
			},
		},
		{
			Name:        "Memory",
			Slug:        "memory",
			Description: "Configure memory management and context compaction.",
			Fields: []ConfigField{
				{
					YAMLName:    "compact_threshold",
					Type:        "float",
					Default:     "0.70",
					Description: "Context usage threshold (0.0-1.0) to trigger compaction",
				},
				{
					YAMLName:    "retention.high_dependency",
					Type:        "int",
					Default:     "90",
					Description: "Percentage of high-dependency task details to retain",
				},
				{
					YAMLName:    "retention.failed_tasks",
					Type:        "int",
					Default:     "80",
					Description: "Percentage of failed task details to retain",
				},
				{
					YAMLName:    "retention.retrospectives",
					Type:        "int",
					Default:     "100",
					Description: "Percentage of retrospective details to retain",
				},
			},
		},
		{
			Name:        "Coordination",
			Slug:        "coordination",
			Description: "Configure multi-agent coordination for parallel work.",
			Fields: []ConfigField{
				{
					YAMLName:    "enabled",
					Type:        "bool",
					Default:     "true",
					Description: "Enable coordination features",
				},
				{
					YAMLName:    "lock_timeout",
					Type:        "duration",
					Default:     "1h",
					Description: "File lock expiration time",
				},
				{
					YAMLName:    "conflict_detection",
					Type:        "bool",
					Default:     "true",
					Description: "Enable conflict detection",
				},
			},
		},
		{
			Name:        "Learning",
			Slug:        "learning",
			Description: "Configure the learning system for continuous improvement.",
			Fields: []ConfigField{
				{
					YAMLName:    "enabled",
					Type:        "bool",
					Default:     "true",
					Description: "Enable learning features",
				},
				{
					YAMLName:    "retro_count",
					Type:        "int",
					Default:     "3",
					Description: "Number of recent retrospectives to keep",
				},
				{
					YAMLName:    "patterns_file",
					Type:        "string",
					Default:     "~/.beads-global/patterns.json",
					Description: "Path to the patterns storage file",
				},
			},
		},
	}
}

// FormatType returns a human-readable type string.
func FormatType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if t == reflect.TypeOf(time.Duration(0)) {
			return "duration"
		}
		return "int"
	case reflect.Bool:
		return "bool"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.Slice:
		return "[]" + FormatType(t.Elem())
	case reflect.Struct:
		return t.Name()
	default:
		return t.Kind().String()
	}
}
