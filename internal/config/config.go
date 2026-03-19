// Package config handles Loom configuration loading and management.
package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the Loom configuration.
type Config struct {
	Beads       BeadsConfig       `yaml:"beads"`
	Scoring     ScoringConfig     `yaml:"scoring"`
	Hooks       HooksConfig       `yaml:"hooks"`
	Safety      SafetyConfig      `yaml:"safety"`
	Memory      MemoryConfig      `yaml:"memory"`
	Coordination CoordinationConfig `yaml:"coordination"`
	Learning    LearningConfig    `yaml:"learning"`
}

// BeadsConfig configures the Beads CLI integration.
type BeadsConfig struct {
	Path    string        `yaml:"path"`
	Timeout time.Duration `yaml:"timeout"`
}

// ScoringConfig configures task prioritization weights.
type ScoringConfig struct {
	BlockingMultiplier int `yaml:"blocking_multiplier"` // +N per blocked task
	PriorityBoost      int `yaml:"priority_boost"`      // +N for P0/P1 tasks
	StalenessDays      int `yaml:"staleness_days"`      // Days before staleness bonus
	StalenessBonus     int `yaml:"staleness_bonus"`     // +N for stale tasks
	FailurePenalty     int `yaml:"failure_penalty"`     // -N for previously failed
}

// HooksConfig configures the hook system.
type HooksConfig struct {
	Enabled      bool             `yaml:"enabled"`
	PrePrompt    []HookDefinition `yaml:"pre_prompt"`
	PreToolCall  []HookDefinition `yaml:"pre_tool_call"`
	PostToolCall []HookDefinition `yaml:"post_tool_call"`
	PostResponse []HookDefinition `yaml:"post_response"`
	OnError      []HookDefinition `yaml:"on_error"`
	OnClaim      []HookDefinition `yaml:"on_claim"`
	PreClose     []HookDefinition `yaml:"pre_close"`
	OnBlock      []HookDefinition `yaml:"on_block"`
}

// HookDefinition defines a hook.
type HookDefinition struct {
	Name    string `yaml:"name"`
	Script  string `yaml:"script,omitempty"`
	Builtin string `yaml:"builtin,omitempty"`
}

// SafetyConfig configures safety guards.
type SafetyConfig struct {
	BlockDestructive            bool     `yaml:"block_destructive"`
	RequireConfirmation         []string `yaml:"require_confirmation"`
	AllowedWithoutConfirmation  []string `yaml:"allowed_without_confirmation"`
}

// MemoryConfig configures memory management.
type MemoryConfig struct {
	CompactThreshold float64            `yaml:"compact_threshold"`
	Retention        RetentionConfig    `yaml:"retention"`
}

// RetentionConfig configures retention policies.
type RetentionConfig struct {
	HighDependency int `yaml:"high_dependency"` // Keep % of high-dependency task details
	FailedTasks    int `yaml:"failed_tasks"`    // Keep % of failed task details
	Retrospectives int `yaml:"retrospectives"`  // Keep % of retrospective details
}

// CoordinationConfig configures multi-agent coordination.
type CoordinationConfig struct {
	Enabled           bool          `yaml:"enabled"`
	LockTimeout       time.Duration `yaml:"lock_timeout"`
	ConflictDetection bool          `yaml:"conflict_detection"`
}

// LearningConfig configures the learning system.
type LearningConfig struct {
	Enabled      bool   `yaml:"enabled"`
	RetroCount   int    `yaml:"retro_count"`
	PatternsFile string `yaml:"patterns_file"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()

	return &Config{
		Beads: BeadsConfig{
			Path:    "bd",
			Timeout: 30 * time.Second,
		},
		Scoring: ScoringConfig{
			BlockingMultiplier: 3,
			PriorityBoost:      2,
			StalenessDays:      3,
			StalenessBonus:     1,
			FailurePenalty:     1,
		},
		Hooks: HooksConfig{
			Enabled: true,
			PrePrompt: []HookDefinition{
				{Name: "inject-context", Builtin: "context-injector"},
			},
			PreToolCall: []HookDefinition{
				{Name: "safety-check", Builtin: "safety-guard"},
			},
			PostToolCall: []HookDefinition{
				{Name: "truncate-output", Builtin: "output-truncator"},
			},
			PostResponse: []HookDefinition{
				{Name: "create-followups", Builtin: "followup-creator"},
			},
			OnError: []HookDefinition{
				{Name: "log-failure", Builtin: "error-logger"},
			},
		},
		Safety: SafetyConfig{
			BlockDestructive: true,
			RequireConfirmation: []string{
				"git push",
				"npm publish",
				"docker push",
			},
			AllowedWithoutConfirmation: []string{
				"git status",
				"git diff",
				"cat",
				"ls",
			},
		},
		Memory: MemoryConfig{
			CompactThreshold: 0.70,
			Retention: RetentionConfig{
				HighDependency: 90,
				FailedTasks:    80,
				Retrospectives: 100,
			},
		},
		Coordination: CoordinationConfig{
			Enabled:           true,
			LockTimeout:       1 * time.Hour,
			ConflictDetection: true,
		},
		Learning: LearningConfig{
			Enabled:      true,
			RetroCount:   3,
			PatternsFile: filepath.Join(home, ".beads-global", "patterns.json"),
		},
	}
}

// Load reads configuration from a file.
func Load(path string) (*Config, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// Save writes configuration to a file.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// FindConfig searches for a loom.yaml file.
func FindConfig() (string, error) {
	// Check current directory
	if _, err := os.Stat("loom.yaml"); err == nil {
		return "loom.yaml", nil
	}

	// Check parent directories
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		configPath := filepath.Join(parent, "loom.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
		dir = parent
	}

	return "", os.ErrNotExist
}
