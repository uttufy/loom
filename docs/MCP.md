# MCP Integration

Loom provides a Model Context Protocol (MCP) server for integration with Claude Code and other AI tools.

## Setup

Add to your `.mcp.json`:

```json
{
  "mcpServers": {
    "loom": {
      "command": "loom",
      "args": ["mcp"]
    }
  }
}
```

Or with a specific config:

```json
{
  "mcpServers": {
    "loom": {
      "command": "loom",
      "args": ["mcp", "--config", "/path/to/loom.yaml"]
    }
  }
}
```

## Available Tools

### loom_run

Start the orchestration loop.

**Parameters**: None

**Returns**:
- `status`: Current status
- `message`: Status message

### loom_ready

Get prioritized ready tasks.

**Parameters**:
- `type` (optional): Filter by issue type
- `priority` (optional): Filter by priority

**Returns**:
- `tasks`: Array of scored tasks

### loom_score

Get score breakdown for a task.

**Parameters**:
- `issue_id`: Issue identifier

**Returns**:
- `score`: Total score
- `breakdown`: Score components

### loom_claim

Claim a task with file declarations.

**Parameters**:
- `issue_id`: Issue identifier
- `files` (optional): Array of files to lock

**Returns**:
- `success`: Whether claim succeeded
- `locks`: Created file locks

### loom_release

Release a task and its locks.

**Parameters**:
- `issue_id`: Issue identifier

**Returns**:
- `success`: Whether release succeeded

### loom_retro

Create or view retrospectives.

**Parameters**:
- `action`: "create" or "list"
- `retro_id` (optional): For viewing

**Returns**:
- `retrospectives`: Array of retros

### loom_patterns

List learned patterns.

**Parameters**: None

**Returns**:
- `patterns`: Array of patterns

### loom_status

Get current orchestrator status.

**Parameters**: None

**Returns**:
- `ready_tasks`: Count of ready tasks
- `in_progress`: Count of in-progress tasks
- `context_usage`: Current usage percentage

## Usage in Claude Code

Once configured, Claude Code can use Loom tools:

```
User: What tasks are ready to work on?

Claude: [Uses loom_ready tool]

I found 3 ready tasks:

1. [issue-42] Implement authentication (score: 8)
   - Blocks 2 other tasks
   - Priority: P1

2. [issue-38] Fix login bug (score: 5)
   - Priority: P1
   - Previously failed once

3. [issue-45] Update documentation (score: 3)
   - Age: 5 days (stale)

I recommend starting with issue-42 as it has the highest score
and is blocking other work.
```

```
User: I'll work on issue-42, it involves auth.go and auth_test.go

Claude: [Uses loom_claim tool]

Claimed issue-42 and locked:
- auth.go
- auth_test.go

No conflicts detected. You're clear to proceed.
```

```
User: I'm done with issue-42

Claude: [Uses loom_release tool]

Released issue-42. Should I close it with a summary?
```

## Workflow Example

### Starting a Session

```
1. Check status
   loom_status -> 5 ready tasks, 2 in progress

2. Get prioritized tasks
   loom_ready -> Ranked list

3. Claim top task
   loom_claim issue-42 src/auth.go
```

### During Work

```
4. Check for conflicts periodically
   loom_status -> Check locks

5. Claim additional files if needed
   loom_claim issue-42 src/middleware.go
```

### Ending a Session

```
6. Create retrospective
   loom_retro create -> New retro

7. Check learned patterns
   loom_patterns -> Updated patterns
```

## Error Handling

Tools return errors in a standard format:

```json
{
  "error": {
    "code": "CONFLICT",
    "message": "File auth.go locked by agent-2",
    "details": {
      "file": "auth.go",
      "agent": "agent-2",
      "issue": "issue-38"
    }
  }
}
```

Common error codes:
- `CONFLICT`: File lock conflict
- `NOT_FOUND`: Issue not found
- `BLOCKED`: Hook blocked operation
- `INVALID`: Invalid parameters

## Security

The MCP server respects the safety configuration:

- Destructive commands are blocked
- Confirmation-required commands need approval
- File access is limited to project directory

Override with:

```json
{
  "mcpServers": {
    "loom": {
      "command": "loom",
      "args": ["mcp", "--unsafe"]
    }
  }
}
```

**Warning**: `--unsafe` disables all safety checks.
