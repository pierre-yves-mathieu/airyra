# Airyra CLI (`ar`) - Architecture Plan

## 1. Overview

The CLI is a thin client that delegates all task operations to the HTTP server. It handles:
- Project context discovery (via `airyra.toml`)
- Agent identity generation
- Command parsing and validation
- HTTP communication with the server
- Output formatting (human-readable and JSON)

---

## 2. Component Structure

```
ar (CLI binary)
├── cmd/                    # Command definitions
│   ├── root                # Base command, help system
│   ├── server              # start, stop, status
│   ├── init                # Project initialization
│   ├── task                # create, list, show, edit, delete
│   ├── status              # claim, done, release, block, unblock
│   ├── dep                 # add, rm, list
│   ├── ready               # ready, next
│   └── history             # history, log
│
├── config/                 # Configuration handling
│   ├── project             # airyra.toml discovery & parsing
│   └── global              # ~/.airyra/config.toml (optional)
│
├── client/                 # HTTP API client
│   ├── http                # Request/response handling
│   └── errors              # Error parsing & formatting
│
├── identity/               # Agent ID generation
│
└── output/                 # Output formatting
    ├── table               # Human-readable tables
    └── json                # JSON output for agents
```

---

## 3. Configuration Discovery

### 3.1 Project Config (`airyra.toml`)

**Discovery Algorithm:**
1. Start from current working directory
2. Check for `airyra.toml` in current directory
3. If not found, traverse up to parent directory
4. Repeat until found or filesystem root reached
5. If not found, return error with helpful message

**Schema:**
```toml
# airyra.toml

# Required
project = "my-app"

# Optional - server connection (overrides global config)
[server]
host = "localhost"      # default: "localhost"
port = 7432             # default: 7432
```

**Parsed Fields:**
| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `project` | Yes | - | Project name used in API URLs |
| `server.host` | No | "localhost" | Server hostname or IP |
| `server.port` | No | 7432 | Server port |

**Use Cases for Custom Server:**
- Team shared server on local network (`host = "192.168.1.100"`)
- Multiple server instances for different environments
- Non-standard port to avoid conflicts

**Error Cases:**
- No `airyra.toml` found → "No airyra.toml found. Run 'ar init <name>' to create one."
- Invalid TOML syntax → "Invalid airyra.toml: {parse error}"
- Missing `project` field → "airyra.toml missing required 'project' field"
- Invalid port (not 1-65535) → "Invalid server.port: must be between 1 and 65535"

### 3.2 Global Config (`~/.airyra/config.toml`)

**Schema:**
```toml
# ~/.airyra/config.toml

[server]
host = "localhost"
port = 7432
```

**Purpose:** Default server settings when project config doesn't specify them.

### 3.3 Configuration Precedence

Server connection settings are resolved in order (first wins):

1. **Project config** (`airyra.toml` in project root)
2. **Global config** (`~/.airyra/config.toml`)
3. **Built-in defaults** (`localhost:7432`)

```
airyra.toml [server.host] → ~/.airyra/config.toml [server.host] → "localhost"
airyra.toml [server.port] → ~/.airyra/config.toml [server.port] → 7432
```

This allows:
- Per-project server targeting (team servers, isolated environments)
- User-level defaults for non-standard setups
- Zero-config for typical local usage

---

## 4. Agent Identity Generation

**Format:** `{user}@{hostname}:{cwd}`

**Components:**
| Component | Source | Fallback |
|-----------|--------|----------|
| user | `os.Getenv("USER")` or `user.Current()` | "unknown" |
| hostname | `os.Hostname()` | "localhost" |
| cwd | `os.Getwd()` | "." |

**Examples:**
- `alice@macbook:/Users/alice/projects/myapp`
- `dev@server:/home/dev/backend`

**Usage:**
- Sent as `X-Airyra-Agent` header on all requests
- Recorded in audit log for traceability
- Used for claim ownership verification

---

## 5. HTTP Client Layer

### 5.1 Request Flow

```
Command → Validate Args → Load Config → Build Request → Send → Parse Response → Format Output
                              │
                              ├── Base URL: http://{host}:{port}
                              │              (from config precedence)
                              ├── Path: /v1/projects/{project}/...
                              └── Headers:
                                  └── X-Airyra-Agent: {agent-id}
```

### 5.2 Server Availability Check

