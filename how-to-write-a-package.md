# Robomotion Go Package Development Guide

*Last updated: May 29, 2026*

Welcome to the **Robomotion package development guide** for Go developers. This guide covers building custom Robomotion packages using the [`robomotion-go`](https://github.com/robomotionio/robomotion-go) SDK.

---

## 1. Prerequisites

| Tool | Minimum version | Purpose |
|------|-----------------|---------|
| Go   | `1.24`          | Compiles your code & provides `go` tooling. The SDK module declares `go 1.24.0` in its `go.mod`; generics (used by `InVariable[T]` et al.) require at least `1.18`, but match the SDK to avoid toolchain downloads |
| `roboctl` | **latest** (run `roboctl version`) | CLI that scaffolds and builds packages |
| Git  | any             | Source-control & versioning |
| A text editor / IDE | – | Coding |

> **Install `roboctl`**  
> Since roboctl is not open source, download the binary for your platform:
> ```bash
> # Download for your platform:
> # macOS Intel: https://packages.robomotion.io/releases/roboctl/roboctl-v1.8.0-darwin-amd64.tar.gz
> # macOS Apple Silicon: https://packages.robomotion.io/releases/roboctl/roboctl-v1.8.0-darwin-arm64.tar.gz  
> # Linux: https://packages.robomotion.io/releases/roboctl/roboctl-v1.8.0-linux-amd64.tar.gz
> # Windows: https://packages.robomotion.io/releases/roboctl/roboctl-v1.8.0-windows-amd64.tar.gz
> 
> # Extract and add to PATH
> tar -xzf roboctl-v1.8.0-*.tar.gz
> sudo mv roboctl /usr/local/bin/  # or add to your PATH
> ```

---

## 2. Package Anatomy – What gets shipped

A Robomotion *package* is nothing more than **a self-contained binary plus a metadata file**. When imported into a robot the runner spawns the binary and talks to it via gRPC (the plumbing is already handled by the SDK).

```
my-package/
 ├─ go.mod              # Go module definition
 ├─ config.json         # Metadata & build scripts        ← required
 ├─ main.go             # Entry-point: register nodes
 ├─ v1/                 # Your first version of nodes
 │   ├─ foo.go          # implementation of a node
 │   └─ bar.go
 ├─ icon.png            # Package icon shown in Designer  ← optional (PNG/SVG)
 └─ dist/               # artefacts created by roboctl
```

### `config.json`
The file is conceptually similar to `package.json` in NodeJS – *it is authoritative*. Below is an abbreviated example taken from the Chat-Assistant package:

```json
{
  "name": "Chat Assistant",
  "namespace": "Robomotion.ChatAssistant",   // must be globally unique
  "version": "1.1.4",                       // SemVer is recommended
  "keywords": ["chat", "agent", "ai"],
  "categories": ["Productivity", "AI"],
  "description": "Create powerful AI chat assistants…",
  "icon": "icon.png",                       // path relative to config.json
  "language": "Go",                         // always "Go" for this guide
  "platforms": ["linux", "windows", "darwin"],
  "author": { "name": "You", "email": "you@example.com" },
  "scripts": {
    "linux":   { "build": ["go build -o dist/my-package"], "run": "my-package" },
    "windows": { "build": ["go build -o dist/my-package.exe"], "run": "my-package.exe" },
    "darwin":  { "build": ["go build -o dist/my-package"], "run": "my-package" }
  }
}
```
The `scripts.<os>.build` array can run **any** shell command – but 90 % of the time one `go build` line is enough.

---

## 3. Bootstrap a New Package (recommended way)

```bash
roboctl package create \
  --name "Weather" \
  --namespace "Acme.Weather" \
  --description "Fetches weather data" \
  --categories "Utilities" \
  --color "#3498db" \
  -o weather
cd weather
```
The wizard sets up **everything** – including `config.json`, a minimal `main.go`, the icon stub and Git ignore.

> **Manual route** – If you prefer full control, create the files yourself; just respect the anatomy from section 2.

---

## 4. Understanding the Runtime

The Robomotion runtime is exposed through the **`runtime`** package. A few key concepts:

| Concept | Type | Description |
|---------|------|-------------|
| *Node* | `runtime.Node` | Embeddable struct that carries common fields (`GUID`, `Name`, `DelayBefore`, `DelayAfter`, `ContinueOnError`) |
| *Variable* | `InVariable[T]`, `OutVariable[T]`, `OptVariable[T]` | Strongly-typed message variables |
| *Lifecycle* | `OnCreate`, `OnMessage`, `OnClose` | Mandatory callbacks a node must implement |
| *Interactive setup* | `OnSetup` (optional, via `SetupHandler`) | An interactive setup action — QR/OAuth/device-code/test-connection — run from the UI **outside any flow run** (§6.1) |
| *AI tool* | `runtime.Tool` / `runtime.Toolkit` (optional) | Marks a node as agent-callable: one tool (`Tool`) or many (`Toolkit` + `ToolkitProvider`), optionally with `SkillProvider` guidance (§17) |
| *Runtime Helper* | interface `RuntimeHelper` | Provided to you inside `Init` – gives access to vault, app requests, file upload, setup-event relay, … |
| *Credential* | `runtime.Credential` | A bound RPA-Vault item; `.Get(ctx)` returns its decrypted fields (§7) |
| *Large Message Object (LMO)* | content-addressed blob store | Transparently swaps large message fields (≥ 4 KB) for `BlobRef` markers backed by zstd blobs (§20) |
| *OAuth2 helper* | `runtime.OpenOAuthDialog` | Browser-based OAuth2 authorization-code flow with a fixed local callback (§19) |

You **never** instantiate a node yourself – the runner does that through reflection.

---

## 5. Declaring a Node (the **`spec`** tag)

Every Go struct that embeds `runtime.Node` and is registered in `main.go` becomes a *node* in Designer.
Metadata is provided via a **struct-tag** called `spec`.

> **Important**: The node `id` must start with the `namespace` from your `config.json`. For example, if your `config.json` has `"namespace": "Acme"`, all your node IDs must start with `Acme.` (e.g., `Acme.Hello`, `Acme.World`).

```go
// v1/hello.go
// Assumes config.json has: "namespace": "Acme"
package v1

import (
    "github.com/robomotionio/robomotion-go/runtime"
    "github.com/robomotionio/robomotion-go/message"
)

type Hello struct {
    // Node declaration - ID starts with "Acme." to match config.json namespace
    runtime.Node `spec:"id=Acme.Hello,name=Hello,icon=mdiHand,color=#3498db,inputs=1,outputs=1"`

    // === INPUT ==========================================================
    InName runtime.InVariable[string] `spec:"title=Name,type=string,scope=Message,messageScope,jsScope,customScope"`

    // === OUTPUT ==========================================================
    OutGreeting runtime.OutVariable[string] `spec:"title=Greeting,type=string,scope=Message,messageScope"`
    
    // === OPTIONS =========================================================
    OptGreeting runtime.OptVariable[string] `spec:"title=Greeting,value=Hello,option,description=Default greeting"`
}

func (n *Hello) OnCreate() error                 { return nil }
func (n *Hello) OnMessage(ctx message.Context) error {
    name, _ := n.InName.Get(ctx)
    greeting, _ := n.OptGreeting.Get(ctx)
    result := fmt.Sprintf("%s %s", greeting, name)
    n.OutGreeting.Set(ctx, result)
    return nil
}
func (n *Hello) OnClose() error                  { return nil }
```

### 5.1 Tag reference (fields & node)

Below is a **non-exhaustive** but practically complete list of keys you can use inside a `spec:"…"` tag:

| Key | Applies to | Example | Meaning |
|-----|------------|---------|---------|
| `id` | Node | `Acme.Hello` | **Unique** identifier that the runtime looks up. **MUST start with the namespace from config.json** |
| `name` | Node | `Hello` | Display name in Designer |
| `icon` | Node | `mdiHand` | Material-Design-Icon identifier – resolved through `runtime/icons.go` |
| `color` | Node | `#3498db` | Hex color shown behind the icon. **IMPORTANT: All nodes in the same package MUST use the same color code for consistency** |
| `inputs` / `outputs` | Node | `inputs=0` | Override default 1-in 1-out configuration |
| `editor` | Node | `editor=tsx` | Custom code editor language if you have a code property |
| `inFilters` | Node | `inFilters=files` | Hide node unless the incoming link carries the specified *filter* |
| `title` | Field | `title=Greeting` | Human friendly caption |
| `type` | Field | `type=string` | Primitive type (`string`, `int`, `object`, …) |
| `value` | Field | `value=Hello` | Default value for **options** |
| `description` | Field | `description=…` | Tooltip text |
| `enum`, `enumNames` | Field | `enum=a|b|c,enumNames=A|B|C` | Enumerations (the SDK splits on `|`). On a plain field → single-select dropdown; on an `OptVariable[[]string]` → multi-select checkbox grid (§17.4) |
| `scope` | Variable | `scope=Message` | One of `Message`, `Custom`, `JS`, `CS`, `AI` |
| `messageScope`, `customScope`, `jsScope`, `csScope`, `aiScope` | Variable | `messageScope` | Flags that control which scopes the Designer offers for this field. `aiScope` lets an LLM agent supply the value when the node is wired as a tool |
| `messageOnly` | Variable | `messageOnly` | Restricts the field to Message scope only (no Custom/JS picker) |
| `option` | Field | `option` | Marks the property as a user-configurable *option* (as opposed to runtime input) |
| `arrayFields` | Field | `arrayFields=Label|Value` | For array-of-object variables – names become sub-fields |
| `format` | Field | `format=password` | JSON-schema format for Designer (dates, passwords, …) |
| `hidden` | Field | `hidden` | Field is invisible but still stored |
| `category` | Field | `category=2` | Group fields under collapsible panels |
| `tool` (own tag) | Node | `tool:"name=send_msg,description=…"` | On a `runtime.Tool` field, exposes the node as a single AI tool / CLI command (§17, §18) |
| `toolkit` (own tag) | Node | `toolkit:""` | On a `runtime.Toolkit` field, the node publishes many tools via `Tools()` (§17.2) |

### 5.2 Node ID Namespace Requirements

**CRITICAL**: All node IDs **MUST** be prefixed with the `namespace` defined in your `config.json` file.

The namespace in `config.json` is the authoritative prefix for all node IDs in your package. This ensures proper node discovery and prevents ID conflicts across packages.

**Rules:**
1. The node ID must start with the exact namespace from `config.json`
2. After the namespace, add a dot (`.`) followed by the node name
3. You can optionally add a category between the namespace and node name: `Namespace.Category.NodeName`

**Example with `config.json`:**
```json
{
  "namespace": "Robomotion.LanceDB",
  "name": "LanceDB",
  "version": "1.0.0"
}
```

**Valid node IDs for this namespace:**
```go
// Direct node name after namespace
type ConnectNode struct {
    runtime.Node `spec:"id=Robomotion.LanceDB.Connect,name=Connect,icon=mdiConnection,color=#3498db"`
}

// With category
type VectorSearchNode struct {
    runtime.Node `spec:"id=Robomotion.LanceDB.Vector.Search,name=Vector Search,icon=mdiMagnify,color=#3498db"`
}

type VectorInsertNode struct {
    runtime.Node `spec:"id=Robomotion.LanceDB.Vector.Insert,name=Vector Insert,icon=mdiPlusBox,color=#3498db"`
}

// More examples
type QueryNode struct {
    runtime.Node `spec:"id=Robomotion.LanceDB.Query,name=Query,icon=mdiDatabase,color=#3498db"`
}

type CreateTableNode struct {
    runtime.Node `spec:"id=Robomotion.LanceDB.Table.Create,name=Create Table,icon=mdiTablePlus,color=#3498db"`
}
```

**INVALID node IDs (will not work):**
```go
// Wrong - different namespace prefix
type ConnectNode struct {
    runtime.Node `spec:"id=LanceDB.Connect,..."`           // ❌ Missing "Robomotion." prefix
}

type ConnectNode struct {
    runtime.Node `spec:"id=Acme.LanceDB.Connect,..."`      // ❌ Wrong namespace prefix
}

type ConnectNode struct {
    runtime.Node `spec:"id=Connect,..."`                   // ❌ No namespace at all
}

// Correct
type ConnectNode struct {
    runtime.Node `spec:"id=Robomotion.LanceDB.Connect,..."` // ✅ Matches config.json namespace
}
```

**More namespace examples:**

| config.json namespace | Valid node IDs |
|-----------------------|----------------|
| `Acme.Weather` | `Acme.Weather.GetForecast`, `Acme.Weather.Current` |
| `MyCompany.Slack` | `MyCompany.Slack.SendMessage`, `MyCompany.Slack.Channel.List` |
| `OpenAI` | `OpenAI.ChatCompletion`, `OpenAI.Embedding.Create` |
| `AWS.S3` | `AWS.S3.Upload`, `AWS.S3.Download`, `AWS.S3.Bucket.List` |

### 5.3 Package Color Consistency Requirements

**CRITICAL**: All nodes within the same package MUST use identical color codes to maintain visual consistency in the Flow Designer.

**Implementation Rules:**
- Choose ONE hex color code for your entire package (e.g., `#3498db`)
- Apply this exact color code to ALL nodes in your package
- Never mix different colors within a single package
- The color should represent the service/platform your package integrates with

**Example Implementation:**
```go
// ALL nodes in the package use the same color
type ConnectNode struct {
    runtime.Node `spec:"id=Slack.Connect,name=Connect,icon=mdiConnection,color=#4A154B"`
}

type SendMessageNode struct {
    runtime.Node `spec:"id=Slack.SendMessage,name=Send Message,icon=mdiMessage,color=#4A154B"`
}

type GetChannelsNode struct {
    runtime.Node `spec:"id=Slack.GetChannels,name=Get Channels,icon=mdiFormatListBulleted,color=#4A154B"`
}
```

**Common Color Choices:**
- **Slack**: `#4A154B` (Slack brand purple)
- **Notion**: `#000000` (Notion brand black)
- **GitHub**: `#333333` (GitHub dark gray)
- **AWS**: `#FF9900` (AWS orange)
- **Azure**: `#0078D4` (Microsoft blue)

**Why This Matters:**
- Creates professional, cohesive visual experience
- Users can instantly identify which nodes belong to the same package
- Maintains brand consistency with the integrated service
- Reduces cognitive load when building flows

### 5.4 Node Parameter Naming Rules

**CRITICAL**: Follow these standardized naming conventions for all node parameters to ensure consistency across packages:

#### Variable Naming Requirements

1. **InVariable names always start with `In`** followed by PascalCase:
   ```go
   InPageTitle runtime.InVariable[string]
   InDatabaseID runtime.InVariable[string]
   InClientID runtime.InVariable[string]
   ```

2. **OutVariable names always start with `Out`** followed by PascalCase:
   ```go
   OutResult runtime.OutVariable[string]
   OutPageID runtime.OutVariable[string]
   OutDatabaseInfo runtime.OutVariable[map[string]interface{}]
   ```

3. **OptVariable names always start with `Opt`** followed by PascalCase:
   ```go
   OptTimeout runtime.OptVariable[int]
   OptMaxResults runtime.OptVariable[int]
   OptToken runtime.Credential
   ```

#### Message Scope Naming Convention

4. **For variables with `scope=Message`, add `name=` with camelCase version** (removing In/Out/Opt prefix):
   ```go
   // Correct examples
   InPageTitle runtime.InVariable[string] `spec:"title=Page Title,type=string,scope=Message,name=pageTitle,messageScope"`
   InDatabaseID runtime.InVariable[string] `spec:"title=Database ID,type=string,scope=Message,name=databaseId,messageScope"`
   OutResult runtime.OutVariable[string] `spec:"title=Result,type=string,scope=Message,name=result,messageScope"`
   ```

5. **All InVariable fields should include `jsScope,customScope`** for maximum flexibility:
   ```go
   InPageTitle runtime.InVariable[string] `spec:"title=Page Title,type=string,scope=Message,name=pageTitle,messageScope,jsScope,customScope"`
   ```

6. **OptVariable fields can have `customScope` and `jsScope`** for user configuration flexibility:
   ```go
   // Correct - OptVariable can have all scope types
   OptTimeout runtime.OptVariable[int] `spec:"title=Timeout (seconds),type=int,value=30,scope=Message,name=timeout,messageScope,customScope,jsScope"`
   OptMaxResults runtime.OptVariable[int] `spec:"title=Max Results,type=int,value=100,scope=Message,name=maxResults,messageScope,customScope"`
   ```

7. **OutVariable fields should ONLY have `messageScope`** - never include `customScope` or `jsScope`:
   ```go
   // Correct - OutVariable with only messageScope
   OutResult runtime.OutVariable[string] `spec:"title=Result,type=string,scope=Message,name=result,messageScope"`
   OutPageID runtime.OutVariable[string] `spec:"title=Page ID,type=string,scope=Message,name=pageId,messageScope"`
   
   // INCORRECT - OutVariable should not have customScope or jsScope
   OutResult runtime.OutVariable[string] `spec:"title=Result,customScope,jsScope"` // ❌ Wrong
   ```

#### Type Usage Rules

8. **Use Variable wrappers for inputs/outputs, raw types for enums and bool options**:
   ```go
   // CORRECT - Variable wrappers for inputs, outputs, and non-bool options
   InPageTitle runtime.InVariable[string] `spec:"..."`
   OutResult runtime.OutVariable[string] `spec:"..."`
   OptTimeout runtime.OptVariable[int] `spec:"..."`

   // CORRECT - Raw types for enums and bool options (accessed directly, no .Get())
   OptLabelFilter string `spec:"title=Label,value=INBOX,enum=INBOX|SENT,option"`
   OptIsHTML bool `spec:"title=HTML Content,value=false,option"`

   // INCORRECT - Raw types for inputs/outputs
   PageTitle string `spec:"..."` // ❌ Wrong
   Result string `spec:"..."` // ❌ Wrong

   // INCORRECT - Variable wrappers for enums or bool options
   OptIsHTML runtime.OptVariable[bool] `spec:"..."` // ❌ Wrong
   ```

9. **Use `interface{}` for JSON input/output types**:
   ```go
   // CORRECT - Use interface{} for JSON data
   InRequestBody runtime.InVariable[interface{}] `spec:"title=Request Body,type=object,scope=Message,name=requestBody,messageScope,jsScope,customScope"`
   OutDatabaseInfo runtime.OutVariable[interface{}] `spec:"title=Database Info,type=object,scope=Message,name=databaseInfo,messageScope"`
   OptMetadata runtime.OptVariable[interface{}] `spec:"title=Metadata,type=object,scope=Message,name=metadata,messageScope"`
   
   // INCORRECT - Don't use specific struct types for JSON
   InRequestBody runtime.InVariable[map[string]string] `spec:"..."` // ❌ Too restrictive
   OutDatabaseInfo runtime.OutVariable[DatabaseStruct] `spec:"..."` // ❌ Too specific
   ```

#### Custom Scope Usage Guidelines

10. **Use `scope=Custom,name=Default Value` for user-friendly input fields** where manual entry is more convenient than setting message variables:
   ```go
   // Good candidates for Custom scope - easy to type manually with default values
   InPageTitle runtime.InVariable[string] `spec:"title=Page Title,type=string,scope=Custom,name=My New Page,customScope,messageScope"`
   InDescription runtime.InVariable[string] `spec:"title=Description,type=string,scope=Custom,name=This is a sample description,customScope,messageScope"`
   InSearchQuery runtime.InVariable[string] `spec:"title=Search Query,type=string,scope=Custom,name=golang tutorial,customScope,messageScope"`
   
   // Poor candidates for Custom scope - usually programmatically generated
   InClientID runtime.InVariable[string] `spec:"title=Client ID,type=string,scope=Message,name=clientId,messageScope,jsScope"`
   InSessionToken runtime.InVariable[string] `spec:"title=Session Token,type=string,scope=Message,name=sessionToken,messageScope,jsScope"`
   ```

#### Complete Example Implementation

```go
type CreatePageNode struct {
    runtime.Node `spec:"id=Notion.CreatePage,name=Create Page,icon=mdiNotebook,color=#000000"`
    
    // Connection ID - typically from Connect node (Message scope only)
    InClientID runtime.InVariable[string] `spec:"title=Client ID,type=string,scope=Message,name=clientId,messageScope,jsScope"`
    
    // User-friendly inputs - Custom scope for easy manual entry
    InPageTitle runtime.InVariable[string] `spec:"title=Page Title,type=string,scope=Custom,name=My New Page,customScope,messageScope,jsScope"`
    
    // Optional parameters
    OptTimeout runtime.OptVariable[int] `spec:"title=Timeout (seconds),type=int,value=30,scope=Message,name=timeout,messageScope,customScope"`
    
    // Outputs
    OutPageID runtime.OutVariable[string] `spec:"title=Page ID,type=string,scope=Message,name=pageId,messageScope"`
    OutPageURL runtime.OutVariable[string] `spec:"title=Page URL,type=string,scope=Message,name=pageUrl,messageScope"`
}
```

### 5.5 Variables cheat-sheet

| Use-case | Type | Example | Access | Naming Pattern |
|----------|------|---------|--------|----------------|
| **Mandatory input**  | `InVariable[T]`  | `InPageTitle InVariable[string]` | `.Get(ctx)` | `In` + PascalCase |
| **Optional input**   | `OptVariable[T]` | `OptTimeout OptVariable[int]` | `.Get(ctx)` | `Opt` + PascalCase |
| **Output**           | `OutVariable[T]` | `OutResult OutVariable[string]` | `.Set(ctx,val)` | `Out` + PascalCase |
| **Credential**       | `Credential`     | `OptToken Credential` | `.Get(ctx)` | `Opt` + PascalCase |
| **Enum option**      | raw `string`     | `OptLabel string` | direct (`n.OptLabel`) | `Opt` + PascalCase |
| **Bool option**      | raw `bool`       | `OptIsHTML bool` | direct (`n.OptIsHTML`) | `Opt` + PascalCase |

Variable wrappers (`InVariable`, `OptVariable`, `OutVariable`) use `.Get(ctx)` / `.Set(ctx,val)`. Raw types (enums, bool options) are accessed directly on the struct — no `.Get()` call.

### 5.6 Robomotion Variable Type Rules

**CRITICAL**: Follow these standardized variable type rules for all Robomotion package development:

#### Variable Type Requirements

1. **Enums**: Always use raw `string` type with `option` tag, never `InVariable[string]` or `OptVariable[string]`
   ```go
   // CORRECT - Raw string with option tag
   OptBundle string `spec:"title=Bundle,value=messaging_non_clips,enum=clips_grid_picker|messaging_non_clips,enumNames=Efficient Clip Grid|Quick GIFs,option"`

   // INCORRECT - Never use Variable wrappers for enums
   OptBundle InVariable[string] `spec:"..."` // ❌ Wrong
   OptBundle OptVariable[string] `spec:"..."` // ❌ Wrong
   ```

2. **Bool options**: Always use raw `bool` type with `option` tag, never `OptVariable[bool]`
   ```go
   // CORRECT - Raw bool with option tag and default value
   OptIsHTML bool `spec:"title=HTML Content,value=false,option,description=Send body as HTML content"`

   // Usage: direct field access (no .Get() call)
   if n.OptIsHTML {
       // handle HTML
   }

   // INCORRECT - Never use OptVariable for bool options
   OptIsHTML OptVariable[bool] `spec:"..."` // ❌ Wrong - causes name=false bug in spec tags
   ```

3. **Required Inputs**: Use `InVariable[T]` type
   ```go
   InDatabaseId InVariable[string] `spec:"title=Database ID,type=string,scope=Message,name=databaseId,messageScope,jsScope,customScope"`
   ```

4. **Optional Inputs**: Use `OptVariable[T]` type with `option` tag (except for enums and bools — see rules 1-2)
   ```go
   OptFilter OptVariable[interface{}] `spec:"title=Filter,type=object,option"`
   OptTimeout OptVariable[int] `spec:"title=Timeout (seconds),type=int,value=30,option"`
   ```

5. **Outputs**: Use `OutVariable[T]` type
   ```go
   OutResult OutVariable[interface{}] `spec:"title=Result,type=object,scope=Message,name=result,messageScope"`
   ```

6. **JSON Data**: Always use `interface{}` for complex data structures, never `string`
   ```go
   // CORRECT - Use interface{} for JSON data
   InRequestBody InVariable[interface{}] `spec:"title=Request Body,type=object"`
   OutResponseData OutVariable[interface{}] `spec:"title=Response Data,type=object"`
   OptMetadata OptVariable[interface{}] `spec:"title=Metadata,type=object,option"`
   
   // INCORRECT - Never use string for JSON
   InRequestBody InVariable[string] `spec:"..."` // ❌ Wrong
   ```

#### Spec Tag Rules

1. **Unicode Commas**: Use Unicode commas (，) in descriptions to avoid breaking spec tag parsing
   ```go
   `spec:"description=Connects to external API，retrieves data，and processes results"`
   ```

2. **Business-Focused Descriptions**: Write user-friendly descriptions for Flow Designer tooltips
   ```go
   `spec:"title=Database ID,description=Unique identifier for the Notion database"`
   ```

3. **Purpose-Driven**: Explain the purpose and expected input/output formats
   ```go
   `spec:"title=Filter Criteria,description=JSON object containing query filters for database search"`
   ```

#### Function Logic Rules

1. **Enum and Bool Option Access**: Access enum and bool option fields directly (e.g., `n.OptLabelFilter`, `n.OptIsHTML`), not via `.Get()` calls
   ```go
   // CORRECT - Direct access for enum fields
   if n.OptSearchType == "pages" {
       // Process pages
   }

   // CORRECT - Direct access for bool option fields
   if n.OptIsHTML {
       // Handle HTML content
   }

   // INCORRECT - Never use .Get() for enums or bool options
   searchType, _ := n.OptSearchType.Get(ctx) // ❌ Wrong
   isHTML, _ := n.OptIsHTML.Get(ctx)          // ❌ Wrong
   ```

2. **Variable Scope**: Maintain proper scope and naming for all variable types
3. **JSON Handling**: Preserve `interface{}` types for complex data structures

### 5.7 Enumerations (`enum` / `enumNames`)

Use **`enum`** to define the allowed values of a field and **`enumNames`** to provide human-readable labels.  
A common use-case is to show a dropdown in the Designer where the developer picks one of the options.

```go
type Divider struct {
    runtime.Node `spec:"id=Example.Divider,name=Divider"`

    InBorder string `spec:"title=Border,option,value=solid,enum=solid|dashed|dotted,enumNames=Solid|Dashed|Dotted,description=Border style"`
}
```

Things to remember:

1. The SDK splits on the pipe symbol (`|`) – **no spaces** inside the list.  
2. The **position** of each item in `enumNames` must match the one in `enum`.
3. Because `InBorder` is a **plain string field marked as `option`**, you can use it directly inside your node **without** calling `.Get(ctx)`:
   
   ```go
   log.Println("chosen border:", n.InBorder)
   ```
4. Only variable wrappers (`InVariable`, `OptVariable`, `OutVariable`) require `.Get(ctx)` / `.Set(ctx)` accessors.
5. Enums work on most primitive types too – for integers just list the numbers: `enum=0|1|2`.

---

## 6. Node Lifecycle

1. **OnCreate** – called exactly once when the flow starts or the node is (re-)deployed. Heavy initialisation goes here.
2. **OnMessage** – called for every incoming token.  
   • Retrieve inputs via variables.  
   • Call external APIs via the runtime helper.  
   • Emit outputs by setting `OutVariable`s.  
   • Return an error to stop the entire flow **unless** the node property `ContinueOnError` (inherited from `runtime.Node`) is `true`.
3. **OnClose** – counterpart to *OnCreate*. Close files, flush buffers, etc.

Delays can be added via `DelayBefore` and `DelayAfter` (milliseconds) – especially useful for rate-limited APIs.

### 6.1 Interactive setup (`OnSetup`) — optional lifecycle

Some nodes need a one-time **interactive setup** before they can run: a WhatsApp
QR scan, an OAuth redirect, a device-code entry, a "test connection". Doing this
*during a flow run* is awkward (the user isn't watching the robot, and a cloud
robot has no screen). `OnSetup` lets the **UI** drive that handshake on the
connected robot **outside any flow run** — e.g. inside a hire/onboarding wizard —
and persist the result (a session blob, a token) into the node's bound vault item
so the real run just works.

It is **opt-in and interface-gated**. A node implements the optional
`SetupHandler` interface; the host invokes `OnSetup` only on nodes that do.
Nodes that don't implement it report an unsupported error, and older hosts that
predate `OnSetup` simply never call it — so it is fully backward-compatible.
(There is no `OnSetup` capability bit — gating is the interface check, not a
negotiated capability.)

```go
type SetupHandler interface {
    OnSetup(ctx runtime.SetupContext) error
}
```

When the host runs setup it instantiates your node **from its config exactly like
`OnCreate`** (so your `Opt*` fields and the bound `runtime.Credential` are
populated), spawns the package process, and calls `OnSetup`. There is **no
message loop** — `OnMessage` is never called for a setup run. `OnSetup` may block
for the whole handshake (e.g. the ~3-minute QR window); the process is torn down
afterwards.

`SetupContext` is how you talk to the UI:

| Method | Purpose |
|--------|---------|
| `Emit(SetupEvent) error` | Push a prompt/status frame **up** to the UI (rendered live; the UI REST-polls these). Call it for every rotating QR / status change. |
| `Await(SetupInputSpec, timeoutSec) (SetupInput, error)` | Block for user-supplied input (a code, a confirmation). Optional — no-input setups never call it. |
| `Config() []byte` | The node's raw config JSON (same bytes `OnCreate` parses). |
| `GUID()`, `SessionID()`, `Context()` | Node GUID, the setup session id, and a `context.Context` that cancels on host timeout/abort. |

`SetupEvent` is channel-agnostic so one shape serves every kind of setup:

```go
type SetupEvent struct {
    Kind      string // "qr" | "code" | "oauth" | "status" | "prompt"
    Step      string // "pending" | "completed" | "timeout" | "error"
    Image     string // data URL, e.g. a QR PNG (the UI renders it with a plain <img>)
    Text      string // a code / OAuth URL / value to display
    Message   string // human-facing status or error text
    ExpiresAt string // when the current code rotates (optional)
}
```

Emit a terminal `Step` (`completed` / `timeout` / `error`) when the handshake ends —
the UI advances on `completed`.

**Example — a QR-pairing setup:**

```go
import (
    "encoding/base64"
    qrcode "github.com/skip2/go-qrcode"
    "github.com/robomotionio/robomotion-go/runtime"
)

// Connect supports interactive setup. Implementing SetupHandler makes the host
// offer an in-wizard setup step for this node (gated by the interface, no
// capability bit).
func (n *Connect) OnSetup(ctx runtime.SetupContext) error {
    // n.OptCredentials is the bound (possibly empty) vault item — populated
    // from config just like in OnCreate.

    // … open the provider's pairing channel; for each rotating code: …
    png, _ := qrcode.Encode(code, qrcode.Medium, 320)
    ctx.Emit(runtime.SetupEvent{
        Kind:  "qr",
        Step:  "pending",
        Image: "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
    })

    // … when the provider signals the scan succeeded, persist the session to the
    // bound credential (same vault API you'd use in OnMessage) …
    // n.OptCredentials.Set(msgCtx, sessionBlobJSON)

    ctx.Emit(runtime.SetupEvent{Kind: "status", Step: "completed", Message: "Linked"})
    return nil
}
```

Notes:
- **Keep the run-time path too.** `OnSetup` is *additive* — if a node can also
  pair/auth lazily during a flow run, leave that path in place for older robots
  and standalone use. `OnSetup` is the better UX, not a hard requirement.
- For an **input-driven** setup (device code, confirmation), use `Await`; the UI
  posts the input back and the call returns it (or a timeout).
- The QR/relay delivery is handled entirely by the host (runner → deskbot → api,
  surfaced to the UI by REST poll). Your node only `Emit`s structured events; it
  never talks to the api or a WebSocket itself.

---

## 7. Working with Credentials & RPA Vault

Robomotion includes a secure **RPA Vault** system for managing sensitive data like API tokens, passwords, database credentials, and certificates. Nodes can access vault items through the `runtime.Credential` type.

### 7.1 Declaring a Credential Field

```go
type MyAPINode struct {
    runtime.Node `spec:"id=Acme.MyAPI,name=My API"`
    
    // Credential field - allows user to select from vault
    OptToken runtime.Credential `spec:"title=API Token,scope=Custom,category=4,messageScope,customScope"`
}
```

### 7.2 Accessing Vault Items

```go
func (n *MyAPINode) OnMessage(ctx message.Context) error {
    // Retrieve the vault item
    item, err := n.OptToken.Get(ctx)
    if err != nil {
        return err
    }
    
    // Extract the token value
    token, ok := item["value"].(string)
    if !ok {
        return runtime.NewError("ErrInvalidArg", "No Token Value")
    }
    
    // Use the token for API calls
    client := &http.Client{}
    req, _ := http.NewRequest("GET", "https://api.example.com/data", nil)
    req.Header.Set("Authorization", "Bearer "+token)
    resp, err := client.Do(req)
    // ... handle response
    
    return nil
}
```

### 7.3 Credential Types & Item Structure

The vault supports 8 different credential types. Each has a specific JSON structure:

| Type | Category | Common Keys | Usage |
|------|----------|-------------|-------|
| **Login** | 1 | `username`, `password` | HTTP auth, FTP, browser automation |
| **Email** | 2 | `inbox.username`, `inbox.password`, `smtp.server`, `smtp.port` | Email operations |
| **Credit Card** | 3 | `cardholder`, `cardnumber`, `cvv`, `expDate` | Payment processing |
| **API Key/Token** | 4 | `value` | API authentication, Office 365 |
| **Database** | 5 | `server`, `port`, `database`, `username`, `password` | Database connections |
| **Document** | 6 | `filename`, `content` | OAuth tokens, certificates |
| **AES Key** | 7 | `value` | Encryption/decryption |
| **RSA Key Pair** | 8 | `publicKey`, `privateKey` | Cryptographic operations |

### 7.4 Common Usage Patterns

**API Token (most common):**
```go
item, err := n.OptToken.Get(ctx)
if err != nil {
    return err
}
token := item["value"].(string)
```

**Username/Password:**
```go
item, err := n.OptLogin.Get(ctx)
if err != nil {
    return err
}
username := item["username"].(string)
password := item["password"].(string)
```

**Database Connection:**
```go
item, err := n.OptDatabase.Get(ctx)
if err != nil {
    return err
}
server := item["server"].(string)
port := int(item["port"].(float64))
database := item["database"].(string)
username := item["username"].(string)
password := item["password"].(string)
```

**Email Configuration:**
```go
item, err := n.OptEmail.Get(ctx)
if err != nil {
    return err
}
smtpServer := item["smtp"].(map[string]interface{})["server"].(string)
smtpPort := int(item["smtp"].(map[string]interface{})["port"].(float64))
smtpUser := item["smtp"].(map[string]interface{})["username"].(string)
smtpPass := item["smtp"].(map[string]interface{})["password"].(string)
```

### 7.5 Error Handling

Always check for the existence of required keys:
```go
item, err := n.OptToken.Get(ctx)
if err != nil {
    return err
}

value, exists := item["value"]
if !exists {
    return runtime.NewError("ErrInvalidArg", "Missing required field: value")
}

token, ok := value.(string)
if !ok {
    return runtime.NewError("ErrInvalidArg", "Invalid token format")
}
```

### 7.6 Vault Item Metadata

All vault items include metadata when retrieved:
```go
meta := item["meta"].(map[string]interface{})
vaultId := meta["vaultId"].(string)
itemId := meta["itemId"].(string)
name := meta["name"].(string)
category := int(meta["category"].(float64))
```

### 7.7 Shared Credential Pattern

For packages with multiple nodes requiring the same credentials, implement a shared credential store:

```go
// v1/common.go - Shared credential store
var (
    credentialStore = make(map[string]interface{})
    credentialMutex = sync.RWMutex{}
)

func setCredential(clientID string, credential interface{}) {
    credentialMutex.Lock()
    defer credentialMutex.Unlock()
    credentialStore[clientID] = credential
}

func getCredential(clientID string) (interface{}, bool) {
    credentialMutex.RLock()
    defer credentialMutex.RUnlock()
    cred, exists := credentialStore[clientID]
    return cred, exists
}
```

**Usage Pattern:**
1. **Connect Node** - Gets credential from vault, generates client ID, stores credential
2. **Operation Nodes** - Use client ID to retrieve stored credential
3. **Disconnect Node** - Clean up stored credential

This avoids requiring users to select credentials for every node in a flow.

---

## 8. Registering & Starting

`main.go` is trivial – list every versioned node and let the runtime take control:

```go
func main() {
    runtime.RegisterNodes(
        &v1.Hello{},  // v1 directory
        &v1.World{},
    )
    runtime.Start()
}
```

> **Multiple versions** – You can have `v2/`, `v3/` directories. Just import and register them side by side.

---

## 9. Build & Run locally

```bash
# Build the binary for your host OS
roboctl package build

# Run it inside a Robot (assumes you are developing *inside* a robot)
./dist/my-package -a   # "attach" – streams logs & debugging to Designer
```

### 9.1 Generate *pspec* only
Sometimes you just want to refresh the Designer specification (JSON) without rebuilding everything:

```bash
./dist/my-package -s   # outputs <namespace>-<version>.pspec next to the binary
```

The file is created by `runtime.generateSpecFile()` and automatically picked up by roboctl when packaging.

### 9.2 Cross-compiling & multi-arch builds

Need packages for Windows, macOS and different CPU architectures? `roboctl` wraps the `go build` commands declared in `config.json`, so you only have to pass the desired architecture:

```bash
# Build for Windows on a Linux workstation
roboctl package build --arch windows/amd64

# Build for Apple silicon (macOS arm64)
roboctl package build --arch darwin/arm64
```

If you keep the default build scripts (`go build -o dist/<name>`), `roboctl` automatically sets `GOOS`/`GOARCH` before invoking them.  
Multiple `--arch` values can be passed or repeated to emit several binaries in a single run.

---

## 10. Debugging & Logs (attach / detach)

During local development you often want to see **stdout/stderr** and runtime events inside the Robomotion **Designer**. The SDK already includes helper functions in `debug/`:

| CLI Flag | Meaning |
|----------|---------|
| `-a` | **Attach** – the process discovers the local robot, connects via gRPC and streams logs/errors |
| *(none)* | Run standalone – useful for unit tests |

During an attached session **the robot will prefer the already-running binary** over any version found in the repository.  
Thus the workflow is:

```bash
# 1. Rebuild after code change
roboctl package build

# 2. Start (or restart) in attach mode
./dist/my-package -a &   # keep it running in background

# 3. In Designer ➜ run your flow that uses the package
#    The robot detects the listening plugin and routes messages to it.
```

Inside your node you can sprinkle regular `log.Printf` statements:

```go
func (n *Hello) OnMessage(ctx message.Context) error {
    log.Printf("Hi there! incoming GUID=%s", n.GUID)
    // … business logic …
    return nil
}
```

Anything written to `stdout` or `stderr` will appear in the **Console** panel of the Designer while the flow is running, making printf-style debugging extremely quick.

---

## 11. Anatomy of the generated *.pspec* file

The spec file (sometimes called *pspec*) is what the Designer consumes to render nodes, editors and ports.

* Generated by `generateSpecFile()` (invoked when the binary starts with **`-s`**).  
* File name: `<namespace>-<version>.pspec` (written to *stdout* – `roboctl` captures and stores it next to the binary).  
* Top-level keys:
  * `name` – package display name
  * `version`
  * `nodes[]` – array with everything described in section 5 (ID, icon, colors, `properties[]`, `customPorts[]` …)

Open it once and you’ll immediately see where your tag information ends up – this is invaluable when something doesn’t look right in the Designer.

---

## 12. Repository Management

### 12.1 Building Packages

```bash
# Build package
roboctl package build

# Build for specific platforms
roboctl package build --arch windows/amd64
roboctl package build --arch darwin/arm64
roboctl package build --arch linux/amd64
```

### 12.2 Multi-Platform Building

Build packages for different target platforms:

```bash
# Build for specific platforms
roboctl package build --arch linux/amd64
roboctl package build --arch windows/amd64  
roboctl package build --arch darwin/amd64
roboctl package build --arch darwin/arm64

# Build multiple platforms at once
roboctl package build --arch linux/amd64 --arch windows/amd64 --arch darwin/amd64
```

### 12.3 Repository Management

Create and serve package repositories:

```bash
# Generate repository index from .tgz files
roboctl repo index

# Merge with existing index (preserve previous packages)
roboctl repo index --merge

# Serve repository locally for testing
roboctl repo serve --port 8080

# Serve on custom IP and port
roboctl repo serve --ip 0.0.0.0 --port 3000
```

**Repository Server Features:**
- Serves package downloads (`*.tgz` files)
- Provides index access (`index.json`, `index.sha256sum`)
- Delivers specification files (`*.pspec`) for Flow Designer
- CORS support for web-based access


---

## 13. Publishing Packages

1. Update version in `config.json`
2. Build for target platforms: `roboctl package build --arch <platform>`
3. Generate repository index: `roboctl repo index`
4. Serve or deploy packages to your repository

---

## 14. Development & Testing

### 14.1 Development Tips

**Attach Mode for Development**:
Use attach mode for rapid testing of logic changes:

```bash
# Build and run in attach mode for development
go build -o dist/my-package
./dist/my-package -a
```

**Debugging with Logs**:
```go
func (n *MyNode) OnMessage(ctx message.Context) error {
    log.Printf("Processing message for node %s", n.GUID)
    
    input, err := n.InData.Get(ctx)
    if err != nil {
        log.Printf("Error getting input: %v", err)
        return err
    }
    
    // Your business logic here
    log.Printf("Processing complete")
    return nil
}
```

**Note**: Attach mode only works for logic changes. For structural changes (new nodes, property changes), use the full build and deployment workflow.

### 14.2 Development Workflow Decision Tree

Use this decision tree to determine the correct development approach:

```
┌─────────────────────────────────┐
│     Made changes to code?       │
└─────────────┬───────────────────┘
              │
              ▼
┌─────────────────────────────────┐
│    Did you add/modify any:      │
│  • New nodes                    │
│  • Node properties (spec tags) │  
│  • InVariable/OutVariable       │
│  • Node titles/descriptions     │
│  • Icons, colors, categories    │
└─────────────┬───┬───────────────┘
              │   │
         YES  │   │ NO
              ▼   ▼
    ┌─────────────────┐    ┌──────────────────┐
    │ Use FULL        │    │ Use ATTACH MODE  │
    │ WORKFLOW        │    │ (Quick & Easy)   │
    │                 │    │                  │
    │ 1. roboctl      │    │ 1. go build      │
    │    package      │    │ 2. ./binary -a   │
    │    build        │    │ 3. Run flows     │
    │ 2. roboctl repo │    │ 4. View logs     │
    │    index        │    │                  │
    │ 3. roboctl repo │    │ ✅ Real-time     │
    │    serve        │    │    debugging     │
    │ 4. Clear cache  │    │ ✅ log.Printf    │
    │ 5. Refresh      │    │    output        │
    │    Designer     │    │ ✅ Fast iteration│
    └─────────────────┘    └──────────────────┘
```

**Quick Reference:**
- **Logic changes only** → Use `-a` attach mode
- **Any structural changes** → Use full workflow with `roboctl repo serve`
- **When in doubt** → Use full workflow (safer but slower)

---

## 15. Common Issues & Troubleshooting

### 15.1 Build and Runtime Issues

| Symptom | Explanation | Solution |
|---------|-------------|----------|
| `timeout: plugin listener is nil` | Binary launched but never called `runtime.Start()` | Add `runtime.Start()` call in your `main.go` |
| "node handler not found" | Node not registered or ID mismatch | Check node registration in `main.go` and verify node ID matches Designer |
| "plugin already attached" error | Multiple plugin instances running | Stop previous instances, check running processes |
| Empty pspec file | Missing struct tag on Node | Add `spec` struct tag to embedded `runtime.Node` |
| Package not updating | Robot using cached version | Clear package cache or increment version in `config.json` |
| Node not appearing in Designer | Node ID doesn't match namespace | Ensure node ID starts with namespace from `config.json` (e.g., if namespace is `Acme.Weather`, node ID must be `Acme.Weather.NodeName`) |

### 15.2 Variable and Scope Issues

| Symptom | Explanation | Solution |
|---------|-------------|----------|
| Variables always empty | Incorrect scope configuration | Check `scope` and `messageScope` flags in spec tags |
| Input not appearing in Designer | Missing scope flags | Add `messageScope,jsScope,customScope` for InVariable |
| Custom fields not working | Incorrect Custom scope setup | Use `scope=Custom,name=Field Label,customScope` |
| Enum dropdown not showing | Missing enum configuration | Add `enum=val1\|val2,enumNames=Label1\|Label2,option` |

### 15.3 Flow Designer Issues

| Symptom | Explanation | Solution |
|---------|-------------|----------|
| Node not appearing in palette | Package not properly imported | Check repository index and package installation |
| Port names missing | Missing port configuration | Add `direction` and `position` tags to Port fields |
| Wrong node icon/color | Incorrect spec configuration | Verify `icon` and `color` in Node spec tag |
| Properties not updating | Stale pspec file | Rebuild package and refresh Designer |

### 15.4 Development Workflow Issues

| Symptom | Explanation | Solution |
|---------|-------------|----------|
| Attach mode not working | Robot not found or flows running | Ensure robot connected, stop running flows |
| Debug logs not showing | Not using attach mode | Run with `./binary -a` flag |
| Changes not reflected | Using wrong development approach | Use attach mode for logic changes, full rebuild for structural changes |

---

## 16. Important Notes

### Commas in Spec Tags
Avoid regular commas in descriptions as they break parsing. Use Unicode comma `，` (U+FF0C) instead:

```go
// Wrong - breaks parsing
runtime.Node `spec:"description=Hello, world"`

// Correct - use Unicode comma
runtime.Node `spec:"description=Hello，world"`
```

---

## 17. AI Agent Tools (Tool / Toolkit / SkillProvider)

Any node can be exposed to an AI agent runtime (HermesAgent, ADK Agent, …) as a
callable tool, with **no per-package code inside the agent**. The agent reads the
node's pspec, registers the tool(s), and dispatches calls over the normal
Robomotion message bus. Three opt-in markers, all additive and independent:

| You want… | Add | pspec key | Agent consumes as |
|-----------|-----|-----------|-------------------|
| One callable tool | `runtime.Tool` field + `tool:"…"` tag | `tool` | one tool registration |
| Many tools in one node | `runtime.Toolkit` field + `ToolkitProvider` | `tools[]` | N tool registrations, one node guid |
| System-prompt guidance | `SkillProvider` interface | `skill` | text appended to the agent system message |

> **Deep dive:** `docs/toolkit-skill-provider.md` covers the full mental model,
> the wire protocol, and the production Agent Teams toolkit. This section is the
> quick reference.

### 17.1 Single Tool (`runtime.Tool`)

Embed `runtime.Tool` and tag it. The node still works as a normal flow node;
the marker just makes it *also* agent-callable (and CLI-callable, see §18).

```go
type CreateIssue struct {
    runtime.Node `spec:"id=Linear.CreateIssue,name=Create Issue,icon=mdiPlus,color=#5e6ad2"`
    runtime.Tool `tool:"name=linear_create_issue,description=Create a Linear issue"`

    InTitle runtime.InVariable[string] `spec:"title=Title,type=string,scope=AI,name=title,aiScope,messageScope"`
    OutID   runtime.OutVariable[string] `spec:"title=Issue ID,type=string,scope=Message,name=issueId,messageScope"`
}
```

When the agent calls the tool, the SDK's `ToolInterceptor` routes the request
to your `OnMessage`. Use `scope=AI` + `aiScope` on the inputs the LLM should
fill. If `OnMessage` returns without calling `ToolResponse`, the interceptor
auto-collects every `OutVariable` and replies with a success payload.

### 17.2 Toolkit (`runtime.Toolkit` + `ToolkitProvider`)

One node, many tools. Embed `runtime.Toolkit`, implement `Tools()`, and
dispatch inside `OnMessage` on the tool name:

```go
type SearchToolkit struct {
    runtime.Node    `spec:"id=Acme.Search.Toolkit,name=Search Toolkit,icon=mdiMagnify,color=#3b82f6"`
    runtime.Toolkit `toolkit:""`
}

func (n *SearchToolkit) Tools() []runtime.ToolDef {
    return []runtime.ToolDef{
        {
            Name:        "web_search",
            Description: "Search the web. Returns the top hits.",
            Schema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "query": map[string]interface{}{"type": "string", "description": "Free-text query."},
                },
                "required": []string{"query"},
            },
        },
        // … more ToolDefs …
    }
}

func (n *SearchToolkit) OnCreate() error { return nil }
func (n *SearchToolkit) OnClose() error  { return nil }
func (n *SearchToolkit) OnMessage(ctx message.Context) error {
    if !runtime.IsToolRequest(ctx) {
        return nil
    }
    params := runtime.ToolParameters(ctx)
    switch runtime.ToolName(ctx) {
    case "web_search":
        hits, err := webSearch(params["query"].(string))
        if err != nil {
            return runtime.ToolResponse(ctx, "error", nil, err.Error())
        }
        return runtime.ToolResponse(ctx, "success", map[string]interface{}{"results": hits}, "")
    }
    return runtime.ToolResponse(ctx, "error", nil, "unknown tool: "+runtime.ToolName(ctx))
}
```

`Tools()` is called **once at build time** during spec generation; the list is
frozen in the pspec. The four request helpers:

| Helper | Returns |
|--------|---------|
| `runtime.IsToolRequest(ctx)` | `bool`, is this an agent tool call? |
| `runtime.ToolName(ctx)` | `string`, which tool (the discriminator for a Toolkit) |
| `runtime.ToolParameters(ctx)` | `map[string]interface{}`, the call args |
| `runtime.ToolResponse(ctx, status, data, errMsg)` | sends the reply and stops downstream flow |

### 17.3 SkillProvider: ship guidance with the tools

Implement `Skill() string` on any node (toolkit, single tool, or even a no-tool
"context" node). The returned markdown is appended to the consuming agent's
system message when the node is wired to its tools port. It is the in-tree
equivalent of MCP's `prompts/get`. A common pattern is `//go:embed AGENT.md`.

```go
//go:embed AGENT.md
var agentMD string

func (n *SearchToolkit) Skill() string { return agentMD }
```

### 17.4 Letting flow authors choose which tools to expose

Add an `OptVariable[[]string]` with an `enum` tag. The SDK emits a multi-select
checkbox grid (`ui:field=multiSelectCheckbox`); the selected names arrive as a
`[]string`. Convention: **empty = expose all** (users opt out, not in).

```go
OptEnabled runtime.OptVariable[[]string] `spec:"title=Enabled Tools,type=array,scope=Custom,name=,customScope,enum=web_search|image_search|news_search,enumNames=Web Search|Image Search|News Search,description=Empty = all"`
```

### 17.5 Credentials for a Toolkit — pick the right vault type

A Toolkit node is usually **action-only**: unlike WhatsApp/Telegram (which have
a `Receive`/session node the credential hangs on), a Ghost or WordPress toolkit
has nowhere else to carry auth, so it **declares its own `runtime.Credential`
field** and resolves it per call:

```go
type Toolkit struct {
    runtime.Node    `spec:"id=Acme.Wordpress.Agents.Toolkit,name=Toolkit,icon=mdiWordpress,color=#21759b"`
    runtime.Toolkit `toolkit:""`

    OptCredentials runtime.Credential `spec:"title=Credentials,category=1,scope=Custom,messageScope,customScope,description=…"`
}
```

**Choosing the `category` is a real decision — get it right:**

1. **Use the matching PRIMITIVE type when the credential fits one** (the 8 types
   in §7.3). The category is a label the Vault UI and the hire wizard use to
   render/group/reuse the item, so be honest:
   - `username` + `password` → **Login (1)** (e.g. WordPress app-password).
   - a single token → **API Key/Token (4)** (e.g. Ghost Admin key, OpenRouter).
   - `server`/`port`/`database`/`username`/`password` → **Database (5)**, etc.

2. **Only when the secret set fits NO primitive, use Document (6) — a generic
   JSON bag.** The vault stores every item's fields as one encrypted JSON blob
   keyed by name (category is just an `int` label; there is **no** per-category
   field schema — see `robomotion-deskbot/runtime/vault.go`), so a Document item
   can hold any keys you define. Use it for opaque/odd secrets that don't map to
   a primitive — e.g. a WhatsApp **session** blob (`runtime_paired`), or an auth
   set with an unusual combination of fields. `cred.Get(ctx)` returns the JSON
   as `map[string]interface{}` either way.
   - **Don't force Document "because a toolkit might need anything."** Pick the
     honest primitive when one fits; reach for Document only when none do.

3. **Non-secret config is NOT a credential.** A base URL, a region, a workspace
   id — these are plain **node properties** (`OptVariable[string]`), never vault
   fields. Wire them as a normal option/setting, not into `OptCredentials`. (So
   the WordPress toolkit takes `OptSiteUrl` as a property and only
   `username`/`password` as the Login credential.)

When the toolkit is hired through the Agent Hub, the consuming role's
`credentials.yaml` slot must declare the **same `category`** (and, for Document,
the `fields` it collects). The role groups its slots for onboarding with two
opaque author strings: `group:` co-locates a credential + its companion property
(e.g. a WordPress login + its Site URL) in one wizard step, and `phase:` buckets
steps into the left-rail sections (`Essentials` / `Integrations` / `Tuning`).
See the Agent Hub guide `how-to-write-credentials-yaml.md` (§7 `group`, §8
`phase`) for the slot side and how a credential + a companion property are
presented in onboarding.

---

## 18. CLI Command Mode (the binary as a standalone tool)

Every package binary can run **without a robot** as a self-contained CLI
program. Any node with a `runtime.Tool` field becomes a subcommand. This is what
lets AI-agent skills and shell scripts call a package directly.

> **Deep dive:** `docs/package-cli.md`. Quick reference below.

`runtime.Start()` inspects `os.Args[1]` and dispatches by mode:

| First arg | Mode |
|-----------|------|
| *(none)* | gRPC plugin (normal robot runtime) |
| `-a` | Attach (debug, see §10) |
| `-s` | Print the pspec JSON to stdout |
| `--skill-md` | Print a generated `SKILL.md` to stdout |
| `--list-commands [cmd]` | List CLI commands (add `--output json` for machine output) |
| `--help` / `-h` | Usage |
| anything not starting with `-` | **CLI mode**, treat as a command name |

```bash
# Discover
robomotion-googledrive --list-commands
robomotion-googledrive --skill-md > SKILL.md

# Invoke (flags are kebab-case of each variable's spec name; output is JSON on stdout)
robomotion-googledrive upload_file --file-path=/tmp/x.pdf --folder-id=abc123
```

**Credentials** resolve through the vault, by name or by ID:

```bash
robomotion-googledrive upload_file --vault="My Vault" --item="Drive Token" --file-path=/tmp/x.pdf
robomotion-googledrive upload_file --vault-id=<id>     --item-id=<id>       --file-path=/tmp/x.pdf
```

Auth comes from `robomotion login`, or from the `ROBOMOTION_API_TOKEN` /
`ROBOMOTION_ROBOT_ID` / `ROBOMOTION_API_URL` environment variables (set by the
runner). Each invocation is its own process, naturally isolated and parallel.

**Sessions** keep the process warm for Connect/operation/Disconnect flows that
reuse a connection:

```bash
robomotion-database connect --vault="DB" --item="prod" --session     # → {"session_id": "..."}
robomotion-database query   --sql="SELECT 1" --session-id=<id>
robomotion-database --session-close=<id>
```

Output goes to **stdout as JSON**; errors go to **stderr as JSON** with exit
code `1`. Node code is unchanged: the same `InVariable.Get` / `OutVariable.Set`
/ `Credential.Get` calls work because CLI mode installs a `CLIRuntimeHelper` in
place of the gRPC client.

---

## 19. OAuth2 Authorization Helper

For nodes that need an interactive OAuth2 authorization-code grant, the SDK
ships `runtime.OpenOAuthDialog`. It opens the system browser, runs a local
callback server on a **fixed** redirect URL, and returns the authorization code.

```go
import (
    "context"
    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
    "github.com/robomotionio/robomotion-go/runtime"
)

func (n *Connect) OnMessage(ctx message.Context) error {
    cfg := &oauth2.Config{
        ClientID:     clientID,
        ClientSecret: clientSecret,
        Scopes:       []string{"https://www.googleapis.com/auth/drive"},
        Endpoint:     google.Endpoint,
        RedirectURL:  runtime.OAuth2RedirectURL, // forced to http://localhost:9876/oauth2/callback
    }
    code, err := runtime.OpenOAuthDialog(cfg) // blocks up to 5 min for the user
    if err != nil {
        return err
    }
    token, err := cfg.Exchange(context.Background(), code)
    if err != nil {
        return err
    }
    // … persist token (e.g. into a Credential) for later runs …
    return nil
}
```

Notes:
- The redirect URL is **always** `runtime.OAuth2RedirectURL`
  (`http://localhost:9876/oauth2/callback`). Register exactly that in your
  OAuth app. `OpenOAuthDialog` overwrites `cfg.RedirectURL` to match.
- The callback port (`9876`) is shared, so the helper retries binding for up to
  5 minutes if another OAuth flow holds it; the user-authorization wait is also
  capped at 5 minutes.
- The request asks for offline access + forced consent so a refresh token is
  returned. For a cloud robot with no browser, prefer the `OnSetup` lifecycle
  (§6.1) and drive the OAuth handshake from the UI instead.

---

## 20. Large Message Objects (LMO)

Large message fields (≥ 4 KB) are transparently swapped for `BlobRef` markers
backed by zstd-compressed, content-addressed blobs on disk. This keeps the gRPC
messages small while still letting nodes pass big payloads. The SDK does
**read + write**: it resolves incoming `BlobRef`s and packs large outgoing
values automatically, so most nodes never touch the API.

LMO is **capability-gated** (`CapabilityLMO`, advertised by default): it only
activates when the robot also supports it; otherwise payloads stay inline.

For the rare cases you need manual control:

```go
// Resolve one field that may be a BlobRef:
result, _ := runtime.LMOResolve(data, "fieldName")

// Resolve a field plus all nested BlobRefs inside it:
sub, _ := runtime.LMOResolveSubtree(data, "payload")

// Resolve every BlobRef in a payload:
resolved, _ := runtime.LMOResolveAll(data)

// Pack large fields out of a payload into blobs:
packed, _ := runtime.LMOPack(payload)

// Pack a single Go value if it exceeds the threshold:
ref, packed, _ := runtime.PackValue(value) // (refMap, didPack, err)
```

LMO is not initialized in CLI mode (§18); packages that move >1 MB payloads via
LMO need the full robot runtime.

---

## 21. Capabilities (feature negotiation)

The runner and the package each advertise a bitmask of capabilities; a feature
is active only where **both** agree (`runtime.HasCapability`). This is how new
SDK features ship without breaking older robots.

| Capability | Meaning |
|------------|---------|
| `CapabilityLMO` | Content-addressed blob store (§20), advertised by the package by default |
| `CapabilityUseS3`, `CapabilityTerminateOnStop`, `CapabilityIgnoreVersionCheck`, `CapabilityDiagnostics`, `CapabilityLazyMessage` | Robot-side runtime behaviors |

(`OnSetup` is **not** a capability — it's gated by implementing the
`SetupHandler` interface; see §6.1. See `CAPABILITIES.md` for the bit registry.)

You rarely set these by hand: `CapabilityLMO` is on by default. Use
`runtime.HasCapability(cap)` if you need to branch on what the host supports.

