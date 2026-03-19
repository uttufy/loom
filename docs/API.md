# Loom API Reference

## Package api

The `pkg/api` package provides the public API for programmatic use of Loom.

### Creating a Loom Instance

```go
import "github.com/uttufy/loom/pkg/api"

// From config file
loom, err := api.NewFromConfigFile("loom.yaml")

// Or with explicit config
cfg := config.DefaultConfig()
loom, err := api.New(cfg)
```

### Core Operations

#### Run the Orchestrator

```go
err := loom.Run(ctx)
```

Starts the main orchestration loop. Runs until context is cancelled or no work remains.

#### Stop the Orchestrator

```go
loom.Stop()
```

Stops a running orchestrator gracefully.

#### Get Ready Tasks

```go
tasks, err := loom.GetReadyTasks(ctx)
for _, task := range tasks {
    fmt.Printf("%s: %s (score: %d)\n", task.Issue.ID, task.Issue.Title, task.Score)
}
```

Returns ready tasks sorted by priority score.

### Task Management

#### Claim a Task

```go
err := loom.ClaimTask(ctx, "issue-123", "my-agent", []string{"src/main.go"})
```

Claims a task and optionally locks files for coordination.

#### Complete a Task

```go
err := loom.CompleteTask(ctx, "issue-123", "Implemented feature X")
```

Marks a task as completed with a summary.

#### Record Task Failure

```go
err := loom.FailTask(ctx, "issue-123", fmt.Errorf("tests failed"))
```

Records a failure and releases the task back to open status.

### Hooks

#### Register a Hook

```go
loom.RegisterHook(hooks.EventPreToolCall, func(ctx context.Context, hc *hooks.Context) (*hooks.Result, error) {
    // Validate tool call
    if hc.ToolCall.Name == "bash" {
        if isDestructive(hc.ToolCall.Command) {
            return &hooks.Result{
                Block:  true,
                Reason: "destructive command blocked",
            }, nil
        }
    }
    return &hooks.Result{}, nil
})
```

### Safety

#### Validate Commands

```go
err := loom.ValidateCommand("bash", "rm -rf /")
if safety.IsBlocked(err) {
    fmt.Println("Command was blocked")
}
```

#### Approve Commands

```go
loom.ApproveCommand("git push origin main")
```

### Coordination

#### Get File Locks

```go
locks := loom.GetLocks()
for _, lock := range locks {
    fmt.Printf("%s locked by %s\n", lock.FilePath, lock.AgentID)
}
```

#### Detect Conflicts

```go
conflicts, err := loom.DetectConflicts(ctx)
for _, c := range conflicts {
    fmt.Printf("Conflict on %s between %s and %s\n", c.File, c.Agent1, c.Agent2)
}
```

### Learning

#### Get Patterns

```go
patterns := loom.GetPatterns()
for _, p := range patterns {
    fmt.Printf("%s: %s\n", p.Name, p.Description)
}
```

#### Save Retrospective

```go
retro := &retrospective.Retrospective{
    SessionID:  "session-123",
    AgentID:    "my-agent",
    Completed:  []string{"issue-1", "issue-2"},
    Strategies: []string{"TDD approach worked well"},
}
err := loom.SaveRetrospective(retro)
```

### Memory

#### Check Memory Stats

```go
stats := loom.GetMemoryStats()
fmt.Printf("Context usage: %.0f%%\n", stats.ContextUsage * 100)
```

#### Check if Compaction Needed

```go
if loom.ShouldCompact(0.75) {
    loom.Compact(ctx)
}
```

### Scoring

#### Score a Single Task

```go
issue := &beadsclient.Issue{
    ID:       "issue-1",
    Title:    "Fix bug",
    Priority: 1,
}
score, err := loom.ScoreTask(ctx, issue)
fmt.Printf("Score: %d\n", score)
```

### Low-Level Access

#### Get Beads Client

```go
client := loom.GetClient()
issues, err := client.Ready(ctx, beadsclient.WorkFilter{})
```

#### Get Configuration

```go
cfg := loom.GetConfig()
fmt.Printf("Compact threshold: %.0f%%\n", cfg.Memory.CompactThreshold * 100)
```

## Types

### ScoredIssue

```go
type ScoredIssue struct {
    Issue *beadsclient.Issue
    Score int
}
```

### FileLock

```go
type FileLock struct {
    FilePath  string
    IssueID   string
    AgentID   string
    ClaimedAt time.Time
    ExpiresAt time.Time
}
```

### Pattern

```go
type Pattern struct {
    ID            string
    Name          string
    Description   string
    Applicability string
    SuccessRate   float64
    Examples      []string
    UseCount      int
}
```

### Retrospective

```go
type Retrospective struct {
    ID          string
    SessionID   string
    AgentID     string
    CreatedAt   time.Time
    Completed   []string
    Failed      []FailedTask
    Strategies  []string
    Patterns    []Pattern
}
```

## Error Types

### BlockedError

```go
err := loom.ValidateCommand("bash", "rm -rf /")
if blocked, ok := err.(*safety.BlockedError); ok {
    fmt.Printf("Blocked: %s\n", blocked.Reason)
}
```

### ConfirmationRequiredError

```go
if confirm, ok := err.(*safety.ConfirmationRequiredError); ok {
    fmt.Printf("Needs confirmation: %s\n", confirm.Command)
}
```

### ConflictError

```go
err := loom.ClaimTask(ctx, issueID, agentID, files)
if conflict, ok := err.(*coordinator.ConflictError); ok {
    fmt.Printf("Conflict on %s with %s\n", conflict.File, conflict.AgentID)
}
```