Before any project operation:
1. Build server URL from resolved config
2. Check if server is reachable (GET `/v1/health`)
3. If connection refused → "Error: airyra server not running at {host}:{port}\nStart with: ar server start"

### 5.3 Response Handling

**Success responses:**
- Parse JSON body
- Extract `data` field for display
- Track `updated_at` for optimistic locking warnings

**Error responses:**
- Parse error envelope: `{error: {code, message, context}}`
- Format user-friendly message based on error code
- Include context details where helpful

### 5.4 Pagination Support

For list commands:
- Accept `--page` and `--per-page` flags
- Pass as query parameters
- Display pagination info in output footer

---

## 6. Command Structure

### 6.1 Command Groups

| Group | Commands | Requires Project |
|-------|----------|------------------|
| Server | `server start`, `server stop`, `server status` | No |
| Setup | `init` | No (creates it) |
| Tasks | `create`, `list`, `show`, `edit`, `delete` | Yes |
| Status | `claim`, `done`, `release`, `block`, `unblock` | Yes |
| Deps | `dep add`, `dep rm`, `dep list` | Yes |
| Queue | `ready`, `next` | Yes |
| History | `history`, `log` | Yes |

### 6.2 Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON (for AI agents) |
| `--help` | Show command help |
| `--version` | Show CLI version |

### 6.3 Command → API Mapping

| Command | Method | Endpoint |
|---------|--------|----------|
| `create` | POST | `/v1/projects/{p}/tasks` |
| `list` | GET | `/v1/projects/{p}/tasks` |
| `show <id>` | GET | `/v1/projects/{p}/tasks/{id}` |
| `edit <id>` | PATCH | `/v1/projects/{p}/tasks/{id}` |
| `delete <id>` | DELETE | `/v1/projects/{p}/tasks/{id}` |
| `claim <id>` | POST | `/v1/projects/{p}/tasks/{id}/claim` |
| `done <id>` | POST | `/v1/projects/{p}/tasks/{id}/done` |
| `release <id>` | POST | `/v1/projects/{p}/tasks/{id}/release` |
| `ready` | GET | `/v1/projects/{p}/tasks/ready` |
| `next` | GET | `/v1/projects/{p}/tasks/ready?per_page=1` |

---

## 7. Output Formatting

### 7.1 Human-Readable Mode (default)

**Task List:**
```
ID        STATUS       PRI  TITLE
ar-a1b2   open         2    Build auth system
ar-c3d4   in_progress  1    Fix login bug
```

**Single Task:**
```
ar-a1b2: Build auth system
Status: open | Priority: 2 (normal)
Created: 2024-01-15 10:30
Dependencies: ar-x1y2 (done), ar-z3w4 (open)

Description:
  Implement the authentication system with JWT tokens...
```

### 7.2 JSON Mode (`--json`)

Direct pass-through of API response for machine parsing.

---

## 8. Error Handling Strategy

### 8.1 Error Categories

| Category | Handling |
|----------|----------|
| No config | Show init instructions |
| Server down | Show start instructions with configured host:port |
| Network error | Retry once, then fail with details |
| API error | Parse error envelope, show formatted message |
| Validation | Show field-specific errors |

### 8.2 Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Server not running |
| 3 | Project not configured |
| 4 | Task not found |
| 5 | Permission denied (not owner) |
| 6 | Conflict (already claimed, cycle, etc.) |

---

## 9. Server Management (Special Case)

Server commands use global config only (not project config) since they manage the local server instance.

### `ar server start`
1. Check if PID file exists (`~/.airyra/airyra.pid`)
2. If exists, check if process is alive
3. If alive → "Server already running (PID: xxx)"
4. If stale → Remove PID file
5. Start server binary as background process
6. Wait for health check to pass
7. Report success with PID

### `ar server stop`
1. Read PID from file
2. Send SIGTERM to process
3. Wait for graceful shutdown (timeout: 5s)
4. If still running, SIGKILL
5. Remove PID file

### `ar server status`
1. Check PID file existence
2. Verify process is running
3. Call health endpoint
4. Report: running/stopped/unhealthy

**Note:** Server management commands control the local server at the default/global config address. Projects pointing to remote servers would manage those servers separately.

---

## 10. Init Command

### `ar init <name>`

Creates `airyra.toml` with optional server settings:

