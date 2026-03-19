// Package hooks provides the lifecycle hook engine for Loom.
package hooks

import (
	"context"
	"fmt"
)

// Event represents a hook lifecycle event.
type Event string

const (
	EventPrePrompt    Event = "pre-prompt"
	EventPreToolCall  Event = "pre-tool-call"
	EventPostToolCall Event = "post-tool-call"
	EventPostResponse Event = "post-response"
	EventOnError      Event = "on-error"
	EventOnClaim      Event = "on-claim"
	EventPreClose     Event = "pre-close"
	EventOnBlock      Event = "on-block"
)

// Context provides context for hook execution.
type Context struct {
	Event    Event
	IssueID  string
	AgentID  string
	ToolCall *ToolCall
	Response string
	Error    error
	Metadata map[string]any
}

// ToolCall represents a tool call being validated.
type ToolCall struct {
	ID       string
	Name     string
	Command  string
	Args     map[string]any
	Approved bool
}

// Result is the result of hook execution.
type Result struct {
	Block      bool   `json:"block"`
	Reason     string `json:"reason,omitempty"`
	Modified   bool   `json:"modified"`
	NewContext string `json:"new_context,omitempty"`
	Data       map[string]any `json:"data,omitempty"`
}

// Handler is a function that handles a hook event.
type Handler func(ctx context.Context, hc *Context) (*Result, error)

// Engine manages hook registration and execution.
type Engine struct {
	handlers map[Event][]Handler
}

// NewEngine creates a new hook engine.
func NewEngine() *Engine {
	return &Engine{
		handlers: make(map[Event][]Handler),
	}
}

// Register adds a handler for an event.
func (e *Engine) Register(event Event, handler Handler) {
	e.handlers[event] = append(e.handlers[event], handler)
}

// Execute runs all handlers for an event.
func (e *Engine) Execute(ctx context.Context, hc *Context) (*Result, error) {
	var result *Result

	for _, handler := range e.handlers[hc.Event] {
		r, err := handler(ctx, hc)
		if err != nil {
			return nil, fmt.Errorf("hook failed for %s: %w", hc.Event, err)
		}

		if r != nil {
			if result == nil {
				result = r
			} else {
				// Merge results
				if r.Block {
					result.Block = true
					result.Reason = r.Reason
				}
				if r.Modified {
					result.Modified = true
					result.NewContext = r.NewContext
				}
				for k, v := range r.Data {
					if result.Data == nil {
						result.Data = make(map[string]any)
					}
					result.Data[k] = v
				}
			}

			// Stop execution if blocked
			if r.Block {
				break
			}
		}
	}

	if result == nil {
		result = &Result{}
	}

	return result, nil
}

// HasHandlers returns true if there are handlers for an event.
func (e *Engine) HasHandlers(event Event) bool {
	return len(e.handlers[event]) > 0
}

// Clear removes all handlers for an event.
func (e *Engine) Clear(event Event) {
	delete(e.handlers, event)
}

// List returns all registered handlers for an event.
func (e *Engine) List(event Event) []Handler {
	return e.handlers[event]
}
