// Package discovery provides dynamic task discovery during execution.
package discovery

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
)

// Detector discovers implied work from file changes and code analysis.
type Detector struct {
	patterns []*WorkPattern
}

// WorkPattern represents a pattern for detecting implied work.
type WorkPattern struct {
	Name        string
	Trigger     Trigger
	Suggestions []Suggestion
}

// Trigger defines when a pattern applies.
type Trigger struct {
	FilePattern string         // Glob pattern for files
	ContentRegex *regexp.Regexp // Regex to match in content
	FileAdded   bool           // Trigger on new files
	FileModified bool          // Trigger on modified files
}

// Suggestion is a suggested follow-up task.
type Suggestion struct {
	Title       string
	Type        string // task, bug, chore, docs
	Priority    int
	Labels      []string
	Description string
}

// NewDetector creates a new detector with default patterns.
func NewDetector() *Detector {
	d := &Detector{}

	// Add default patterns
	d.patterns = []*WorkPattern{
		{
			Name: "new-file-build",
			Trigger: Trigger{
				FileAdded: true,
			},
			Suggestions: []Suggestion{{
				Title:       "chore: add new files to build",
				Type:        "chore",
				Priority:    3,
				Labels:      []string{"build"},
				Description: "New files may need to be added to build configuration",
			}},
		},
		{
			Name: "api-change-docs",
			Trigger: Trigger{
				FilePattern: "**/*.go",
				ContentRegex: regexp.MustCompile(`(?:func|type|interface)\s+[A-Z]`),
			},
			Suggestions: []Suggestion{{
				Title:       "docs: update API documentation",
				Type:        "task",
				Priority:    2,
				Labels:      []string{"docs"},
				Description: "API changes detected - documentation may need updates",
			}},
		},
		{
			Name: "test-coverage",
			Trigger: Trigger{
				FilePattern: "**/*.go",
				FileModified: true,
			},
			Suggestions: []Suggestion{{
				Title:       "test: add tests for modified code",
				Type:        "task",
				Priority:    2,
				Labels:      []string{"testing"},
				Description: "Modified code should have corresponding test updates",
			}},
		},
		{
			Name: "todo-comment",
			Trigger: Trigger{
				ContentRegex: regexp.MustCompile(`TODO|FIXME|XXX|HACK`),
			},
			Suggestions: []Suggestion{{
				Title:       "task: address TODO/FIXME comments",
				Type:        "task",
				Priority:    3,
				Labels:      []string{"cleanup"},
				Description: "TODO/FIXME comments found in code",
			}},
		},
		{
			Name: "config-change",
			Trigger: Trigger{
				FilePattern: "**/config.*",
				FileModified: true,
			},
			Suggestions: []Suggestion{{
				Title:       "chore: update configuration documentation",
				Type:        "chore",
				Priority:    3,
				Labels:      []string{"config", "docs"},
				Description: "Configuration changes may need documentation",
			}},
		},
		{
			Name: "db-migration",
			Trigger: Trigger{
				FilePattern: "**/migrations/**",
				FileAdded: true,
			},
			Suggestions: []Suggestion{{
				Title:       "task: run database migrations",
				Type:        "task",
				Priority:    1,
				Labels:      []string{"database"},
				Description: "New database migration detected",
			}},
		},
		{
			Name: "dependency-change",
			Trigger: Trigger{
				FilePattern: "{go.mod,package.json,requirements.txt,Cargo.toml}",
				FileModified: true,
			},
			Suggestions: []Suggestion{{
				Title:       "chore: review dependency changes",
				Type:        "chore",
				Priority:    2,
				Labels:      []string{"dependencies"},
				Description: "Dependencies changed - review for security/compatibility",
			}},
		},
	}

	return d
}

// AddPattern adds a custom work pattern.
func (d *Detector) AddPattern(pattern *WorkPattern) {
	d.patterns = append(d.patterns, pattern)
}

// Detect analyzes file changes and returns suggested work.
func (d *Detector) Detect(ctx context.Context, changes []FileChange) []*Suggestion {
	var suggestions []*Suggestion
	seen := make(map[string]bool)

	for _, change := range changes {
		for _, pattern := range d.patterns {
			if d.matchesTrigger(pattern.Trigger, change) {
				for _, s := range pattern.Suggestions {
					key := s.Title
					if !seen[key] {
						seen[key] = true
						// Clone suggestion to avoid mutations
						sugg := s
						suggestions = append(suggestions, &sugg)
					}
				}
			}
		}
	}

	return suggestions
}

// FileChange represents a file change event.
type FileChange struct {
	Path     string
	Content  string
	Added    bool
	Modified bool
	Deleted  bool
}

// matchesTrigger checks if a change matches a trigger.
func (d *Detector) matchesTrigger(trigger Trigger, change FileChange) bool {
	// Check file operation type
	if trigger.FileAdded && !change.Added {
		return false
	}
	if trigger.FileModified && !change.Modified {
		return false
	}

	// Check file pattern
	if trigger.FilePattern != "" {
		matched, err := filepath.Match(trigger.FilePattern, change.Path)
		if err != nil || !matched {
			// Try as glob
			matched, err = filepath.Match(trigger.FilePattern, filepath.Base(change.Path))
			if err != nil || !matched {
				return false
			}
		}
	}

	// Check content regex
	if trigger.ContentRegex != nil {
		if !trigger.ContentRegex.MatchString(change.Content) {
			return false
		}
	}

	return true
}

// DetectFromContent analyzes content and returns suggestions.
func (d *Detector) DetectFromContent(ctx context.Context, path, content string) []*Suggestion {
	change := FileChange{
		Path:     path,
		Content:  content,
		Modified: true,
	}
	return d.Detect(ctx, []FileChange{change})
}

// DetectBugPatterns analyzes code for potential bugs.
func (d *Detector) DetectBugPatterns(ctx context.Context, content string) []string {
	var bugs []string

	// Common bug patterns
	patterns := []struct {
		regex   *regexp.Regexp
		message string
	}{
		{
			regex:   regexp.MustCompile(`panic\(`),
			message: "Potential unhandled panic",
		},
		{
			regex:   regexp.MustCompile(`defer.*Close\(\)`),
			message: "Resource cleanup deferred - verify error handling",
		},
		{
			regex:   regexp.MustCompile(`fmt\.Sprintf.*%s.*\+`),
			message: "Potential SQL injection - use parameterized queries",
		},
		{
			regex:   regexp.MustCompile(`exec\.Command.*\+`),
			message: "Potential command injection - validate inputs",
		},
		{
			regex:   regexp.MustCompile(`http\.Get\(.*\+`),
			message: "Potential SSRF - validate URLs",
		},
	}

	for _, p := range patterns {
		if p.regex.MatchString(content) {
			bugs = append(bugs, p.message)
		}
	}

	return bugs
}

// ExtractTODOs extracts TODO comments from content.
func (d *Detector) ExtractTODOs(content string) []TODO {
	var todos []TODO

	// Match TODO patterns
	regex := regexp.MustCompile(`(?:TODO|FIXME|XXX|HACK)(?:\(([a-z]+)\))?:?\s*(.+)`)
	matches := regex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		todo := TODO{
			Raw:   match[0],
			Level: match[1],
			Text:  strings.TrimSpace(match[2]),
		}
		if todo.Level == "" {
			todo.Level = "normal"
		}
		todos = append(todos, todo)
	}

	return todos
}

// TODO represents a TODO comment.
type TODO struct {
	Raw   string // Original comment
	Level string // Priority level (low, normal, high, urgent)
	Text  string // TODO text
}