```bash
ar init my-app                           # Basic init
ar init my-app --host 192.168.1.50       # Custom host
ar init my-app --port 8080               # Custom port
ar init my-app --host 10.0.0.5 --port 9000  # Both
```

**Generated file examples:**

Basic:
```toml
project = "my-app"
```

With custom server:
```toml
project = "my-app"

[server]
host = "192.168.1.50"
port = 9000
```

---

## 11. Optimistic Locking Flow

For commands that modify tasks:

1. User runs `ar show ar-a1b2` → CLI caches `updated_at`
2. User runs `ar edit ar-a1b2 -t "New title"`
3. Server returns 409 CONFLICT if task changed
4. CLI displays warning with diff:
   ```
   Warning: Task ar-a1b2 was modified since you last read it.
   Current title: "Different title" (changed by user@host:/path)
   Proceed anyway? [y/N]
   ```
5. On confirmation, retry with force flag

---

## 12. Dependencies

**External:**
- TOML parser (for config files)
- HTTP client (standard library)
- CLI framework (cobra or similar)

**No dependencies on:**
- Database drivers (all storage is server-side)
- File system operations (beyond config reading)

---

## 13. Testing Strategy

| Layer | Test Type |
|-------|-----------|
| Config discovery | Unit tests with temp directories |
| Config precedence | Unit tests verifying merge order |
| Agent ID generation | Unit tests with mocked OS calls |
| HTTP client | Integration tests with mock server |
| Commands | End-to-end tests with real server |
| Output formatting | Snapshot tests for table/JSON output |

---

## 14. Complete Test Suite

### 14.1 Config Discovery Tests (`config/project_test.go`)

**Finding `airyra.toml`:**

| Test | Description |
|------|-------------|
| `TestDiscovery_CurrentDirectory` | Finds config in current working directory |
| `TestDiscovery_ParentDirectory` | Finds config in parent when not in cwd |
| `TestDiscovery_DeeplyNested` | Finds config when 3+ levels deep in project |
| `TestDiscovery_RootDirectory` | Returns error when started from filesystem root |
| `TestDiscovery_NotFound` | Returns helpful error with init instructions |

**Parsing `airyra.toml`:**

| Test | Description |
|------|-------------|
| `TestParse_MinimalConfig` | Parses config with only `project` field |
| `TestParse_FullConfig` | Parses config with project and all server fields |
| `TestParse_PartialServer` | Parses config with only host or only port |
| `TestParse_InvalidTOML` | Returns parse error for malformed TOML |
| `TestParse_MissingProject` | Returns error when `project` field missing |
| `TestParse_EmptyProject` | Returns error when `project` is empty string |
| `TestParse_InvalidPortZero` | Returns error when port is 0 |
| `TestParse_InvalidPortNegative` | Returns error when port is negative |
| `TestParse_InvalidPortTooHigh` | Returns error when port > 65535 |
| `TestParse_InvalidPortNonInteger` | Returns error when port is not a number |
| `TestParse_ExtraFieldsIgnored` | Ignores unknown fields without error |

### 14.2 Global Config Tests (`config/global_test.go`)

| Test | Description |
|------|-------------|
| `TestGlobal_FileExists` | Loads and parses existing global config |
| `TestGlobal_FileNotExists` | Returns empty config when file missing |
| `TestGlobal_InvalidTOML` | Returns parse error for malformed TOML |
| `TestGlobal_DirectoryNotExists` | Returns empty config when ~/.airyra doesn't exist |
| `TestGlobal_PartialConfig` | Loads config with only host or only port |

### 14.3 Config Precedence Tests (`config/resolve_test.go`)

| Test | Description |
|------|-------------|
| `TestPrecedence_ProjectOverridesGlobal` | Project config host/port wins over global |
| `TestPrecedence_GlobalOverridesDefaults` | Global config wins over built-in defaults |
| `TestPrecedence_DefaultsUsed` | Uses localhost:7432 when no config specifies server |
| `TestPrecedence_MixedSources` | Project host + global port + default |
| `TestPrecedence_ProjectHostOnly` | Uses project host, falls back for port |
| `TestPrecedence_GlobalHostOnly` | Uses global host when project doesn't specify |

### 14.4 Agent Identity Tests (`identity/agent_test.go`)

