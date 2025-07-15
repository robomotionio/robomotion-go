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

All three wrappers expose `.Get(ctx)` (for `In`/`Opt`) and `.Set(ctx,val)` (`Out`).

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

## 7. Accessing the Runtime Helper

In `Init` the SDK injects an instance that implements `runtime.RuntimeHelper`:

```go
func (n *MyNode) Init(r runtime.RuntimeHelper) error {
    n.Runtime = r // store if you need it later
    return nil
}
```

However, the **simpler way** is to call the global functions defined in `runtime/event.go`, `credential.go`, … because they proxy the helper automatically (they panic if the node is not running inside a robot).

Example: Emit an output on port **2**

```go
if err := runtime.EmitOutput(n.GUID, data, 2); err != nil {
    return err
}
```

---

## 8. Large Message Objects (LMO) – sending >64 KB

Robomotion runners optimise bandwidth. If you need to output a large payload (>64 KB) pack it first:

```go
packed, _ := runtime.PackMessageBytes(rawJSON)
ctx.SetRaw(packed, runtime.WithPack())
```
Unpacking is symmetrical on the receiving node (`runtime.WithUnpack`). See `runtime/lmo.go` for internals.

---

## 9. Custom Ports

If the default left-side input / right-side output layout does not fit your UX you can declare **named ports**:

```go
type Uploader struct {
    runtime.Node `spec:"id=Acme.Uploader,name=Uploader,icon=mdiUpload,color=#9b59b6,inputs=0,outputs=0"`

    // Port is simply an alias for []string – it will **not** be visible in Go code
    Files runtime.Port `direction="input"  position="left"  name="files"  icon="mdiFile"  filters="files"`
    Done  runtime.Port `direction="output" position="right" name="done"  icon="mdiCheck" color="#2ecc71"`
}
```
The code generator (`generateSpecFile()`) serialises those tags into `customPorts` so Designer knows where to draw the port.

---

## 10. Registering & Starting

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

## 11. Build & Run locally

```bash
# Build the binary for your host OS
roboctl package build

# Run it inside a Robot (assumes you are developing *inside* a robot)
./dist/my-package -a   # "attach" – streams logs & debugging to Designer
```

### 11.1 Generate *pspec* only
Sometimes you just want to refresh the Designer specification (JSON) without rebuilding everything:

```bash
./dist/my-package -s   # outputs <namespace>-<version>.spec.json next to the binary
```

The file is created by `runtime.generateSpecFile()` and automatically picked up by roboctl when packaging.

### 11.2 Cross-compiling & multi-arch builds

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

## 15. Debugging & Logs (attach / detach)

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

## 16. Anatomy of the generated *.spec.json* file

The spec file (sometimes called *pspec*) is what the Designer consumes to render nodes, editors and ports.

* Generated by `generateSpecFile()` (invoked when the binary starts with **`-s`**).  
* File name: `<namespace>-<version>.spec.json` (written to *stdout* – `roboctl` captures and stores it next to the binary).  
* Top-level keys:
  * `name` – package display name
  * `version`
  * `nodes[]` – array with everything described in section 5 (ID, icon, colors, `properties[]`, `customPorts[]` …)

Open it once and you’ll immediately see where your tag information ends up – this is invaluable when something doesn’t look right in the Designer.

---

## 12. Publish to a Repository

1. Make sure `config.json` is committed & version bumped.
2. Login (`roboctl login`).
3. Run `roboctl package build --arch amd64` for every platform you want.
4. Upload to your private or public repository (see `roboctl repo index` / `serve`).

Robomotion Cloud customers usually let the CI pipeline push artefacts directly to an S3-compatible bucket served by `roboctl repo serve`.

---

## 13. Troubleshooting Checklist

| Symptom | Explanation |
|---------|-------------|
| `timeout: plugin listener is nil` | You launched the binary but never called `runtime.Start()` |
| "node handler not found" | Your node is not registered in `main.go` or the GUID differs from Designer UI |
| Variables always empty | Check `scope` and `messageScope` flags in the spec tag |
| Designer port names missing | Port field lacks `direction` / `position` tags |
| Empty *pspec* file | You forgot to set the struct tag on the embedded `runtime.Node` |

---

## 14. Further Reading & Code Dive

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
