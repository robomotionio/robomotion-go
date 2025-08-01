# Robomotion Go Packages – A Complete Guide

*Version: 2024-07-15*

Welcome to the **Robomotion package development guide**.  
This document walks you through **every single step** required to build a custom Robomotion package with Go – from `go mod init` to publishing a compressed artefact to the Robomotion repository.  
If you are holding this file you already have the [`robomotion-go`](https://github.com/robomotionio/robomotion-go) SDK checked-out locally. We will use it extensively.

---

## 1. Prerequisites

| Tool | Minimum version | Purpose |
|------|-----------------|---------|
| Go   | `1.20`          | Compiles your code & provides `go` tooling |
| `roboctl` | **latest** (run `roboctl version`) | CLI that scaffolds and builds packages |
| Git  | any             | Source-control & versioning |
| A text editor / IDE | – | Coding |

> **Install `roboctl`**  
> ```bash
> go install github.com/robomotionio/roboctl@latest
> export PATH=$PATH:$(go env GOPATH)/bin
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
| *Node* | `runtime.Node` | Embeddable struct that carries common fields (GUID, delays, …) |
| *Variable* | `InVariable[T]`, `OutVariable[T]`, `OptVariable[T]` | Strongly-typed message variables |
| *Lifecycle* | `OnCreate`, `OnMessage`, `OnClose` | Mandatory callbacks a node must implement |
| *Runtime Helper* | interface `RuntimeHelper` | Provided to you inside `Init` – gives access to vault, app requests, file upload, … |
| *Large Message Object (LMO)* | `runtime.LargeMessageObject` | Mechanism to transport objects >64 KB |

You **never** instantiate a node yourself – the runner does that through reflection.

---

## 5. Declaring a Node (the **`spec`** tag)

Every Go struct that embeds `runtime.Node` and is registered in `main.go` becomes a *node* in Designer.  
Metadata is provided via a **struct-tag** called `spec`.

```go
// v1/hello.go
package v1

import (
    "github.com/robomotionio/robomotion-go/runtime"
    "github.com/robomotionio/robomotion-go/message"
)

type Hello struct {
    // Node declaration (visible in Designer)
    runtime.Node `spec:"id=Acme.Hello,name=Hello,icon=mdiHand,color=#3498db,inputs=1,outputs=1"`

    // === OPTIONS =========================================================
    InGreeting string `spec:"title=Greeting,value=Hello,option,description=Default greeting"`

    // === INPUT  ==========================================================
    InName runtime.InVariable[string] `spec:"title=Name,type=string,scope=Message,messageScope"`

    // === OUTPUT ==========================================================
    OutGreeting runtime.OutVariable[string] `spec:"title=Greeting,type=string,scope=Message,messageScope"`
}

func (n *Hello) OnCreate() error                 { return nil }
func (n *Hello) OnMessage(ctx message.Context) error {
    name, _ := n.InName.Get(ctx)
    greeting := fmt.Sprintf("%s %s", n.InGreeting, name)
    n.OutGreeting.Set(ctx, greeting)
    return nil
}
func (n *Hello) OnClose() error                  { return nil }
```

### 5.1 Tag reference (fields & node)

Below is a **non-exhaustive** but practically complete list of keys you can use inside a `spec:"…"` tag:

| Key | Applies to | Example | Meaning |
|-----|------------|---------|---------|
| `id` | Node | `Acme.Hello` | **Unique** identifier that the runtime looks up. Prefer dotted notation `<Namespace>.<NodeName>` |
| `name` | Node | `Hello` | Display name in Designer |
| `icon` | Node | `mdiHand` | Material-Design-Icon identifier – resolved through `runtime/icons.go` |
| `color` | Node | `#3498db` | Hex color shown behind the icon |
| `inputs` / `outputs` | Node | `inputs=0` | Override default 1-in 1-out configuration |
| `editor` | Node | `editor=tsx` | Custom code editor language if you have a code property |
| `inFilters` | Node | `inFilters=files` | Hide node unless the incoming link carries the specified *filter* |
| `title` | Field | `title=Greeting` | Human friendly caption |
| `type` | Field | `type=string` | Primitive type (`string`, `int`, `object`, …) |
| `value` | Field | `value=Hello` | Default value for **options** |
| `description` | Field | `description=…` | Tooltip text |
| `enum`, `enumNames` | Field | `enum=a|b|c,enumNames=A|B|C` | Enumerations (the SDK splits on `|`) |
| `scope` | Variable | `scope=Message` | One of `Message`, `Custom`, `JS`, … |
| `messageScope`, `customScope`, `jsScope`, `csScope` | Variable | `messageScope` | Flags that control scope availability in Designer |
| `option` | Field | `option` | Marks the property as a user-configurable *option* (as opposed to runtime input) |
| `arrayFields` | Field | `arrayFields=Label|Value` | For array-of-object variables – names become sub-fields |
| `format` | Field | `format=password` | JSON-schema format for Designer (dates, passwords, …) |
| `hidden` | Field | `hidden` | Field is invisible but still stored |
| `category` | Field | `category=2` | Group fields under collapsible panels |

### 5.2 Variables cheat-sheet

| Use-case | Type | Example |
|----------|------|---------|
| **Mandatory input**  | `InVariable[T]`  | `InPrompt InVariable[string]` |
| **Optional input**   | `OptVariable[T]` | `OptTimeout OptVariable[int]` |
| **Output**           | `OutVariable[T]` | `OutEmbedding OutVariable[any]` |
| **Credential**       | `Credential`     | `OptToken Credential` |

All variable wrappers expose `.Get(ctx)` (for `In`/`Opt`/`Credential`) and `.Set(ctx,val)` (`Out`/`Credential`).

### 5.3 Enumerations (`enum` / `enumNames`)

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

### 7.7 Practical Advice: Shared Credential Management

When building packages with many nodes that all require the same credentials, manually selecting the vault and vault item for each node in the Flow Designer becomes cumbersome. Instead, you can implement a shared credential store pattern within your package.

**Implementation Pattern:**

Create a `common.go` file in your `v1/` directory to hold a shared credential map:

```go
// v1/common.go
package v1

import (
    "sync"
    "github.com/google/uuid"
)

// Global credential store protected by mutex
var (
    credentialStore = make(map[string]interface{})
    credentialMutex = sync.RWMutex{}
)

// Alternative: Use sync.Map for better concurrent performance
// var credentialStore = sync.Map{}

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

func removeCredential(clientID string) {
    credentialMutex.Lock()
    defer credentialMutex.Unlock()
    delete(credentialStore, clientID)
}
```

**Connect Node Implementation:**

```go
// v1/connect.go
type ConnectNode struct {
    runtime.Node `spec:"id=MyPackage.Connect,name=Connect,icon=mdiConnection,color=#2ecc71"`
    
    // User selects credential from vault
    OptToken runtime.Credential `spec:"title=API Credential,scope=Custom,customScope"`
    
    // Output the client ID for other nodes
    OutClientID runtime.OutVariable[string] `spec:"title=Client ID,type=string,scope=Message"`
}

func (n *ConnectNode) OnMessage(ctx message.Context) error {
    // Get credential from vault
    item, err := n.OptToken.Get(ctx)
    if err != nil {
        return err
    }
    
    // Generate unique client ID
    clientID := uuid.New().String()
    
    // Store credential in shared map
    setCredential(clientID, item)
    
    // Output client ID for downstream nodes
    n.OutClientID.Set(ctx, clientID)
    
    return nil
}
```

**Disconnect Node Implementation:**

```go
// v1/disconnect.go
type DisconnectNode struct {
    runtime.Node `spec:"id=MyPackage.Disconnect,name=Disconnect,icon=mdiConnectionOff,color=#e74c3c"`
    
    // Input client ID to disconnect
    InClientID runtime.InVariable[string] `spec:"title=Client ID,type=string,scope=Message"`
}

func (n *DisconnectNode) OnMessage(ctx message.Context) error {
    clientID, err := n.InClientID.Get(ctx)
    if err != nil {
        return err
    }
    
    // Remove credential from shared store
    removeCredential(clientID)
    
    return nil
}
```

**Usage in Other Nodes:**

```go
// v1/api_call.go
type APICallNode struct {
    runtime.Node `spec:"id=MyPackage.APICall,name=API Call,icon=mdiApi,color=#3498db"`
    
    // Input client ID from Connect node
    InClientID runtime.InVariable[string] `spec:"title=Client ID,type=string,scope=Message"`
    InEndpoint runtime.InVariable[string] `spec:"title=Endpoint,type=string,scope=Message"`
    
    OutResponse runtime.OutVariable[string] `spec:"title=Response,type=string,scope=Message"`
}

func (n *APICallNode) OnMessage(ctx message.Context) error {
    clientID, err := n.InClientID.Get(ctx)
    if err != nil {
        return err
    }
    
    // Retrieve credential from shared store
    credItem, exists := getCredential(clientID)
    if !exists {
        return runtime.NewError("ErrInvalidArg", "Client ID not found. Use Connect node first.")
    }
    
    // Extract token from credential
    item := credItem.(map[string]interface{})
    token, ok := item["value"].(string)
    if !ok {
        return runtime.NewError("ErrInvalidArg", "Invalid credential format")
    }
    
    // Use token for API call
    endpoint, _ := n.InEndpoint.Get(ctx)
    client := &http.Client{}
    req, _ := http.NewRequest("GET", endpoint, nil)
    req.Header.Set("Authorization", "Bearer "+token)
    
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // Process response...
    n.OutResponse.Set(ctx, "API call successful")
    
    return nil
}
```

**Flow Designer Usage:**

1. **Connect Node**: User selects credential once, outputs `msg.client_id`
2. **Multiple Operation Nodes**: All receive `msg.client_id` as input, retrieve credential from shared store
3. **Disconnect Node**: Receives `msg.client_id`, cleans up credential from memory

**Benefits:**
- **Single Credential Selection**: Configure credentials once per flow
- **Concurrent Safety**: Protected with mutex or sync.Map
- **Memory Management**: Explicit cleanup via Disconnect node
- **Reusable Pattern**: Works across all nodes in the same package
- **Flow Clarity**: Clear connection lifecycle in Designer

**Thread Safety Options:**
```go
// Option 1: Manual mutex (more control)
var credentialStore = make(map[string]interface{})
var credentialMutex = sync.RWMutex{}

// Option 2: sync.Map (better for high concurrency)
var credentialStore = sync.Map{}

func setCredential(clientID string, credential interface{}) {
    credentialStore.Store(clientID, credential)
}

func getCredential(clientID string) (interface{}, bool) {
    return credentialStore.Load(clientID)
}
```

This pattern is particularly valuable for packages like database connectors, API integrations, or any service requiring authentication across multiple operations.

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

## 12. Repository Management with `roboctl`

The `roboctl` command-line tool provides comprehensive package and repository management capabilities. It handles building packages, creating repositories, and serving them for distribution.

### 12.1 Package Building

The primary command for building packages is `roboctl package build`:

```bash
# Build package from current directory
roboctl package build

# Build package from specific directory
roboctl package build /path/to/package

# Build for specific architecture
roboctl package build --arch arm64

# Build for specific OS/architecture combination
roboctl package build --arch windows/amd64
roboctl package build --arch darwin/arm64
roboctl package build --arch linux/amd64

# Use custom config file
roboctl package build --file custom-config.json

# Skip build process (package existing binaries)
roboctl package build --no-build
```

**Build Process Overview:**
1. Reads `config.json` for package metadata and build scripts
2. Validates package version (must be valid SemVer)
3. Executes platform-specific build scripts from `config.json`
4. Generates `.pspec` specification file by running the binary with `-s` flag
5. Compresses all files into a `.tgz` archive
6. Names the output as `{namespace}-{version}-{os}-{arch}.tgz`

### 12.2 Repository Index Management

Create and maintain package repository indexes:

```bash
# Generate index.json from .tgz files in current directory
roboctl repo index

# Generate index from specific directory
roboctl repo index /path/to/packages

# Merge with existing index (preserve previous packages)
roboctl repo index --merge

# Generate index from subdirectory structure
roboctl repo index ./packages --merge
```

**Index Generation Process:**
- Scans directory for `.tgz` package files
- Extracts `config.json` from each package
- Creates `index.json` with package metadata
- Generates `index.sha256sum` for integrity verification
- Extracts and places `.pspec` files for Designer consumption
- Sorts versions using semantic versioning

**Index Structure:**
```json
{
  "generated": "2024-07-22T10:30:00-0000",
  "packages": {
    "namespace\\.package": {
      "name": "Package Name",
      "namespace": "Namespace.Package", 
      "version": "1.2.3",
      "versions": ["1.2.3", "1.2.2", "1.2.1"],
      "description": "Package description",
      "author": {"name": "Author", "email": "author@example.com"},
      "categories": ["Productivity"],
      "platforms": ["linux", "windows", "darwin"],
      "language": "Go",
      "path": "subfolder/path"
    }
  }
}
```

### 12.3 Repository Server

Serve packages over HTTP with CORS support:

```bash
# Serve on default address (127.0.0.1:8080)
roboctl repo serve

# Serve on custom IP and port
roboctl repo serve --ip 0.0.0.0 --port 3000

# Serve on localhost with custom port
roboctl repo serve --port 8888
```

The server serves static files from the current directory, enabling:
- Package downloads (`*.tgz` files)
- Index access (`index.json`, `index.sha256sum`)
- Specification files (`*.pspec`)

### 12.4 Integration with Robomotion Admin Console

**Adding a Repository:**
1. In Admin Console, navigate to **Repositories**
2. Click **Add Repository** button
3. Configure repository settings:
   - **Name**: `Local Repo` (or your preferred name)
   - **Description**: `Development local repo`
   - **URL**: `http://127.0.0.1:8080` (or your server address)
   - **Access Level**: `Everyone`

**Repository Workflow:**
```bash
# 1. Build your packages
cd /path/to/package1
roboctl package build
cd /path/to/package2 
roboctl package build

# 2. Organize packages in repository directory
mkdir my-repo
mv package1/*.tgz my-repo/
mv package2/*.tgz my-repo/

# 3. Generate repository index
cd my-repo
roboctl repo index

# 4. Serve repository
roboctl repo serve --ip 0.0.0.0 --port 8080
```

### 12.5 Advanced Repository Management

**Multi-Platform Building:**
```bash
# Build for all supported platforms
roboctl package build --arch linux/amd64
roboctl package build --arch windows/amd64  
roboctl package build --arch darwin/amd64
roboctl package build --arch darwin/arm64
```

**Repository Organization:**
```
repository/
├── index.json              # Generated package index
├── index.sha256sum         # Integrity checksum
├── package1-1.0.0-linux-amd64.tgz
├── package1-1.0.0-windows-amd64.tgz
├── package1-1.0.0-darwin-amd64.tgz
├── package1-1.0.0.pspec    # Designer specification
├── package2-2.1.0-linux-amd64.tgz
└── package2-2.1.0.pspec
```

**Incremental Updates:**
- Use `--merge` flag to preserve existing packages when adding new ones
- Repository server automatically handles version ordering (newest first)
- Empty `.tgz` files are automatically removed from index

### 12.6 CI/CD Integration

**Typical CI Pipeline:**
```yaml
# .github/workflows/build-packages.yml
- name: Build Package
  run: |
    roboctl package build --arch linux/amd64
    roboctl package build --arch windows/amd64
    roboctl package build --arch darwin/amd64

- name: Update Repository
  run: |
    cp *.tgz /repo/packages/
    cd /repo/packages
    roboctl repo index --merge

- name: Deploy to S3
  run: |
    aws s3 sync /repo/packages s3://my-package-repo/
```

---

## 13. Publish to a Repository

1. Make sure `config.json` is committed & version bumped.
2. Login (`roboctl login`).
3. Run `roboctl package build --arch amd64` for every platform you want.
4. Upload to your private or public repository (see `roboctl repo index` / `serve`).

Robomotion Cloud customers usually let the CI pipeline push artefacts directly to an S3-compatible bucket served by `roboctl repo serve`.

---

## 14. Development Troubleshooting & Workflow

### 14.1 Package Caching Issues

**Problem**: After updating and building your package, the code seems to be running an old version despite successful build.

**Root Cause**: Robomotion robots cache downloaded packages locally. If you don't increment the version in `config.json`, the same package version is created, and robots won't re-download it.

**Package Cache Locations**:
- **Linux/macOS**: `~/.config/robomotion/packages/bin/Robomotion/{PACKAGE_NAME}/{PACKAGE_VERSION}`
- **Windows**: `%LOCALAPPDATA%/Robomotion/packages/bin/Robomotion/{PACKAGE_NAME}/{PACKAGE_VERSION}`

**Solutions**:

#### Option 1: Clear Package Cache (Quick Fix)
```bash
# Linux/macOS - Remove specific package version
rm -rf ~/.config/robomotion/packages/bin/Robomotion/YourPackage/1.0.0

# Windows - Remove specific package version  
rmdir /s "%LOCALAPPDATA%\Robomotion\packages\bin\Robomotion\YourPackage\1.0.0"

# Or clear entire package cache
rm -rf ~/.config/robomotion/packages/bin/Robomotion/YourPackage
```

#### Option 2: Use Attach Mode (Recommended for Development)
This is the **most effective development approach** for **logic changes only**:

```bash
# 1. Build your package
go build -o dist/my-package

# 2. Make sure you have a connected robot and no flows are running
# 3. Run with attach flag
./dist/my-package -a
```

**Attach Mode Benefits**:
- Connects directly to running robot via gRPC
- Bypasses package cache entirely
- Real-time `log.Printf()` output visible in stdout
- No need to increment versions during development
- Robot prefers attached plugin over cached versions

**⚠️ Critical Limitation: Attach Mode Only Works for Logic Changes**

Attach mode **DOES NOT** work if you've made any of these changes:
- **Added new nodes** to your package
- **Modified node properties** (struct fields with `spec` tags)
- **Updated existing node properties** (title, type, description, etc.)
- **Added/removed/changed** `InVariable`, `OutVariable`, `OptVariable`, or `Credential` fields

**Why This Limitation Exists**:
1. Flow Designer downloads `.pspec` files from repository for drag-and-drop functionality
2. Node properties are displayed based on cached `.pspec` file content
3. Attach mode only affects runtime execution, not the Designer's metadata

**Attach Mode Requirements**:
- Must have a connected robot
- Stop any running flows that use your package
- Ensure no other instance of your package is running
- **Only use for business logic changes** (no node structure changes)

### 14.2 Node/Property Updates: Full Repository Workflow

When you've made **structural changes** to your nodes (new nodes, property changes), you must follow the complete workflow:

```bash
# 1. Update your code with new nodes or properties
# 2. Build package with roboctl (generates new .pspec)
roboctl package build

# 3. Update repository index (includes new .pspec files)  
roboctl repo index

# 4. Restart repository server (serves updated index and .pspec)
roboctl repo serve

# 5. Clear package cache to force re-download
rm -rf ~/.config/robomotion/packages/bin/Robomotion/YourPackage

# 6. Refresh Flow Designer (Ctrl+F5 or hard refresh)
# This downloads the updated .pspec file

# 7. Now you can use the updated nodes with new properties
```

**What Each Step Does**:
- **roboctl package build**: Creates new `.tgz` and regenerates `.pspec` file
- **roboctl repo index**: Updates `index.json` and extracts `.pspec` to repository root  
- **roboctl repo serve**: Makes updated `.pspec` file available via HTTP
- **Clear cache**: Forces robot to re-download package binaries
- **Designer refresh**: Downloads updated `.pspec` for node metadata

**When to Use Full Workflow vs Attach Mode**:

| Change Type | Use Attach Mode `-a` | Use Full Workflow |
|-------------|---------------------|-------------------|
| Bug fixes in `OnMessage()` logic | ✅ Yes | ❌ No |
| Algorithm improvements | ✅ Yes | ❌ No |
| Error handling updates | ✅ Yes | ❌ No |  
| `log.Printf()` additions | ✅ Yes | ❌ No |
| External API call changes | ✅ Yes | ❌ No |
| **New nodes added** | ❌ No | ✅ Required |
| **Property title/description changes** | ❌ No | ✅ Required |
| **New InVariable/OutVariable fields** | ❌ No | ✅ Required |
| **Spec tag modifications** | ❌ No | ✅ Required |
| **Node icon/color changes** | ❌ No | ✅ Required |

### 14.3 Attach Mode Deep Dive

**How Attach Mode Works**:
1. Your binary starts a gRPC server on a random port
2. Discovers local robot using network scanning (`debug/common.go`)
3. Registers with robot's debug service
4. Robot routes your package's node executions to the attached process

**Typical Attach Workflow**:
```bash
# Terminal 1: Start attached plugin
cd my-package
go build -o dist/my-package
./dist/my-package -a

# Output: "Attached to localhost:12345"

# Terminal 2: Monitor logs in real-time
# Your log.Printf statements appear here during flow execution
```

**Debug Logging Example**:
```go
func (n *MyNode) OnMessage(ctx message.Context) error {
    log.Printf("Processing message for node %s", n.GUID)
    
    input, err := n.InData.Get(ctx)
    if err != nil {
        log.Printf("Error getting input: %v", err)
        return err
    }
    
    log.Printf("Input value: %+v", input)
    
    // Your business logic here
    result := processData(input)
    
    log.Printf("Processed result: %+v", result)
    n.OutResult.Set(ctx, result)
    
    return nil
}
```

### 14.4 Common Development Issues

| Issue | Symptom | Solution |
|-------|---------|----------|
| **Stale Package Cache** | Code changes not reflected | Use attach mode or clear cache |
| **Multiple Plugin Instances** | "plugin already attached" error | Stop previous instances, check processes |
| **Robot Not Found** | "timeout: plugin listener is nil" | Ensure robot is connected and running |
| **Flow Still Running** | Attach fails or inconsistent behavior | Stop all flows using your package |
| **Port Conflicts** | gRPC connection errors | Check for conflicting processes on ports |
| **Missing Logs** | No debug output visible | Use attach mode, ensure `log.Printf` statements exist |

### 14.5 Development Workflow Decision Tree

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

**Quick Reference**:
- **Logic changes only** → Use `-a` attach mode
- **Any structural changes** → Use full workflow
- **When in doubt** → Use full workflow (safer but slower)

### 14.6 Debugging Best Practices

**Development Workflow**:
```bash
# 1. Initial development setup
roboctl package create --name "MyPackage" --namespace "Dev.MyPackage"
cd mypackage

# 2. Development cycle (repeat as needed)
# - Make code changes
# - Build: go build -o dist/mypackage
# - Test: ./dist/mypackage -a
# - Run flows in Designer
# - View logs in terminal

# 3. When ready for testing with real package system
# - Update version in config.json
# - Build: roboctl package build  
# - Clear cache if needed
# - Test with actual package deployment
```

**Logging Strategies**:
```go
// Use structured logging for complex debugging
log.Printf("[%s] OnMessage started - GUID: %s", time.Now().Format("15:04:05"), n.GUID)
log.Printf("[DEBUG] Input validation - value: %+v, type: %T", input, input)
log.Printf("[ERROR] Failed to process: %v", err)
log.Printf("[SUCCESS] Output generated - size: %d bytes", len(output))
```

**Error Handling for Debug**:
```go
func (n *MyNode) OnMessage(ctx message.Context) error {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("[PANIC] Node %s panic recovered: %v", n.GUID, r)
        }
    }()
    
    log.Printf("[START] %s processing message", n.Name)
    
    // Your logic here with detailed logging
    
    log.Printf("[END] %s completed successfully", n.Name)
    return nil
}
```

### 14.7 Network and Connection Issues

**Robot Discovery Problems**:
- Ensure robot and development machine are on same network
- Check firewall settings allow gRPC traffic
- Verify robot is in "Connected" status in Admin Console

**Attach Timeout Issues**:
- Default timeout is 30 seconds (`debug/attach.go`)
- If robot is slow to respond, restart robot service
- Check for network connectivity issues

**Multiple Robot Environments**:
- Attach connects to first discovered robot
- Use specific network interfaces if multiple robots present
- Consider using different development environments

---

## 15. Troubleshooting Checklist

| Symptom | Explanation |
|---------|-------------|
| `timeout: plugin listener is nil` | You launched the binary but never called `runtime.Start()` |
| "node handler not found" | Your node is not registered in `main.go` or the GUID differs from Designer UI |
| Variables always empty | Check `scope` and `messageScope` flags in the spec tag |
| Designer port names missing | Port field lacks `direction` / `position` tags |
| Empty *pspec* file | You forgot to set the struct tag on the embedded `runtime.Node` |

---

## 16. Further Reading & Code Dive

* [`robomotion-go/runtime`](../runtime) – SDK implementation (worth skimming)
* [`robomotion-chat-assistant/v1`](https://github.com/robomotionio/robomotion-chat-assistant) – extensive real-world nodes
* Hashicorp *go-plugin* – the underlying IPC transport

Happy automating! 💫

## ⚠️  Commas inside `spec:` values

The parser that converts the `spec:"…"` string into key/value pairs is extremely simple – it **splits on every comma** (`strings.Split(spec, ",")`, see `runtime/spec.go`).

```go
kvs := strings.Split(spec, ",") // ← no escaping supported
```

That means if you write:

```go
runtime.Node `spec:"id=Acme.My,name=My,description=Hello, world"`
```

the text after the comma – ` world` – is treated as a **new** key without a value, breaking the entire spec.

### Work-arounds

1. **Replace the comma with another character** – Designers usually accept a plain semicolon (`;`) or an em-dash (`—`).
2. **HTML-escape** it – `&#44;` is decoded by most browsers/UI frameworks so:
   ```go
   description=Hello&#44; world
   ```
3. **Custom parsing** – If you really need commas everywhere, consider extending `parseSpec()` to support escaped commas (`\,`) and submit a PR.

> The guide therefore uses **semicolon** in examples where a comma would normally appear in prose.