| Test | Description |
|------|-------------|
| `TestIdentity_FullFormat` | Generates `user@hostname:/path` format |
| `TestIdentity_FallbackUser` | Uses "unknown" when USER env var not set |
| `TestIdentity_FallbackHostname` | Uses "localhost" when hostname lookup fails |
| `TestIdentity_FallbackCwd` | Uses "." when getwd fails |
| `TestIdentity_SpecialCharactersInPath` | Handles spaces and special chars in cwd |
| `TestIdentity_Deterministic` | Same inputs produce same identity |

### 14.5 HTTP Client Tests (`client/http_test.go`)

**Request Building:**

| Test | Description |
|------|-------------|
| `TestRequest_BaseURL` | Builds correct URL from host:port config |
| `TestRequest_AgentHeader` | Includes X-Airyra-Agent header on all requests |
| `TestRequest_ContentType` | Sets Content-Type: application/json for POST/PATCH |
| `TestRequest_ProjectInPath` | Correctly interpolates project name in URL path |

**Server Availability:**

| Test | Description |
|------|-------------|
| `TestHealth_ServerRunning` | Health check passes when server responds |
| `TestHealth_ServerDown` | Returns exit code 2 with helpful message |
| `TestHealth_ConnectionRefused` | Handles connection refused gracefully |
| `TestHealth_Timeout` | Handles timeout with appropriate error |

**Response Handling:**

| Test | Description |
|------|-------------|
| `TestResponse_Success200` | Parses successful response with data field |
| `TestResponse_Success201` | Parses created response |
| `TestResponse_Success204` | Handles no-content response |
| `TestResponse_Error400` | Parses validation error envelope |
| `TestResponse_Error404` | Parses not found error, returns exit code 4 |
| `TestResponse_Error409` | Parses conflict error, returns exit code 6 |
| `TestResponse_Error500` | Handles server error gracefully |
| `TestResponse_MalformedJSON` | Handles non-JSON error response |
| `TestResponse_NetworkError` | Retries once then fails with details |

**Pagination:**

| Test | Description |
|------|-------------|
| `TestPagination_DefaultValues` | Uses page=1, per_page=20 by default |
| `TestPagination_CustomPage` | Passes custom page parameter |
| `TestPagination_CustomPerPage` | Passes custom per_page parameter |
| `TestPagination_ParsesMetadata` | Extracts total_count, total_pages from response |

### 14.6 Error Handling Tests (`client/errors_test.go`)

| Test | Description |
|------|-------------|
| `TestError_ParseEnvelope` | Parses {error: {code, message, context}} |
| `TestError_FormatTaskNotFound` | Formats TASK_NOT_FOUND with task ID |
| `TestError_FormatNotOwner` | Formats NOT_OWNER with owner info |
| `TestError_FormatAlreadyClaimed` | Formats ALREADY_CLAIMED with claimer info |
| `TestError_FormatCycleDetected` | Formats CYCLE_DETECTED with cycle path |
| `TestError_FormatValidation` | Formats field-specific validation errors |
| `TestError_UnknownCode` | Falls back to generic message for unknown codes |
| `TestExitCode_Mapping` | Verifies all error codes map to correct exit codes |

### 14.7 Output Formatting Tests (`output/`)

**Table Output (`output/table_test.go`):**

| Test | Description |
|------|-------------|
| `TestTable_TaskList` | Formats task list with ID, STATUS, PRI, TITLE columns |
| `TestTable_TaskListEmpty` | Shows "No tasks found" for empty list |
| `TestTable_TaskShow` | Formats single task with all fields |
| `TestTable_TaskShowWithDeps` | Includes dependency list with statuses |
| `TestTable_TaskShowNoDeps` | Handles task with no dependencies |
| `TestTable_TruncateLongTitle` | Truncates titles exceeding column width |
| `TestTable_PriorityLabels` | Displays priority as number + label |
| `TestTable_StatusColors` | Applies correct colors to statuses |
| `TestTable_DateFormatting` | Formats dates as YYYY-MM-DD HH:MM |
| `TestTable_PaginationFooter` | Shows "Page X of Y (Z total)" footer |

**JSON Output (`output/json_test.go`):**

| Test | Description |
|------|-------------|
| `TestJSON_Passthrough` | Outputs API response as-is |
| `TestJSON_PrettyPrint` | Optionally pretty-prints with indentation |
| `TestJSON_ErrorFormat` | Outputs errors in JSON format with --json |

