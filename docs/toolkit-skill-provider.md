# Toolkit & Skill Providers

*Last updated: May 15, 2026*

Two opt-in SDK capabilities that let any Robomotion package expose AI-agent functionality without the agent package (Hermes, ADK, …) needing per-package knowledge. **ToolkitProvider** bundles multiple tools under a single node; **SkillProvider** ships markdown guidance that the agent appends to its system prompt. They are independent — a node can implement either, both, or neither.

This document covers the Go SDK. The Python SDK has the same shape — see `robomotion-python/docs/toolkit-skill-provider.md`.

---

## Table of Contents

1. [Why these exist](#1-why-these-exist)
2. [The mental model](#2-the-mental-model)
3. [ToolkitProvider](#3-toolkitprovider)
4. [SkillProvider](#4-skillprovider)
5. [Wire protocol — how tool calls flow](#5-wire-protocol--how-tool-calls-flow)
6. [The OptVariable+enum multi-select pattern](#6-the-optvariableenum-multi-select-pattern)
7. [Spec emission reference](#7-spec-emission-reference)
8. [End-to-end example: Agent Teams Toolkit](#8-end-to-end-example-agent-teams-toolkit)
9. [When to use single Tool vs Toolkit](#9-when-to-use-single-tool-vs-toolkit)
10. [Testing](#10-testing)
11. [Files reference](#11-files-reference)

---

## 1. Why these exist

Before these capabilities, exposing a *bundle* of related tools to an AI agent required either:

- Wiring **N individual Tool In nodes** on the canvas (a `Receive → Function → Hermes (8 × Tool In)` mess), or
- Building a **per-package branch inside every agent package** (`if node_type == 'Robomotion.AgentTeams.Toolkit': ...`) — leaks domain knowledge into Hermes/ADK and breaks the moment a new agent runtime ships.

Behavioral guidance had a similar problem: workspace-level Skills (in the Designer's Skills tab) had to be toggled separately, and the rules describing how a package's tools should be used lived in a different file than the tools themselves.

ToolkitProvider and SkillProvider solve both at the SDK level. Any agent package that reads pspec via `Runtime.get_port_connections` can consume them generically — Hermes, ADK, and any future runtime work the same way without changes.

---

## 2. The mental model

| Question | Interface | pspec key | Agent-side consumer |
|---|---|---|---|
| **What** can the agent do? | `ToolkitProvider` | `tools[]` | tool registry — one callable per ToolDef |
| **How** should the agent behave? | `SkillProvider` | `skill` | system message — appended to the prompt |
| (Legacy: one tool only) | `runtime.Tool` marker + tag | `tool` | tool registry — one callable per node |

Two independent walks of the same connected-nodes list. One harvests callables, the other harvests markdown. The agent doesn't need to know which interface a node implements.

---

## 3. ToolkitProvider

### Marker

```go
type Toolkit struct{}
```

Embed it in your node struct. The spec generator detects the embedded type by name (`Toolkit`) and switches into multi-tool emission.

### Interface

```go
type ToolDef struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Schema      map[string]interface{} `json:"schema"`
}

type ToolkitProvider interface {
    Tools() []ToolDef
}
```

`Tools()` is called once at package build time during `go run main.go -s` (spec generation). The list is **frozen at build** — flow-author scoping at runtime happens through an `OptVariable` (see §6), not by re-evaluating `Tools()` per turn.

`Schema` is JSON Schema. Build it inline, load it from `//go:embed`, generate it from struct types — any approach that yields a `map[string]interface{}` works.

### Helpers

In your `OnMessage`, the agent invocation arrives with a tool-name discriminator and parameters dict:

```go
func (n *MyToolkit) OnMessage(ctx message.Context) error {
    if !runtime.IsToolRequest(ctx) {
        return nil
    }
    name   := runtime.ToolName(ctx)
    params := runtime.ToolParameters(ctx)

    switch name {
    case "send_message":
        return n.sendMessage(ctx, params)
    case "comment_on_ticket":
        return n.commentOnTicket(ctx, params)
    // ...
    }
    return runtime.ToolResponse(ctx, "error", nil, "unknown tool: "+name)
}
```

Each handler returns via `runtime.ToolResponse(ctx, status, data, errMsg)`. The SDK's `ToolInterceptor` is responsible for routing the response back to the calling agent over Robomotion's emit_input bus — see §5.

### Minimal example

```go
type SearchToolkit struct {
    runtime.Node    `spec:"id=Acme.Search.Toolkit,name=Search Toolkit,icon=mdiMagnify,color=#3b82f6"`
    runtime.Toolkit `toolkit:""`
}

func (n *SearchToolkit) OnCreate() error                   { return nil }
func (n *SearchToolkit) OnClose() error                    { return nil }
func (n *SearchToolkit) OnMessage(ctx message.Context) error {
    if !runtime.IsToolRequest(ctx) {
        return nil
    }
    params := runtime.ToolParameters(ctx)
    switch runtime.ToolName(ctx) {
    case "web_search":
        result, err := webSearch(params["query"].(string))
        if err != nil {
            return runtime.ToolResponse(ctx, "error", nil, err.Error())
        }
        return runtime.ToolResponse(ctx, "success", map[string]interface{}{"results": result}, "")
    case "image_search":
        // ...
    }
    return runtime.ToolResponse(ctx, "error", nil, "unknown tool")
}

func (n *SearchToolkit) Tools() []runtime.ToolDef {
    return []runtime.ToolDef{
        {
            Name:        "web_search",
            Description: "Search the web. Returns the top 10 hits as {title, url, snippet}.",
            Schema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "query": map[string]interface{}{
                        "type":        "string",
                        "description": "Free-text search query.",
                    },
                },
                "required": []string{"query"},
            },
        },
        {
            Name:        "image_search",
            Description: "Search for images. Returns top 20 image URLs with alt text.",
            Schema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "query": map[string]interface{}{"type": "string"},
                },
                "required": []string{"query"},
            },
        },
    }
}
```

Register the toolkit in your `main.go` just like any other node:

```go
runtime.RegisterNodes(&SearchToolkit{})
```

The LLM now sees two callable tools (`web_search`, `image_search`) — but only one node on the canvas.

---

## 4. SkillProvider

### Interface

```go
type SkillProvider interface {
    Skill() string
}
```

Returns markdown. The spec generator emits it as the node's `skill` field. The consuming agent appends it to the system prompt when the node is wired to its tools port.

Conceptually the in-tree equivalent of MCP's `prompts/get` — the behavioral contract travels with the tools instead of living in a separate skill store.

### Independence from Toolkit

SkillProvider is **not** part of ToolkitProvider. A node can:

- Implement both (the common case — a toolkit + its AGENT.md)
- Implement only `Toolkit` (raw tools, no extra guidance — the function descriptions are enough)
- Implement only `Skill` (no callable tools, just system-prompt extension)
- Implement neither (a regular flow node — unchanged behavior)

### Three patterns

#### Pattern A — Skill alongside a Toolkit

The canonical pairing. The toolkit's AGENT.md explains when to call each tool, the etiquette, the gotchas:

```go
//go:embed AGENT.md
var agentMD string

type Toolkit struct {
    runtime.Node    `spec:"id=Robomotion.AgentTeams.Toolkit,..."`
    runtime.Toolkit `toolkit:""`
}

func (n *Toolkit) Tools() []runtime.ToolDef { /* 8 tools */ }
func (n *Toolkit) Skill() string            { return agentMD }
```

`AGENT.md` excerpt:

```markdown
# Agent Teams operating contract

- For chat (mention/DM): reply with `send_message`. Be useful in one
  message when you can.
- For tickets: comment with `comment_on_ticket`. Only call
  `update_ticket(status="closed")` when work is complete AND verified.
- Use `request_approval(question, options?)` for any irreversible
  action — status transition to closed, external email, large data
  deletion, budget spend.
- Do not @-mention the original sender just to acknowledge. Silence
  ends a conversation cleanly.
```

#### Pattern B — Skill alongside a single Tool

A `runtime.Tool`-tagged node ships its own usage convention:

```go
type CreateLinearIssue struct {
    runtime.Node `spec:"id=Linear.CreateIssue,..."`
    runtime.Tool `tool:"name=linear_create_issue,description=Create a Linear issue"`
}

func (n *CreateLinearIssue) Skill() string {
    return `When creating Linear issues:
- Always set team_id explicitly. Never default to "engineering".
- Label "bug" is for confirmed defects only; user reports use "user-feedback".
- Priority: P2 for production issues, P3 for everything else.
- Title must start with a verb: "Fix...", "Add...", "Investigate...".`
}
```

The LLM gets the tool **and** the team's specific conventions. The convention used to live in someone's head or a separate doc; now it ships with the node.

#### Pattern C — Skill with no tools at all

A "context node" wired to the tools port purely to inject guidance:

```go
type BrandVoiceContext struct {
    runtime.Node `spec:"id=Acme.BrandVoice,name=Brand Voice,icon=mdiBullhorn,color=#10b981"`
}

//go:embed brand-voice.md
var brandVoiceMD string

func (n *BrandVoiceContext) Skill() string { return brandVoiceMD }
func (n *BrandVoiceContext) OnCreate() error                   { return nil }
func (n *BrandVoiceContext) OnClose() error                    { return nil }
func (n *BrandVoiceContext) OnMessage(_ message.Context) error { return nil }
```

`brand-voice.md`:

```markdown
# Acme Brand Voice

When writing customer-facing copy:
- Use second person ("you can…") not first ("we can…")
- Never exclamation marks. Period.
- "Acme" is always capitalized. Never "ACME" or "acme".
- Don't use the words "easy" or "simple" — they're patronizing.
```

Wire this to a HermesAgent. The LLM now writes copy in Acme's voice on every turn even though no callable tool was added. Pure system-message extension by canvas wiring.

---

## 5. Wire protocol — how tool calls flow

Agents call toolkit tools over Robomotion's normal `emit_input` message bus. **No HTTP, no MCP subprocess, no new transport** — the same gRPC + NATS path every other inter-node message uses.

```
1. Agent's LLM emits a tool call:    web_search(query="cats")
2. Agent runtime serializes:
       message.Context with
         __message_type__   = "tool_request"
         __tool_name__      = "web_search"
         __parameters__     = {"query": "cats"}
         __tool_caller_id__ = "<uuid>"
         __agent_node_id__  = "<agent guid>"
3. Event.emit_input(toolkitNodeGuid, ctx)
4. Toolkit's OnMessage fires; ToolInterceptor recognizes IsToolRequest(ctx).
5. Handler runs; calls runtime.ToolResponse(ctx, "success", data, "")
6. ToolResponse emits a __message_type__="tool_response" back to
   __agent_node_id__ via Event.emit_input.
7. Agent's on_message catches the tool_response at the top, looks up
   the caller_id, unblocks the waiting handler with the result.
```

This is the same protocol single-Tool nodes have always spoken (see `runtime/tool_response.go`). The only difference for a Toolkit is the `__tool_name__` discriminator — which is what lets one node guid host N tools.

### Reading helpers

```go
runtime.IsToolRequest(ctx)         // bool — am I being called as a tool?
runtime.ToolName(ctx)              // string — which one?
runtime.ToolParameters(ctx)        // map[string]interface{} — the args
runtime.ToolResponse(ctx, status, data, errMsg)   // reply
```

### Auto-response from ToolInterceptor

If your `OnMessage` returns without calling `ToolResponse` explicitly, the `ToolInterceptor` collects every OutVariable's value and auto-responds with a synthesized success. This is fine for trivial passthrough cases. **For toolkits, always call `ToolResponse` explicitly** — it gives you control over the success/error split and the response payload shape.

---

## 6. The OptVariable+enum multi-select pattern

A toolkit usually wants flow authors to choose **which** tools to expose to the agent (security + LLM-attention budget). The SDK supports this with a Catch-style checkbox grid:

```go
type Toolkit struct {
    runtime.Node    `spec:"id=...,..."`
    runtime.Toolkit `toolkit:""`

    OptEnabled runtime.OptVariable[[]string] `spec:"
        title=Enabled Tools,
        type=array,
        scope=Custom,
        name=,
        customScope,
        enum=web_search|image_search|news_search,
        enumNames=Web Search|Image Search|News Search,
        description=Pick which tools this toolkit exposes. Empty = all."`
}
```

What this produces:

- **Spec** — `type=array`, `multiple=true`, `items.enum=[...]`, `items.enumNames=[...]`, `formData=[]` default
- **Designer UISchema** — `ui:field=multiSelectCheckbox` (the widget added in `robomotion-new-designer`)
- **Runtime** — the user-selected names land as a `[]string` on `OptEnabled`. Agent-side discovery reads it and registers only the selected tools.

In the agent-side discovery branch (Hermes example):

```python
enabled = read_optvar_string_list(inner, 'optEnabled')
for tdef in tools_array:
    if enabled and tdef['name'] not in enabled:
        continue
    register(tdef)
```

The convention is: **empty list = expose all tools**. Users opt out, they don't opt in. This makes a freshly-dropped toolkit useful by default.

---

## 7. Spec emission reference

What lands in the generated pspec JSON for each shape:

### Toolkit + Skill

```json
{
  "id": "Acme.Search.Toolkit",
  "name": "Search Toolkit",
  "icon": "...",
  "color": "#3b82f6",
  "tools": [
    {
      "name": "web_search",
      "description": "Search the web...",
      "schema": { "type": "object", "properties": { ... }, "required": [...] }
    },
    { "name": "image_search", ... }
  ],
  "skill": "# How to use Acme Search\n\nFor general queries...",
  "properties": [ ... ]
}
```

### Single Tool (legacy, unchanged)

```json
{
  "id": "Linear.CreateIssue",
  "tool": { "name": "linear_create_issue", "description": "Create a Linear issue" }
}
```

A node should emit **either** `tool` (singular) **or** `tools[]` (plural), never both. The spec generator does not enforce this, but consuming agents may pick whichever they see first; mixing them is a bug.

### Skill-only

```json
{
  "id": "Acme.BrandVoice",
  "skill": "# Acme Brand Voice\n\nWhen writing customer-facing copy..."
}
```

No `tool` or `tools` keys. The agent's `connect_tools` walk skips this node (no callable to register); its `collect_node_skills` walk picks it up.

---

## 8. End-to-end example: Agent Teams Toolkit

The reference implementation lives in `packages-main/src/robomotion-agent-teams/v1/agents/toolkit.go`. Highlights:

```go
//go:embed AGENT.md
var agentMD string

type Toolkit struct {
    runtime.Node    `spec:"id=Robomotion.AgentTeams.Agents.Toolkit,name=Agent Teams Toolkit,icon=agentTeamsIcon,color=#f97316"`
    runtime.Toolkit `toolkit:""`

    OptEnabled runtime.OptVariable[[]string] `spec:"...,enum=send_message|comment_on_ticket|update_ticket|get_ticket|list_channels|list_channel_members|open_dm|request_approval,..."`
    OptAutoAck runtime.OptVariable[bool]     `spec:"title=Auto-Ack Inbox,type=boolean,..."`
}

func (n *Toolkit) Tools() []runtime.ToolDef {
    return []runtime.ToolDef{
        {Name: "send_message",      ...},
        {Name: "comment_on_ticket", ...},
        {Name: "update_ticket",     ...},
        {Name: "get_ticket",        ...},
        {Name: "list_channels",     ...},
        {Name: "list_channel_members", ...},
        {Name: "open_dm",           ...},
        {Name: "request_approval",  ...},
    }
}

func (n *Toolkit) Skill() string { return agentMD }

func (n *Toolkit) OnMessage(ctx message.Context) error {
    if !runtime.IsToolRequest(ctx) { return nil }
    tool := runtime.ToolName(ctx)
    params := runtime.ToolParameters(ctx)

    if !n.toolEnabled(ctx, tool) {
        return runtime.ToolResponse(ctx, "error", nil, "tool '"+tool+"' is not enabled")
    }

    switch tool {
    case "send_message":      // POST /v1/teams.agents.messages.post
    case "comment_on_ticket": // POST /v1/teams.agents.tickets.comment
    // ... 6 more
    }
}
```

What this delivers to the flow author:

- **One node** wired to Hermes/ADK's tools port instead of 8 individual nodes
- **Checkbox grid** in the Designer for narrowing the exposed surface
- **AGENT.md guidance** automatically merged into the agent's system prompt
- **Inbox auto-ack** on first reply tool (`OptAutoAck`)
- **Cross-agent** — the same toolkit works against Hermes and ADK with no per-agent code

---

## 9. When to use single Tool vs Toolkit

Use **single `runtime.Tool`** when:

- The node has one obvious action and the flow author wants it visibly on the canvas (e.g., `Linear.CreateIssue`, `Stripe.RefundPayment`)
- Auditability matters — each tool gets its own node, each call leaves its own trace pulse on the canvas
- Different tools have very different parameter shapes that wouldn't benefit from sharing context

Use **`runtime.Toolkit`** when:

- 3+ related actions share context (auth, env, conventions) and the flow author shouldn't have to wire them individually
- The agent should pick from a menu based on the situation (chat vs ticket, etc.) — bundling lets the LLM see them all together
- You want a checkbox-grid surface filter

Both shapes use the same wire protocol — you can migrate one to the other later without breaking consumers.

---

## 10. Testing

The SDK ships tests at `runtime/tool_test.go` and `runtime/spec_test.go`:

```bash
cd robomotion-go
go test ./runtime/ -run 'TestTool|TestSkill|TestSpec' -v
```

Patterns for testing your own toolkit:

```go
func TestMyToolkit_Tools(t *testing.T) {
    tk := &MyToolkit{}
    tools := tk.Tools()
    if len(tools) != 3 {
        t.Fatalf("expected 3 tools, got %d", len(tools))
    }
    for _, td := range tools {
        if td.Name == "" || td.Description == "" || td.Schema == nil {
            t.Errorf("incomplete ToolDef: %+v", td)
        }
    }
}

func TestMyToolkit_Dispatch(t *testing.T) {
    ctx := message.NewContext([]byte(`{
        "__message_type__": "tool_request",
        "__tool_name__":    "web_search",
        "__parameters__":   {"query": "cats"}
    }`))
    if !runtime.IsToolRequest(ctx) {
        t.Fatal("ctx should be a tool request")
    }
    if runtime.ToolName(ctx) != "web_search" {
        t.Fatal("wrong tool name")
    }
    if runtime.ToolParameters(ctx)["query"] != "cats" {
        t.Fatal("wrong params")
    }
}
```

For full integration testing including spec emission, see the patterns in `runtime/spec_test.go` — it captures `generateSpecFile`'s stdout, parses the JSON, and makes structured assertions about pspec shape.

---

## 11. Files reference

| File | What |
|---|---|
| `runtime/tool.go` | `Tool`, `Toolkit` markers; `ToolDef`, `ToolkitProvider`, `SkillProvider` |
| `runtime/tool_response.go` | `IsToolRequest`, `ToolName`, `ToolParameters`, `ToolResponse` |
| `runtime/tool_interceptor.go` | Auto-wraps `OnMessage` to recognize tool requests |
| `runtime/spec.go` | Spec generator — emits `tool`, `tools[]`, `skill` plus the OptVariable+enum multi-select shape |
| `runtime/tool_test.go` | Unit tests for helpers + interface compliance |
| `runtime/spec_test.go` | Spec emission tests with stdout capture |

---

## See also

- **Python equivalent** — `robomotion-python/docs/toolkit-skill-provider.md`
- **First production user** — `packages-main/src/robomotion-agent-teams/v1/agents/toolkit.go`
- **Agent-side discovery** — `packages-main/src/hermes-agent/nodes/simulation/tool_simulation.py` (the `'tools'`/`'tool'`/`skill` branches in `connect_tools` + `collect_node_skills`) and the matching path in `packages-main/src/adk-agent/`
- **Designer widget** — `robomotion-new-designer/src/components/ui/jsonSchemaForm/components/multiSelectCheckbox/`
