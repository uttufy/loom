// Package safety provides safety guards for destructive operations.
package safety

import (
	"regexp"
	"sync"
)

// Guard validates tool calls for safety.
type Guard struct {
	destructivePatterns     []*regexp.Regexp
	requireConfirmation     []*regexp.Regexp
	allowedWithoutApproval  []*regexp.Regexp
	userApproved            map[string]bool
	mu                      sync.RWMutex
}

// NewGuard creates a new safety guard.
func NewGuard() *Guard {
	g := &Guard{
		userApproved: make(map[string]bool),
	}

	// Destructive commands that are always blocked without approval
	destructive := []string{
		`rm\s+(-[rf]+\s+|.+ -[rf]+)`, // rm -rf
		`git\s+push\s+.*--force`,     // git push --force
		`git\s+reset\s+--hard`,       // git reset --hard
		`DROP\s+(TABLE|DATABASE)`,    // SQL DROP
		`TRUNCATE\s+TABLE?`,          // SQL TRUNCATE
		`DELETE\s+FROM`,              // SQL DELETE
		`:\(\s*\)\s*\{\s*:\|\s*:&\s*\}\s*;:`, // Fork bomb
		`mkfs`,                       // Format filesystem
		`dd\s+if=.*of=/dev/`,         // dd to device
		`chmod\s+(-R\s+)?000`,        // Remove all permissions
		`chown\s+(-R\s+)?root:root\s+/`, // Change root ownership
	}

	for _, pattern := range destructive {
		g.destructivePatterns = append(g.destructivePatterns, regexp.MustCompile("(?i)"+pattern))
	}

	// Commands requiring user confirmation
	requireConfirm := []string{
		`git\s+push`,
		`npm\s+publish`,
		`docker\s+push`,
		`cargo\s+publish`,
		`go\s+mod\s+tidy`, // Can remove dependencies
	}

	for _, pattern := range requireConfirm {
		g.requireConfirmation = append(g.requireConfirmation, regexp.MustCompile("(?i)"+pattern))
	}

	// Commands always allowed
	allowed := []string{
		`git\s+status`,
		`git\s+diff`,
		`git\s+log`,
		`git\s+branch`,
		`cat\s+`,
		`ls\s*`,
		`head\s+`,
		`tail\s+`,
		`grep\s+`,
		`find\s+`,
	}

	for _, pattern := range allowed {
		g.allowedWithoutApproval = append(g.allowedWithoutApproval, regexp.MustCompile("(?i)"+pattern))
	}

	return g
}

// Validate checks if a command is safe to execute.
func (g *Guard) Validate(toolName, command string) error {
	// Check if always allowed
	for _, pattern := range g.allowedWithoutApproval {
		if pattern.MatchString(command) {
			return nil
		}
	}

	// Check if user has approved this specific command
	g.mu.RLock()
	if g.userApproved[command] {
		g.mu.RUnlock()
		return nil
	}
	g.mu.RUnlock()

	// Check for destructive patterns
	for _, pattern := range g.destructivePatterns {
		if pattern.MatchString(command) {
			return &BlockedError{
				Command: command,
				Reason:  "destructive command blocked",
			}
		}
	}

	// Check if confirmation is required
	for _, pattern := range g.requireConfirmation {
		if pattern.MatchString(command) {
			return &ConfirmationRequiredError{
				Command: command,
				Reason:  "command requires user confirmation",
			}
		}
	}

	return nil
}

// Approve marks a command as approved by the user.
func (g *Guard) Approve(command string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.userApproved[command] = true
}

// Revoke removes approval for a command.
func (g *Guard) Revoke(command string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.userApproved, command)
}

// IsApproved checks if a command is approved.
func (g *Guard) IsApproved(command string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.userApproved[command]
}

// BlockedError indicates a command was blocked.
type BlockedError struct {
	Command string
	Reason  string
}

func (e *BlockedError) Error() string {
	return e.Reason + ": " + e.Command
}

// ConfirmationRequiredError indicates a command needs confirmation.
type ConfirmationRequiredError struct {
	Command string
	Reason  string
}

func (e *ConfirmationRequiredError) Error() string {
	return e.Reason + ": " + e.Command
}

// IsBlocked returns true if the error is a BlockedError.
func IsBlocked(err error) bool {
	_, ok := err.(*BlockedError)
	return ok
}

// NeedsConfirmation returns true if the error is a ConfirmationRequiredError.
func NeedsConfirmation(err error) bool {
	_, ok := err.(*ConfirmationRequiredError)
	return ok
}