### 14.8 Init Command Tests (`cmd/init_test.go`)

| Test | Description |
|------|-------------|
| `TestInit_CreatesMinimalConfig` | Creates airyra.toml with just project name |
| `TestInit_WithHost` | Creates config with [server] host |
| `TestInit_WithPort` | Creates config with [server] port |
| `TestInit_WithHostAndPort` | Creates config with both host and port |
| `TestInit_ConfigAlreadyExists` | Returns error if airyra.toml exists |
| `TestInit_InvalidProjectName` | Rejects empty or invalid project names |
| `TestInit_InvalidPort` | Rejects port outside 1-65535 range |
| `TestInit_CreatesValidTOML` | Output is parseable TOML |

### 14.9 Server Management Tests (`cmd/server_test.go`)

**Server Start:**

| Test | Description |
|------|-------------|
| `TestServerStart_Success` | Starts server, creates PID file, reports success |
| `TestServerStart_AlreadyRunning` | Returns error with PID when already running |
| `TestServerStart_StalePIDFile` | Removes stale PID file and starts |
| `TestServerStart_BinaryNotFound` | Returns error if server binary missing |
| `TestServerStart_HealthCheckFails` | Returns error if server doesn't become healthy |
| `TestServerStart_UsesGlobalConfig` | Uses global config port, ignores project config |

**Server Stop:**

| Test | Description |
|------|-------------|
| `TestServerStop_GracefulShutdown` | Sends SIGTERM and removes PID file |
| `TestServerStop_ForceKill` | Sends SIGKILL after timeout |
| `TestServerStop_NotRunning` | Returns error if server not running |
| `TestServerStop_StalePIDFile` | Removes stale PID file |

**Server Status:**

| Test | Description |
|------|-------------|
| `TestServerStatus_Running` | Reports running with PID and health status |
| `TestServerStatus_Stopped` | Reports stopped when no PID file |
| `TestServerStatus_Unhealthy` | Reports unhealthy when process alive but health fails |
| `TestServerStatus_StalePID` | Reports stopped when PID file stale |

### 14.10 Task Command Tests (`cmd/task_test.go`)

**Create:**

| Test | Description |
|------|-------------|
| `TestCreate_MinimalTask` | Creates task with just title |
| `TestCreate_FullTask` | Creates task with title, description, priority |
| `TestCreate_WithDependencies` | Creates task with initial dependencies |
| `TestCreate_OutputsTaskID` | Displays created task ID |
| `TestCreate_JSONOutput` | Outputs full task JSON with --json |
| `TestCreate_MissingTitle` | Returns error when title not provided |
| `TestCreate_InvalidPriority` | Returns error for invalid priority value |

**List:**

| Test | Description |
|------|-------------|
| `TestList_AllTasks` | Lists all tasks for project |
| `TestList_FilterByStatus` | Filters by --status flag |
| `TestList_Pagination` | Handles --page and --per-page flags |
| `TestList_EmptyProject` | Shows "No tasks found" message |
| `TestList_JSONOutput` | Outputs task array with --json |
| `TestList_SortOrder` | Tasks sorted by priority then created_at |

**Show:**

| Test | Description |
|------|-------------|
| `TestShow_ExistingTask` | Displays full task details |
| `TestShow_NotFound` | Returns exit code 4 for unknown task |
| `TestShow_WithDependencies` | Shows dependencies with their statuses |
| `TestShow_JSONOutput` | Outputs full task JSON with --json |

**Edit:**

| Test | Description |
|------|-------------|
| `TestEdit_Title` | Updates task title |
| `TestEdit_Description` | Updates task description |
| `TestEdit_Priority` | Updates task priority |
| `TestEdit_MultipleFields` | Updates multiple fields at once |
| `TestEdit_NotFound` | Returns exit code 4 for unknown task |
| `TestEdit_NoChanges` | Returns error when no fields specified |
| `TestEdit_ConflictDetected` | Handles 409 conflict with warning |

**Delete:**

| Test | Description |
|------|-------------|
| `TestDelete_ExistingTask` | Deletes task and confirms |
| `TestDelete_NotFound` | Returns exit code 4 for unknown task |
| `TestDelete_WithDependents` | Returns error if other tasks depend on it |
| `TestDelete_JSONOutput` | Outputs confirmation with --json |

### 14.11 Status Command Tests (`cmd/status_test.go`)

