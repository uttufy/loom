# Loom - AI Agent Orchestrator for Beads

**"Beads are the task, Loom is what weaves them together"**

Loom is an orchestration layer built on top of [Beads (bd)](https://github.com/uttufy/beads) that enhances AI coding agents with intelligent task prioritization, lifecycle hooks, multi-agent coordination, and learning capabilities.

## Overview

AI coding agents face several challenges when working on long-horizon tasks:

- **Context Loss**: Agents lose context between sessions
- **Poor Prioritization**: Agents don't know which tasks have the most impact
- **Coordination Gaps**: Multiple agents can conflict on shared codebases
- **No Learning**: Agents repeat mistakes without learning from failures
- **Memory Pressure**: Context windows fill with irrelevant data

Loom solves these problems by providing:

- **Smart Prioritization**: Score tasks by downstream impact, staleness, and failure history
- **Lifecycle Hooks**: Inject context, validate commands, auto-create follow-ups
- **Multi-Agent Coordination**: File-level locking and conflict detection
- **Learning System**: Retrospectives and pattern extraction for continuous improvement
- **Memory Management**: Intelligent compaction to preserve important context

## Installation

```bash
# Clone and build
git clone https://github.com/uttufy/loom.git
cd loom
go build ./cmd/loom

# Or install directly
go install github.com/uttufy/loom/cmd/loom@latest
```

## Quick Start

```bash
# Initialize configuration
loom config init

# View prioritized ready tasks
loom ready

# Start the orchestration loop
loom run
```

## CLI Commands

### Core Commands

| Command | Description |
|---------|-------------|
| `loom run` | Start the orchestration loop |
| `loom ready` | Show prioritized ready tasks |
| `loom score <issue-id>` | Show task score breakdown |
| `loom claim <issue-id> [files...]` | Claim a task with file declarations |

### Coordination

| Command | Description |
|---------|-------------|
| `loom locks` | Show current file locks |
| `loom conflicts` | Detect potential conflicts |

### Learning

| Command | Description |
|---------|-------------|
| `loom retro list` | List recent retrospectives |
| `loom retro create` | Create a new retrospective |
| `loom patterns` | List learned patterns |

### Hooks

| Command | Description |
|---------|-------------|
| `loom hooks list` | List registered hooks |
| `loom hooks test <event>` | Test hook execution |

### Memory

| Command | Description |
|---------|-------------|
| `loom status` | Show context usage and stats |
| `loom compact` | Run importance-weighted compaction |

### Configuration

| Command | Description |
|---------|-------------|
| `loom config init` | Initialize loom config |
| `loom config show` | Show current configuration |

## Configuration

Loom is configured via `loom.yaml`:

```yaml
# Beads integration
beads:
  path: bd
  timeout: 30s

# Scoring weights
scoring:
  blocking_multiplier: 3
  priority_boost: 2
  staleness_days: 3
  staleness_bonus: 1
  failure_penalty: 1

# Hook configuration
hooks:
  enabled: true
  pre_prompt:
    - name: inject-context
      builtin: context-injector

# Safety configuration
safety:
  block_destructive: true

# Memory management
memory:
  compact_threshold: 0.70

# Coordination
coordination:
  enabled: true
  lock_timeout: 1h

# Learning
learning:
  enabled: true
  retro_count: 3
```

## Task Scoring

Loom ranks ready tasks using an intelligent scoring formula:

```
Score = (blocking_count × 3) + (priority_boost × 2) + staleness_bonus - failure_penalty
```

- **+3** for tasks blocking 2+ other tasks
- **+2** for P0/P1 priority tasks
- **+1** for tasks open > 3 days
- **-1** for previously failed tasks

## Lifecycle Hooks

Hooks are executed at key points in the orchestration loop:

| Hook | Purpose |
|------|---------|
| `pre-prompt` | Inject context before prompt processing |
| `pre-tool-call` | Validate tool calls, block destructive commands |
| `post-tool-call` | Truncate large outputs, attach snippets |
| `post-response` | Auto-create follow-up beads |
| `on-error` | Log failures, attempt recovery |
| `on-claim` | Run setup/linting on claim |
| `pre-close` | Verify tests pass before closing |
| `on-block` | Notify for reprioritization |

## Multi-Agent Coordination

When multiple agents work on the same codebase:

```bash
# Claim task with file declarations
loom claim issue-123 src/auth.go src/auth_test.go

# Check for conflicts
loom conflicts
```

File locks prevent simultaneous modifications and are automatically released when tasks complete or expire.

## Learning System

Loom learns from each session:

```bash
# View learned patterns
loom patterns

# Create a retrospective
loom retro create
```

Patterns are stored globally in `~/.beads-global/patterns.json` and transfer across projects.

## Integration with Claude Code

Add Loom to your `.mcp.json`:

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

## Documentation

- [Architecture](./docs/ARCHITECTURE.md) - System design and components
- [API Reference](./docs/API.md) - Public API documentation
- [Hooks](./docs/HOOKS.md) - Hook system details
- [MCP Integration](./docs/MCP.md) - Claude Code integration

## License

MIT
