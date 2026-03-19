// Package builtin provides built-in hook implementations.
package builtin

import (
	"context"
	"strings"

	"github.com/uttufy/loom/internal/hooks"
	"github.com/uttufy/loom/internal/safety"
)

// SafetyGuard is a pre-tool-call hook that blocks dangerous commands.
func SafetyGuard() hooks.Handler {
	guard := safety.NewGuard()

	return func(ctx context.Context, hc *hooks.Context) (*hooks.Result, error) {
		if hc.ToolCall == nil {
			return &hooks.Result{}, nil
		}

		if err := guard.Validate(hc.ToolCall.Name, hc.ToolCall.Command); err != nil {
			if safety.IsBlocked(err) {
				return &hooks.Result{
					Block:  true,
					Reason: err.Error(),
				}, nil
			}
			if safety.NeedsConfirmation(err) {
				return &hooks.Result{
					Block:  true,
					Reason: "confirmation required: " + err.Error(),
					Data: map[string]any{
						"requires_confirmation": true,
						"command":              hc.ToolCall.Command,
					},
				}, nil
			}
		}

		return &hooks.Result{}, nil
	}
}

// OutputTruncator is a post-tool-call hook that truncates large outputs.
func OutputTruncator(maxSize int) hooks.Handler {
	return func(ctx context.Context, hc *hooks.Context) (*hooks.Result, error) {
		if hc.Metadata == nil {
			return &hooks.Result{}, nil
		}

		output, ok := hc.Metadata["output"].(string)
		if !ok {
			return &hooks.Result{}, nil
		}

		if len(output) > maxSize {
			truncated := output[:maxSize]
			truncated += "\n... [output truncated]"

			return &hooks.Result{
				Modified: true,
				Data: map[string]any{
					"output":     truncated,
					"truncated":  true,
					"original_size": len(output),
				},
			}, nil
		}

		return &hooks.Result{}, nil
	}
}

// ContextInjector is a pre-prompt hook that injects repository context.
func ContextInjector(getContext func() string) hooks.Handler {
	return func(ctx context.Context, hc *hooks.Context) (*hooks.Result, error) {
		contextStr := getContext()

		return &hooks.Result{
			Modified:   true,
			NewContext: contextStr,
			Data: map[string]any{
				"context_injected": true,
			},
		}, nil
	}
}

// FollowupCreator is a post-response hook that creates follow-up tasks.
func FollowupCreator(createTask func(title, taskType string, priority int) error) hooks.Handler {
	return func(ctx context.Context, hc *hooks.Context) (*hooks.Result, error) {
		// Extract TODOs from response
		todos := extractTODOs(hc.Response)

		for _, todo := range todos {
			if err := createTask(todo.text, "task", 3); err != nil {
				continue // Log error but don't block
			}
		}

		return &hooks.Result{
			Data: map[string]any{
				"followups_created": len(todos),
			},
		}, nil
	}
}

type todoItem struct {
	text string
}

func extractTODOs(response string) []todoItem {
	var todos []todoItem

	// Look for common patterns indicating unfinished work
	patterns := []string{
		"TODO:",
		"FIXME:",
		"still need to",
		"remaining:",
		"left to do:",
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		for _, pattern := range patterns {
			if strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
				todos = append(todos, todoItem{text: strings.TrimSpace(line)})
				break
			}
		}
	}

	return todos
}

// ErrorLogger is an on-error hook that logs failures.
func ErrorLogger(logFailure func(issueID, errorMsg string)) hooks.Handler {
	return func(ctx context.Context, hc *hooks.Context) (*hooks.Result, error) {
		if hc.Error == nil {
			return &hooks.Result{}, nil
		}

		logFailure(hc.IssueID, hc.Error.Error())

		return &hooks.Result{
			Data: map[string]any{
				"logged": true,
				"error":  hc.Error.Error(),
			},
		}, nil
	}
}

// TestVerifier is a pre-close hook that verifies tests pass.
func TestVerifier(runTests func() error) hooks.Handler {
	return func(ctx context.Context, hc *hooks.Context) (*hooks.Result, error) {
		if err := runTests(); err != nil {
			return &hooks.Result{
				Block:  true,
				Reason: "tests failed: " + err.Error(),
			}, nil
		}

		return &hooks.Result{
			Data: map[string]any{
				"tests_passed": true,
			},
		}, nil
	}
}

// LintRunner is an on-claim hook that runs linting.
func LintRunner(runLint func() error) hooks.Handler {
	return func(ctx context.Context, hc *hooks.Context) (*hooks.Result, error) {
		if err := runLint(); err != nil {
			return &hooks.Result{
				Data: map[string]any{
					"lint_warnings": err.Error(),
				},
			}, nil
		}

		return &hooks.Result{
			Data: map[string]any{
				"lint_passed": true,
			},
		}, nil
	}
}

// BlockNotifier is an on-block hook that notifies for reprioritization.
func BlockNotifier(notify func(issueID, blockedBy string)) hooks.Handler {
	return func(ctx context.Context, hc *hooks.Context) (*hooks.Result, error) {
		blockedBy, _ := hc.Metadata["blocked_by"].(string)
		notify(hc.IssueID, blockedBy)

		return &hooks.Result{
			Data: map[string]any{
				"notified": true,
			},
		}, nil
	}
}