**Claim:**

| Test | Description |
|------|-------------|
| `TestClaim_AvailableTask` | Claims open task, updates status to in_progress |
| `TestClaim_AlreadyClaimed` | Returns exit code 6 with owner info |
| `TestClaim_NotFound` | Returns exit code 4 |
| `TestClaim_DependenciesNotMet` | Returns error if dependencies not done |
| `TestClaim_SetsAgentAsOwner` | Records claiming agent identity |
| `TestClaim_JSONOutput` | Outputs updated task with --json |

**Done:**

| Test | Description |
|------|-------------|
| `TestDone_OwnedTask` | Marks claimed task as done |
| `TestDone_NotOwner` | Returns exit code 5 when not owner |
| `TestDone_NotClaimed` | Returns error for unclaimed task |
| `TestDone_NotFound` | Returns exit code 4 |
| `TestDone_JSONOutput` | Outputs updated task with --json |

**Release:**

| Test | Description |
|------|-------------|
| `TestRelease_OwnedTask` | Releases claimed task back to open |
| `TestRelease_NotOwner` | Returns exit code 5 when not owner |
| `TestRelease_NotClaimed` | Returns error for unclaimed task |
| `TestRelease_NotFound` | Returns exit code 4 |
| `TestRelease_JSONOutput` | Outputs updated task with --json |

**Block:**

| Test | Description |
|------|-------------|
| `TestBlock_OwnedTask` | Blocks task with reason |
| `TestBlock_NotOwner` | Returns exit code 5 when not owner |
| `TestBlock_MissingReason` | Returns error when reason not provided |
| `TestBlock_JSONOutput` | Outputs updated task with --json |

**Unblock:**

| Test | Description |
|------|-------------|
| `TestUnblock_BlockedTask` | Unblocks task, returns to in_progress |
| `TestUnblock_NotOwner` | Returns exit code 5 when not owner |
| `TestUnblock_NotBlocked` | Returns error for non-blocked task |
| `TestUnblock_JSONOutput` | Outputs updated task with --json |

### 14.12 Dependency Command Tests (`cmd/dep_test.go`)

**Add Dependency:**

| Test | Description |
|------|-------------|
| `TestDepAdd_ValidDependency` | Adds dependency between tasks |
| `TestDepAdd_TaskNotFound` | Returns exit code 4 for unknown source task |
| `TestDepAdd_DependencyNotFound` | Returns exit code 4 for unknown target task |
| `TestDepAdd_WouldCreateCycle` | Returns exit code 6 with cycle path |
| `TestDepAdd_AlreadyExists` | Returns error if dependency exists |
| `TestDepAdd_SelfDependency` | Returns error for self-referential dependency |

**Remove Dependency:**

| Test | Description |
|------|-------------|
| `TestDepRm_ExistingDependency` | Removes dependency |
| `TestDepRm_NotFound` | Returns error if dependency doesn't exist |
| `TestDepRm_TaskNotFound` | Returns exit code 4 for unknown task |

**List Dependencies:**

| Test | Description |
|------|-------------|
| `TestDepList_ShowsDependencies` | Lists all dependencies with statuses |
| `TestDepList_NoDependencies` | Shows "No dependencies" message |
| `TestDepList_JSONOutput` | Outputs dependency array with --json |

### 14.13 Queue Command Tests (`cmd/queue_test.go`)

**Ready:**

| Test | Description |
|------|-------------|
| `TestReady_ListsReadyTasks` | Shows tasks with all dependencies met |
| `TestReady_ExcludesBlocked` | Excludes blocked tasks |
| `TestReady_ExcludesClaimed` | Excludes already claimed tasks |
| `TestReady_EmptyQueue` | Shows "No ready tasks" message |
| `TestReady_SortedByPriority` | Highest priority tasks first |
| `TestReady_JSONOutput` | Outputs task array with --json |

**Next:**

| Test | Description |
|------|-------------|
| `TestNext_ReturnsHighestPriority` | Returns single highest priority ready task |
| `TestNext_EmptyQueue` | Shows "No ready tasks" message |
| `TestNext_JSONOutput` | Outputs single task with --json |

### 14.14 History Command Tests (`cmd/history_test.go`)

**History:**

