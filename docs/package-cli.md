# Package CLI Mode

*Last updated: March 10, 2026*

CLI mode allows every Robomotion package to be invoked directly from the command line as a standalone tool, without requiring the gRPC robot runtime. This turns compiled package binaries into self-contained CLI programs that AI agents (and humans) can call like any other shell command.

---

## Table of Contents

1. [Why CLI Mode Exists](#1-why-cli-mode-exists)
2. [The Problem It Solves](#2-the-problem-it-solves)
3. [Architecture Overview](#3-architecture-overview)
4. [How It Works Internally](#4-how-it-works-internally)
5. [Usage Reference](#5-usage-reference)
6. [SKILL.md Generation](#6-skillmd-generation)
7. [Credential Handling](#7-credential-handling)
8. [Flag-to-Variable Mapping](#8-flag-to-variable-mapping)
9. [Output Format](#9-output-format)
10. [Error Handling](#10-error-handling)
11. [Concurrency and Isolation](#11-concurrency-and-isolation)
12. [Which Nodes Are Exposed](#12-which-nodes-are-exposed)
13. [Compatibility with gRPC Mode](#13-compatibility-with-grpc-mode)
14. [Files Reference](#14-files-reference)
15. [Limitations](#15-limitations)

---

## 1. Why CLI Mode Exists

The AI agent ecosystem is converging on a pattern called **skills** — lightweight, self-describing capabilities that an LLM can discover, understand, and invoke through simple shell commands. Projects like Claw, OpenAI's function calling, and Anthropic's tool use all follow this model: the agent reads a short description, decides to call a tool, executes it, and reads the JSON output.

Robomotion packages already contain hundreds of production-grade integrations (Google Drive, Dropbox, Slack, databases, browsers, etc.), each with carefully tested node implementations. CLI mode makes these integrations available to AI agents **without rewriting any node code** — the same Go struct that runs inside a robot flow can now be invoked from `bash`.

### The value chain

```
Package author writes node code once
        ↓
Node works in Robomotion Designer (gRPC mode)    ← existing
Node works as AI tool via tool_interceptor        ← existing
Node works from command line (CLI mode)           ← new
Node auto-generates SKILL.md for agent discovery  ← new
```

---

## 2. The Problem It Solves

### Before CLI mode

For an AI agent to use a Robomotion package, it needed the full robot runtime stack:

```
LLM Agent
  → Python tool_simulation.py
    → gRPC connection to robot process
      → robot spawns package binary as plugin
        → package binary connects back via gRPC
          → node receives message, does work, returns result
        → result flows back through gRPC
      → robot forwards result
    → Python parses gRPC response
  → LLM reads result
```

Problems with this approach:

- **Heavy infrastructure**: Requires a running robot, gRPC broker, plugin handshake, bidirectional streaming — all for a single stateless API call.
- **Context bloat**: Each node becomes a separate tool definition with its full JSON Schema. A package with 15 nodes produces 15 tool definitions, each with dozens of parameters. This consumes thousands of LLM context tokens before the agent does anything useful.
- **No progressive disclosure**: The LLM sees all parameters for all nodes upfront, even though it typically needs just one command with 2-3 parameters.
- **Deployment friction**: Every environment that wants to use the package needs the robot runtime installed and running.

### After CLI mode

```
LLM Agent
  → reads SKILL.md (progressive disclosure — only loads when needed)
  → runs: robomotion-googledrive upload_file --file-path=/tmp/x.pdf --folder-id=abc
  → reads JSON from stdout: {"file_id": "xyz123"}
```

- **Zero infrastructure**: The package binary is the entire runtime. No gRPC, no robot, no broker.
- **Progressive disclosure**: Agent sees one-line skill description in catalog. Only loads full SKILL.md when it decides to use the skill. Only sees the parameters for the specific command it's running.
- **Standard interface**: `stdin`/`stdout`/`stderr` + exit codes. Works with any agent framework, any language, any orchestrator.
- **Parallel by default**: Each CLI invocation is a separate OS process. No shared state, no locks, no connection pools to manage.

---

## 3. Architecture Overview

```
                                    ┌──────────────────────┐
                                    │   AI Agent (Claw)    │
                                    │                      │
                                    │  1. Discovers skill   │
                                    │     via SKILL.md      │
                                    │                      │
                                    │  2. Builds command    │
                                    │     from parameters   │
                                    │                      │
                                    │  3. Executes binary   │
                                    └──────────┬───────────┘
                                               │
                                    shell exec │
                                               ▼
┌──────────────────────────────────────────────────────────────────┐
│                     Package Binary (e.g. robomotion-googledrive) │
│                                                                  │
│  main.go                                                         │
│    runtime.RegisterNodes(&v1.Upload{}, &v1.Download{}, ...)      │
│    runtime.Start()                                               │
│         │                                                        │
│         ▼                                                        │
│  registry.go :: Start()                                          │
│    os.Args[1] = "upload_file" (not a flag)                       │
│         │                                                        │
│         ▼                                                        │
│  cli.go :: RunCLI(["upload_file", "--file-path=/tmp/x.pdf"])     │
│    1. RegisterFactories()     — register all node types          │
│    2. buildCommandMap()       — find nodes with Tool field       │
│    3. parseFlags()            — parse --key=value args           │
│    4. NewCLIRuntimeHelper()   — create mock RuntimeHelper        │
│    5. Set global `client`     — so InVariable.Get() works        │
│    6. Instantiate node        — reflect.New(nodeType)            │
│    7. json.Unmarshal(config)  — populate node fields             │
│    8. buildMessageContext()   — flags → message.Context          │
│    9. OnCreate() → OnMessage(ctx) → OnClose()                   │
│   10. collectCLIOutput()      — read OutVariable values          │
│   11. json.Marshal → stdout   — print result                    │
│                                                                  │
│  cli_runtime.go :: CLIRuntimeHelper                              │
│    Implements RuntimeHelper interface without gRPC               │
│    GetVariable/SetVariable → no-op (Message scope resolves       │
│                               directly from context)             │
│    GetVaultItem → returns credentials from vault or env          │
│                                                                  │
│  cli_vault.go :: CLIVaultClient                                  │
│    Direct HTTP calls to Robomotion vault API                     │
│    AES-256-CBC decryption of vault items                         │
│    RSA-OAEP key derivation matching deskbot vault.go             │
└──────────────────────────────────────────────────────────────────┘
                               │
                               │ stdout (JSON)
                               ▼
                    ┌──────────────────────┐
                    │   AI Agent (Claw)    │
                    │                      │
                    │  Reads JSON output   │
                    │  Continues reasoning │
                    └──────────────────────┘
```

### Key design principle: zero changes to node code

The entire CLI mode is implemented in the **runtime layer**. Package node code (`v1/upload.go`, `v1/download.go`, etc.) is never modified. The same `InVariable.Get(ctx)`, `OutVariable.Set(ctx, val)`, and `Credential.Get(ctx)` calls that work in gRPC mode work identically in CLI mode because:

1. `InVariable.Get(ctx)` for `Scope == "Message"` reads from `ctx.Get(name)` — CLI mode populates the same message context from parsed flags.
2. `OutVariable.Set(ctx, val)` for `Scope == "Message"` calls `ctx.Set(name, val)` — CLI mode reads these back after `OnMessage` completes.
3. `Credential.Get(ctx)` calls `client.GetVaultItem(vaultID, itemID)` — CLI mode's `CLIRuntimeHelper` handles this via vault API or env vars.

---

## 4. How It Works Internally

### Entry point: `Start()` in registry.go

The `Start()` function is the entry point for every package binary. It examines `os.Args[1]` to decide the execution mode:

| Argument | Mode | What happens |
|----------|------|-------------|
| `-s` | Spec generation | Outputs JSON node spec for Designer import |
| `-a` | Attach (debug) | Starts gRPC plugin with debug attachment |
| `--skill-md` | SKILL.md generation | Outputs markdown skill description |
| `--list-commands` | Command listing | Outputs JSON array of available CLI commands |
| `--help` / `-h` | Help | Prints usage to stderr |
| `upload_file` (no dash prefix) | **CLI mode** | Dispatches to `RunCLI()` |
| *(no args)* | Normal gRPC mode | Starts as gRPC plugin for robot runtime |

The CLI dispatch rule is simple: if the first argument doesn't start with `-`, it's treated as a command name. This is unambiguous because all existing flags start with `-`.

### Step-by-step execution flow

**Step 1: Command resolution**

`buildCommandMap()` scans all registered node types via `GetNodeTypes()`. For each node type that has an embedded `runtime.Tool` field, it reads the `tool` struct tag to extract the command name and description:

```go
type UploadFile struct {
    runtime.Node `spec:"id=Robomotion.GoogleDrive.Upload,name=Upload File,..."`
    runtime.Tool `tool:"name=upload_file,description=Upload a file to Google Drive"`
    // ...
}
```

This produces a map: `"upload_file" → {nodeType: UploadFile, nodeID: "Robomotion.GoogleDrive.Upload", ...}`

Only nodes with `runtime.Tool` are exposed as CLI commands. Internal nodes (Connect, Disconnect, flow control) are not included.

**Step 2: Flag parsing**

`parseFlags()` accepts three formats:

```bash
--file-path=/tmp/x.pdf     # --key=value (preferred)
--file-path /tmp/x.pdf     # --key value
--recursive                 # --flag (boolean, set to "true")
```

Two flags are treated as global and extracted before command processing:
- `--vault-id=ID` — Robomotion vault ID for credential lookup
- `--item-id=ID` — Robomotion vault item ID for credential lookup

**Step 3: Runtime initialization**

`CLIRuntimeHelper` is created and assigned to the global `client` variable (defined in `grpc.go:24`). This is the same global that `Node.Init(e RuntimeHelper)` sets during gRPC mode. By setting it here, all existing code that calls `client.GetVariable()`, `client.GetVaultItem()`, etc. works without modification.

```go
cliHelper := NewCLIRuntimeHelper()
client = cliHelper  // now InVariable.Get(), Credential.Get(), etc. all work
```

If `--vault-id` and `--item-id` are provided, the credentials are fetched from the Robomotion vault API via `CLIVaultClient` and injected into the helper before any node code runs.

**Step 4: Node instantiation**

The node is created via reflection (`reflect.New(nodeType)`) and initialized with a synthetic JSON config:

```json
{
    "guid": "cli-node",
    "name": "upload_file",
    "optToken": {
        "scope": "Custom",
        "name": {"vaultId": "v1", "itemId": "i1"}
    }
}
```

The `json.Unmarshal(config, &handler)` call populates the node struct's fields — including any `Credential` fields that need vault IDs. This is the same deserialization path that `NodeFactory.OnCreate()` uses in gRPC mode.

**Step 5: Message context construction**

`buildMessageContext()` converts CLI flags into a `message.Context` that the node reads via `InVariable.Get(ctx)`:

1. Build a mapping from kebab-case flag names to spec tag names:
   - Scan all `InVariable`/`OptVariable` fields on the node type
   - Read the `spec` tag's `name` value (e.g., `name=filePath`)
   - Convert to kebab-case: `filePath` → `file-path`
   - Store mapping: `"file-path" → "filePath"`

2. For each CLI flag, look up the spec name and set it in the context:
   ```
   --file-path=/tmp/x.pdf → ctx.Set("filePath", "/tmp/x.pdf")
   --folder-id=abc        → ctx.Set("folderId", "abc")
   ```

3. If a flag name isn't found in the mapping (e.g., a custom parameter), fall back to camelCase conversion: `--my-param` → `myParam`.

**Step 6: Node lifecycle**

The standard three-phase lifecycle runs:

```go
handler.OnCreate()      // Initialize resources (called once)
handler.OnMessage(ctx)  // Do the actual work
handler.OnClose()       // Clean up resources
```

This is identical to the gRPC lifecycle in `GRPCServer.OnCreate/OnMessage/OnClose`, minus the protobuf serialization and compression.

**Step 7: Output collection**

`collectCLIOutput()` reads output values from the message context after `OnMessage` completes:

1. Scan node type for `OutVariable` fields
2. For each, read the `spec` tag's `name` value
3. Call `ctx.Get(name)` to retrieve the value that `OutVariable.Set(ctx, val)` stored
4. Collect into a `map[string]interface{}`
5. If no outputs found, return `{"status": "completed"}`

The result is JSON-serialized and printed to stdout.

---

## 5. Usage Reference

### Running a command

```bash
robomotion-<package> <command> [--flag=value ...]

# Examples:
robomotion-googledrive upload_file --file-path=/tmp/report.pdf --folder-id=abc123
robomotion-dropbox list_folder --path=/documents
robomotion-slack send_message --channel=general --text="Deploy complete"
```

### Listing available commands

```bash
robomotion-googledrive --list-commands
```

Output (JSON array):

```json
[
  {
    "name": "upload_file",
    "description": "Upload a file to Google Drive",
    "node_id": "Robomotion.GoogleDrive.Upload",
    "parameters": [
      {"name": "file-path", "type": "string", "required": true},
      {"name": "folder-id", "type": "string", "required": false}
    ],
    "outputs": [
      {"name": "fileId", "type": "string"}
    ]
  },
  ...
]
```

### Generating SKILL.md

```bash
robomotion-googledrive --skill-md > SKILL.md
```

### Getting help

```bash
robomotion-googledrive --help
```

---

## 6. SKILL.md Generation

The `--skill-md` flag generates a markdown file that AI agents use for skill discovery. It follows the SKILL.md convention used by skill-based agent systems.

### What gets generated

The generator (`skillmd.go`) introspects registered nodes at compile time (via reflection on the same struct tags used for Designer spec generation) and produces:

**YAML frontmatter:**

```yaml
---
name: googledrive
description: Google Drive - upload, download, list, search, delete files
---
```

**For each tool-enabled node:**

- Command name and description (from `tool` tag)
- Usage line with required and optional parameters
- Parameter table with types, required/optional status, and descriptions
- Output format example

**Authentication section** (if any node has a `Credential` field):

Documents `--vault-id` and `--item-id` flags and the `ROBOMOTION_CREDENTIALS` env var alternative.

### Progressive disclosure

The SKILL.md design supports progressive disclosure — the pattern where an LLM agent sees minimal information initially and loads details only when needed:

1. **Catalog level**: Agent sees skill name + one-line description (from YAML frontmatter)
2. **Skill level**: Agent loads full SKILL.md body only when it decides to use this skill
3. **Command level**: Agent reads only the specific command section it needs

This prevents context bloat — instead of loading 15 tool definitions with full JSON schemas upfront, the agent loads one markdown file for the specific skill it's using.

### Source of truth

All parameter information comes from existing struct tags that package authors already write:

| SKILL.md field | Source |
|----------------|--------|
| Command name | `tool:"name=upload_file"` on `runtime.Tool` field |
| Command description | `tool:"description=Upload a file"` on `runtime.Tool` field |
| Parameter name | `spec:"name=filePath"` on `InVariable`/`OptVariable` field → kebab-case |
| Parameter type | `spec:"type=string"` on variable field |
| Required/optional | `InVariable` = required, `OptVariable` = optional |
| Parameter description | `spec:"title=File Path"` or `spec:"description=..."` on variable field |
| Output fields | `spec:"name=fileId"` on `OutVariable` field |
| Authentication needed | Presence of `runtime.Credential` field on any node |

No additional metadata files or configuration needed. Build the package with `runtime.Tool` on your nodes, run `--skill-md`, and the documentation writes itself.

---

## 7. Credential Handling

Robomotion packages typically require API keys, OAuth tokens, or login credentials. In gRPC mode, these flow through the robot's vault system. CLI mode provides three credential mechanisms, in priority order:

### 1. Vault flags (production)

```bash
robomotion-googledrive upload_file \
    --file-path=/tmp/x.pdf \
    --vault-id=abc123 \
    --item-id=def456
```

When `--vault-id` and `--item-id` are provided:

1. `CLIVaultClient` reads the saved auth token from `~/.robomotion/auth.json` (created by `robomotion login`)
2. Calls the Robomotion API: `GET /v1/vaults.items.get?vault_id=abc123&item_id=def456`
3. Decrypts the vault item using the same AES-256-CBC + RSA-OAEP key derivation chain as the robot runtime
4. Injects the decrypted credential map into `CLIRuntimeHelper`
5. When node code calls `Credential.Get(ctx)` → `client.GetVaultItem()` → returns the pre-fetched credentials

The decryption chain mirrors `robomotion-deskbot/runtime/vault.go` exactly:

```
Master key (PBKDF2 from password)
  → Decrypt private key (AES-CBC with master key + keySet.IV)
    → Decrypt vault key (RSA-OAEP with private key)
    → Decrypt secret key (RSA-OAEP with private key)
      → XOR(vault key, secret key) = AES key for items
        → Decrypt item data (AES-CBC with item's IV)
```

Additionally, the vault flags are injected into the node's `Credential` field config via `injectCredentialConfig()`. This sets `vaultId` and `itemId` on every `Credential` field in the node struct, so `Credential.Get(ctx)` resolves them through the standard path.

### 2. Environment variable (development/testing)

```bash
export ROBOMOTION_CREDENTIALS='{"value": "sk-test-abc123"}'
robomotion-googledrive upload_file --file-path=/tmp/x.pdf
```

The `ROBOMOTION_CREDENTIALS` env var accepts a JSON object matching the credential type the node expects:

| Credential type | JSON format |
|----------------|-------------|
| API Key/Token (category 4) | `{"value": "your-api-key"}` |
| Login (category 1) | `{"username": "user", "password": "pass"}` |
| Database (category 5) | `{"type": "postgres", "server": "localhost", "port": 5432, "database": "mydb", "username": "user", "password": "pass"}` |

### 3. No credentials

Some commands don't require authentication (e.g., format conversion, local file operations). These work without any credential flags.

### Per-call isolation

Each CLI invocation is a separate OS process with its own credential context. This means parallel calls with different credentials are naturally isolated:

```bash
# Three parallel uploads with different customer credentials — no conflicts
robomotion-googledrive upload_file --vault-id=v1 --item-id=cust-x --file-path=A &
robomotion-googledrive upload_file --vault-id=v1 --item-id=cust-y --file-path=B &
robomotion-googledrive upload_file --vault-id=v2 --item-id=cust-z --file-path=C &
```

No Connect/Disconnect nodes, no client_id routing, no session state. Each process starts clean and exits after one operation.

---

## 8. Flag-to-Variable Mapping

CLI flags use **kebab-case** (`--file-path`), while node spec tags use **camelCase** (`filePath`). The mapping is automatic and deterministic.

### Mapping rules

1. **Primary**: Scan all `InVariable`/`OptVariable` fields on the node type. For each, read the `spec` tag `name` value and convert to kebab-case:

   | Spec name | CLI flag |
   |-----------|----------|
   | `filePath` | `--file-path` |
   | `folderId` | `--folder-id` |
   | `searchQuery` | `--search-query` |
   | `maxResults` | `--max-results` |

2. **Fallback**: If a CLI flag doesn't match any known spec name, it's converted from kebab-case to camelCase and set in the context directly: `--my-custom-param` → `myCustomParam`.

### Type handling

All CLI flag values arrive as strings. The existing `InVariable.Get()` handles type conversion automatically — it already converts string values to the target type (int, float, bool, etc.) because the same conversion is needed when values come from the message context in gRPC mode.

For example, if a node has `InVariable[int]` with spec name `maxResults`, passing `--max-results=50` sets `"50"` (string) in the context. When `InVariable[int].Get(ctx)` is called, it parses `"50"` into `int64(50)` via the existing `getInt()` method.

Boolean flags can be passed as:
```bash
--recursive=true
--recursive           # shorthand, sets to "true"
--recursive=false
```

---

## 9. Output Format

CLI mode always outputs JSON to stdout. This makes it machine-readable for AI agents while remaining human-inspectable.

### Success output

After `OnMessage` completes successfully, output variable values are collected from the message context:

```json
{"fileId": "abc123", "fileName": "report.pdf", "fileSize": 1048576}
```

The keys match the `spec:"name=..."` values on `OutVariable` fields. Values are whatever the node set via `OutVariable.Set(ctx, value)`.

If no output variables were set (e.g., a delete operation), a status marker is returned:

```json
{"status": "completed"}
```

### Error output

Errors are printed to **stderr** as JSON with a non-zero exit code:

```json
{"error": "file not found: /tmp/nonexistent.pdf"}
```

Exit code is always `1` for errors. This follows the standard convention where `0` = success, non-zero = failure.

### Separation of concerns

| Stream | Content | Consumer |
|--------|---------|----------|
| stdout | JSON result | AI agent / script |
| stderr | JSON errors, debug logs | Human / logging |
| Exit code | 0 = success, 1 = error | Shell / orchestrator |

Debug messages from node code (via `runtime.EmitDebug()`) are printed to stderr in CLI mode, keeping stdout clean for the result.

---

## 10. Error Handling

Errors at each stage produce clean JSON on stderr with exit code 1:

| Stage | Example error |
|-------|---------------|
| Unknown command | `{"error": "unknown command \"uplaod_file\"; available commands: upload_file, download_file, list_folder"}` |
| Missing flag format | `{"error": "unexpected argument \"file.pdf\" (expected --flag=value)"}` |
| Vault auth failure | `{"error": "vault auth error: no saved auth found at ~/.robomotion/auth.json; run 'robomotion login' first"}` |
| Vault fetch failure | `{"error": "vault fetch error: vault item not found: vault=abc item=def"}` |
| Node init failure | `{"error": "failed to initialize node: json: cannot unmarshal ..."}` |
| OnCreate failure | `{"error": "OnCreate failed: connection timeout"}` |
| OnMessage failure | `{"error": "Google Drive API error: 403 Forbidden"}` |
| OnClose failure | `{"error": "OnClose failed: failed to close connection"}` |

The error message is always the Go error string from the failing operation. Nodes that return well-formatted errors (e.g., `runtime.NewError("ErrNotFound", "File not found")`) produce correspondingly clear CLI errors.

---

## 11. Concurrency and Isolation

### Process-level isolation

Every CLI invocation spawns a fresh OS process. There is no shared memory, no global state carried between calls, and no connection pooling. This is a deliberate design choice:

- **No race conditions**: Two parallel calls to the same package binary cannot interfere with each other.
- **No credential leaks**: Credentials are loaded, used, and discarded when the process exits.
- **No stale connections**: Each call creates fresh API clients and closes them on exit.
- **Simple failure model**: If a call crashes, it only affects that one invocation.

### Comparison with gRPC mode

In gRPC mode, the package binary is a long-running process that handles multiple nodes simultaneously. It uses `client_id` maps, mutexes, and shared connection pools. This is necessary for flow-based execution where multiple nodes share a database connection or OAuth session.

In CLI mode, none of this complexity exists. The overhead of creating a fresh process per call (typically 10-50ms) is negligible compared to LLM inference time (typically 1-5 seconds) and network API calls (typically 100ms-2s).

### When process-per-call is not enough

For operations that require persistent connections (databases, WebSocket subscriptions, browser sessions), a future **session mode** is planned:

```bash
robomotion-database connect --host=localhost --session
# → {"session_id": "abc123"}

robomotion-database query --sql="SELECT 1" --session-id=abc123
robomotion-database disconnect --session-id=abc123
```

This is not yet implemented. Most AI agent use cases are stateless request-response patterns where the current process-per-call model works well.

---

## 12. Which Nodes Are Exposed

Only nodes with an embedded `runtime.Tool` field are exposed as CLI commands. This is the same marker used for AI tool support in gRPC mode.

### Included

```go
type UploadFile struct {
    runtime.Node `spec:"id=Robomotion.GoogleDrive.Upload,name=Upload File,..."`
    runtime.Tool `tool:"name=upload_file,description=Upload a file to Google Drive"`
    // ...
}
```

This node becomes CLI command `upload_file`.

### Excluded

```go
type Connect struct {
    runtime.Node `spec:"id=Robomotion.GoogleDrive.Connect,name=Connect,..."`
    // No runtime.Tool field
    // ...
}
```

Internal/infrastructure nodes (Connect, Disconnect, flow control) typically don't have `runtime.Tool` and are not exposed. This is correct behavior — in CLI mode, connections are handled per-call via vault flags, so Connect/Disconnect nodes are unnecessary.

### Adding CLI support to an existing node

To make an existing node available via CLI, add the `runtime.Tool` field:

```go
type MyNode struct {
    runtime.Node `spec:"id=MyPackage.MyNode,name=My Node,..."`
    runtime.Tool `tool:"name=my_command,description=Does something useful"`  // add this line
    // ... existing fields unchanged
}
```

Rebuild the package. No other changes needed — the node's `OnCreate`/`OnMessage`/`OnClose` code works as-is.

---

## 13. Compatibility with gRPC Mode

CLI mode is a pure addition. It does not change any existing behavior:

- **No modified files in node packages**: All changes are in `robomotion-go/runtime/`. Package authors don't need to update their code.
- **gRPC mode unchanged**: When the package binary is launched without a command argument (or with `-a`/`-s`), it enters gRPC plugin mode exactly as before. The CLI dispatch only activates when `os.Args[1]` doesn't start with `-`.
- **Spec generation unchanged**: The `-s` flag still generates the same JSON spec for Designer.
- **Tool interceptor unchanged**: The existing `ToolInterceptor` that handles tool requests in gRPC mode is not involved in CLI mode. CLI mode runs the original handler directly (without the interceptor wrapper) since there's no tool request/response protocol to manage — the output is collected from the context directly.

### What the `CLIRuntimeHelper` does and doesn't support

| RuntimeHelper method | CLI behavior | Why |
|---------------------|-------------|-----|
| `GetVariable` / `SetVariable` | No-op (returns nil) | Variables in CLI mode are always Message scope, resolved from context directly |
| `GetVaultItem` | Returns pre-fetched credentials | Fetched from vault API or env var before node runs |
| `SetVaultItem` | Returns error | Writing vault items from CLI is not supported |
| `Debug` | Prints to stderr | Useful for debugging |
| `EmitError` | Prints to stderr | Error visibility |
| `EmitOutput` / `EmitInput` / `EmitFlowEvent` | No-op | No flow routing in CLI mode |
| `AppRequest` / `AppPublish` | Returns error | App communication requires robot runtime |
| `AppDownload` / `AppUpload` | Returns error | File transfer requires robot runtime |
| `GatewayRequest` / `ProxyRequest` | Returns error | Network proxying requires robot runtime |
| `GetRobotInfo` | Returns `{"id": "cli", "flow_id": "cli"}` | Minimal stub |
| `IsRunning` | Returns true | Always "running" during CLI execution |
| `GetPortConnections` | Returns nil | No port connections in CLI mode |
| `GetInstanceAccess` | Returns error | Platform instance access requires robot runtime |

Nodes that only use `InVariable.Get()`, `OutVariable.Set()`, `Credential.Get()`, and direct HTTP/SDK calls (which is the vast majority of nodes) work without modification in CLI mode. Nodes that rely on `AppRequest`, `EmitOutput` to other nodes, or `GetPortConnections` for flow routing will need the full robot runtime.

---

## 14. Files Reference

All files are in `robomotion-go/runtime/`:

| File | Lines | Purpose |
|------|-------|---------|
| `cli.go` | ~430 | CLI execution engine: arg parsing, node dispatch, flag mapping, output collection, `--list-commands`, `--help` |
| `cli_runtime.go` | ~120 | `CLIRuntimeHelper` — implements all 21 `RuntimeHelper` interface methods for CLI mode |
| `cli_vault.go` | ~400 | `CLIVaultClient` — direct vault API access with full AES-CBC + RSA-OAEP decryption chain, master key derivation |
| `skillmd.go` | ~210 | SKILL.md generator — introspects node types, generates YAML frontmatter + markdown command docs |
| `registry.go` | ~10 lines added | CLI dispatch in `Start()` switch: routes non-flag arguments to `RunCLI()` |

### Key functions

**cli.go:**
- `RunCLI(args []string)` — main entry point for CLI mode
- `buildCommandMap() map[string]commandEntry` — scans node types for Tool field, builds command lookup
- `parseFlags(args []string) (map[string]string, error)` — parses `--key=value` arguments
- `buildMessageContext(nodeType, flags) map[string]interface{}` — converts flags to spec-named context values
- `buildFlagNameMap(nodeType) map[string]string` — maps kebab-case flags to spec tag names
- `injectCredentialConfig(nodeType, config, vaultID, itemID)` — wires vault IDs into Credential fields
- `collectCLIOutput(nodeType, handler, ctx) map[string]interface{}` — reads OutVariable values after execution
- `listCommands()` — prints JSON command catalog
- `cliError(format, args...)` — prints JSON error to stderr and exits

**cli_runtime.go:**
- `NewCLIRuntimeHelper() *CLIRuntimeHelper` — creates helper, reads `ROBOMOTION_CREDENTIALS` env if set
- `SetCredentials(creds map[string]interface{})` — injects fetched vault credentials

**cli_vault.go:**
- `NewCLIVaultClient() (*CLIVaultClient, error)` — creates client from `~/.robomotion/auth.json`
- `FetchVaultItem(vaultID, itemID string) (map[string]interface{}, error)` — fetches and decrypts vault item
- `SetMasterKeyCLI(identity, password, salt string) []byte` — PBKDF2 key derivation

**skillmd.go:**
- `generateSkillMD(name, version string, config gjson.Result)` — generates and prints SKILL.md to stdout

---

## 15. Limitations

### Not yet supported

- **Session/stateful mode**: Operations requiring persistent connections (database queries, browser automation) need the full robot runtime or the planned `--session` flag (future work).
- **Multi-output ports**: Nodes that emit to different output ports via `EmitOutput(guid, data, port)` — CLI mode captures the return context, not port-based emissions.
- **App communication**: `AppRequest`, `AppPublish`, `AppDownload`, `AppUpload` require the robot runtime's app infrastructure.
- **Flow events**: `EmitFlowEvent` has no meaning outside a flow context.
- **LMO (Large Message Objects)**: The content-addressed blob store is not initialized in CLI mode. Nodes transferring very large payloads (>1MB) through LMO will need the robot runtime.

### By design

- **No interactive input**: CLI mode is designed for programmatic use by AI agents and scripts. There is no stdin prompt, no interactive OAuth flow (use `robomotion <pkg> auth` for that).
- **No streaming output**: Each command produces a single JSON result on completion. Long-running operations that produce incremental results should use the gRPC tool mode instead.
- **String-only input**: All CLI flag values are strings. Complex nested objects can't be passed directly (use JSON strings if needed, e.g., `--metadata='{"key": "value"}'`).
