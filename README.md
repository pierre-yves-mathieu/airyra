# Airyra

A lightweight task tracker designed for AI agent coordination. Provides atomic task claiming to prevent multiple agents from working on the same task.

## Features

- **Atomic task claiming** - Prevents race conditions when multiple agents try to claim the same task
- **Project isolation** - Separate SQLite database per project
- **Dependency tracking** - Tasks can depend on other tasks
- **Ready queue** - Automatically computes which tasks are actionable
- **Audit log** - Full history of all changes
- **JSON output** - Machine-readable output for AI agents

## Installation

```bash
# Build from source
go build -o airyra ./cmd/airyra

# Move binary to PATH
sudo mv airyra /usr/local/bin/
```

## Getting Started

### 1. Start the server

Airyra requires a running server. Start it with:

```bash
airyra server start
```

The server runs on `localhost:7432` by default. Check status with:

```bash
airyra server status
```

### 2. Initialize a project

In your project directory, create a configuration file:

```bash
airyra init my-project
```

This creates `airyra.toml`:

```toml
project = "my-project"
```

### 3. Create tasks

```bash
# Create a simple task
airyra create "Implement user authentication"

# Create a high-priority task
airyra create "Fix critical bug" -p high

# Create a task with description
airyra create "Add unit tests" -d "Cover all edge cases"

# Create a subtask
airyra create "Write login endpoint" --parent ar-a1b2
```

### 4. Work on tasks

```bash
# See what's ready to work on
airyra ready

# Get the highest-priority task
airyra next

# Claim a task (atomic - prevents others from claiming it)
airyra claim ar-a1b2

# Mark it done when finished
airyra done ar-a1b2
```

### 5. Stop the server

```bash
airyra server stop
```

## CLI Reference

### Server Management

```bash
airyra server start          # Start the server
airyra server stop           # Stop the server
airyra server status         # Check if server is running
```

### Project Setup

```bash
airyra init <name>           # Create airyra.toml in current directory
```

### Task Management

```bash
airyra create <title>        # Create a new task
  -p, --priority <level>     #   Priority: 0-4 or critical/high/normal/low/lowest
  -d, --description <text>   #   Task description
  --parent <id>              #   Parent task ID

airyra list                  # List all tasks
  --status <status>          #   Filter: open, in_progress, blocked, done
  --page <n>                 #   Page number (default: 1)
  --per-page <n>             #   Items per page (default: 50)

airyra show <id>             # Show task details

airyra edit <id>             # Edit a task
  -t, --title <text>         #   New title
  -d, --description <text>   #   New description
  -p, --priority <level>     #   New priority

airyra delete <id>           # Delete a task
```

### Status Transitions

```bash
airyra claim <id>            # Claim task (open → in_progress)
airyra done <id>             # Complete task (in_progress → done)
airyra release <id>          # Release task (in_progress → open)
  --force                    #   Release task claimed by another agent
airyra block <id>            # Block task (→ blocked)
airyra unblock <id>          # Unblock task (blocked → open)
```

### Dependencies

```bash
airyra dep add <child> <parent>  # Add dependency (child depends on parent)
airyra dep rm <child> <parent>   # Remove dependency
airyra dep list <id>             # List task's dependencies
```

### Ready Queue

```bash
airyra ready                 # List all ready tasks
airyra next                  # Get highest-priority ready task
```

### History

```bash
airyra history <id>          # Show task's change history
airyra log                   # Show recent activity
```

### Output Format

Add `--json` to any command for machine-readable output:

```bash
airyra list --json
airyra show ar-a1b2 --json
airyra ready --json
```

## Task States

```
     ┌──────────────────────────────────────┐
     │                                      │
     ▼                                      │
  ┌──────┐  claim   ┌─────────────┐  done  ┌──────┐
  │ open │ ───────► │ in_progress │ ─────► │ done │
  └──────┘          └─────────────┘        └──────┘
     ▲                    │
     │    release         │ block
     └────────────────────┤
                          ▼
                    ┌─────────┐
                    │ blocked │
                    └─────────┘
                          │
                          │ unblock
                          ▼
                       (open)
```

## Priority Levels

| Level | Name     | Use Case |
|-------|----------|----------|
| 0     | critical | Production outages, security issues |
| 1     | high     | Important features, major bugs |
| 2     | normal   | Regular work (default) |
| 3     | low      | Nice-to-have improvements |
| 4     | lowest   | Backlog items |

## For AI Agents

Airyra is designed for AI agent coordination. Key patterns:

### Claim before working

```bash
# Always claim before starting work
airyra claim ar-a1b2
# ... do work ...
airyra done ar-a1b2
```

If another agent already claimed the task, you'll get an error:

```
Error: task already claimed by agent-x at 2024-01-15T10:30:00Z
```

### Use JSON output

```bash
airyra ready --json | jq '.data[0].id'
```

### Agent identification

The CLI automatically identifies agents as `user@hostname:cwd`. This appears in audit logs and claim records.

### Handle errors

Common error codes:
- `ALREADY_CLAIMED` - Task claimed by another agent
- `NOT_OWNER` - Can't complete/release task you don't own
- `INVALID_TRANSITION` - Invalid status change (e.g., claiming a done task)
- `TASK_NOT_FOUND` - Task doesn't exist

## Storage

```
~/.airyra/
├── airyra.pid        # Server PID file
├── airyra.log        # Server logs (10MB, rotated)
└── projects/
    └── my-project.db # SQLite database per project
```

## API

The server exposes a REST API at `http://localhost:7432/v1/`. See [docs/spec/airyra-spec-v2.md](docs/spec/airyra-spec-v2.md) for full API documentation.

## License

MIT