| Test | Description |
|------|-------------|
| `TestHistory_TaskHistory` | Shows status transitions for a task |
| `TestHistory_IncludesAgent` | Shows which agent made each change |
| `TestHistory_IncludesTimestamp` | Shows when each change occurred |
| `TestHistory_TaskNotFound` | Returns exit code 4 |
| `TestHistory_JSONOutput` | Outputs history array with --json |

**Log:**

| Test | Description |
|------|-------------|
| `TestLog_RecentActivity` | Shows recent activity across all tasks |
| `TestLog_FilterByAgent` | Filters by --agent flag |
| `TestLog_Pagination` | Handles --page and --per-page flags |
| `TestLog_JSONOutput` | Outputs log array with --json |

### 14.15 Global Flag Tests (`cmd/root_test.go`)

| Test | Description |
|------|-------------|
| `TestGlobal_JSONFlag` | --json flag affects all commands |
| `TestGlobal_HelpFlag` | --help shows command help |
| `TestGlobal_VersionFlag` | --version shows CLI version |
| `TestGlobal_UnknownFlag` | Returns error for unknown flags |
| `TestGlobal_NoCommand` | Shows help when no command provided |

### 14.16 Exit Code Tests (`cmd/exitcode_test.go`)

| Test | Description |
|------|-------------|
| `TestExitCode_Success` | Returns 0 on success |
| `TestExitCode_GeneralError` | Returns 1 for general errors |
| `TestExitCode_ServerNotRunning` | Returns 2 when server unreachable |
| `TestExitCode_ProjectNotConfigured` | Returns 3 when no airyra.toml |
| `TestExitCode_TaskNotFound` | Returns 4 for unknown task |
| `TestExitCode_PermissionDenied` | Returns 5 when not task owner |
| `TestExitCode_Conflict` | Returns 6 for conflicts |

### 14.17 Optimistic Locking Tests (`cmd/locking_test.go`)

| Test | Description |
|------|-------------|
| `TestLocking_ConflictDetected` | Detects when task changed since read |
| `TestLocking_ShowsCurrentState` | Displays current task state in warning |
| `TestLocking_ShowsWhoChanged` | Shows which agent made the change |
| `TestLocking_PromptConfirmation` | Prompts user to confirm override |
| `TestLocking_ForceRetry` | Retries with force flag on confirmation |
| `TestLocking_AbortOnDeny` | Aborts when user denies override |

### 14.18 End-to-End Tests (`e2e/`)

| Test | Description |
|------|-------------|
| `TestE2E_FullWorkflow` | Init → create → claim → done → history |
| `TestE2E_DependencyChain` | Create tasks with deps → resolve in order |
| `TestE2E_MultiAgent` | Two agents claiming/releasing tasks |
| `TestE2E_ConflictResolution` | Concurrent edits trigger locking flow |
| `TestE2E_ServerRestartRecovery` | CLI reconnects after server restart |
| `TestE2E_ProjectIsolation` | Tasks isolated between projects |

### 14.19 Test Infrastructure

**Test Helpers:**

| Helper | Purpose |
|--------|---------|
| `setupTempProject()` | Creates temp dir with airyra.toml |
| `setupGlobalConfig()` | Creates temp ~/.airyra/config.toml |
| `mockServer()` | HTTP test server returning canned responses |
| `captureOutput()` | Captures stdout/stderr for assertions |
| `setEnv()` | Sets env vars and restores after test |

**Fixtures:**

| Fixture | Contents |
|---------|----------|
| `fixtures/valid_config.toml` | Valid minimal config |
| `fixtures/full_config.toml` | Config with all fields |
| `fixtures/invalid_*.toml` | Various invalid configs |
| `fixtures/task_response.json` | Sample API responses |
| `fixtures/error_response.json` | Sample error envelopes |

### 14.20 Test Summary

| Category | Count |
|----------|-------|
| Config Discovery | 5 |
| Config Parsing | 11 |
| Global Config | 5 |
| Config Precedence | 6 |
| Agent Identity | 6 |
| HTTP Client | 18 |
| Error Handling | 8 |
| Output Formatting | 13 |
| Init Command | 8 |
| Server Management | 14 |
| Task Commands | 24 |
| Status Commands | 21 |
| Dependency Commands | 9 |
| Queue Commands | 8 |
| History Commands | 7 |
| Global Flags | 5 |
| Exit Codes | 6 |
| Optimistic Locking | 6 |
| End-to-End | 6 |
| **Total** | **186** |
