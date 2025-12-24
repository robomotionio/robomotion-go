# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

robomotion-go is the official Go SDK for building Robomotion packages. It provides a runtime framework for creating plugin-based automation nodes that communicate with the Robomotion platform via gRPC.

## Core Architecture

### Plugin Architecture
- Uses HashiCorp's go-plugin framework for IPC between host and plugins
- Plugins communicate via gRPC with protobuf-defined interfaces
- Each node is a Go struct that embeds `runtime.Node` and implements lifecycle methods

### Key Components

**Runtime Package (`runtime/`)**
- `interface.go` - Core RuntimeHelper interface and plugin definitions
- `node.go` - Base Node struct with common properties (GUID, delays, error handling)
- `factory.go` - Node factory pattern for dynamic node creation
- `handler.go` - Message handling and routing with tool interceptor wrapping
- `grpc.go` - gRPC server/client implementation
- `variable.go` - Strongly-typed variable system (InVariable, OutVariable, OptVariable)
- `spec.go` - Node specification parsing from struct tags
- `tool.go` / `tool_interceptor.go` - AI tool support for cross-language agent integration
- `oauth.go` - OAuth2 dialog support with browser-based authorization flow

**Message System**
- `message/context.go` - Message context for data flow between nodes
- `runtime/message.go` - Message handling utilities
- `runtime/lmo.go` - Large Message Object support for payloads >256KB

**Debug Package (`debug/`)**
- `attach.go/detach.go` - Development-time debugging support
- Platform-specific network utilities for connection discovery

## Development Commands

### Building
```bash
# Build using roboctl (recommended)
roboctl package build

# Cross-compile for different platforms
roboctl package build --arch windows/amd64
roboctl package build --arch darwin/arm64

# Standard Go build
go build -o dist/package-name
```

### Running/Testing
```bash
# Run in attach mode for debugging (connects to local robot)
./dist/package-name -a

# Generate spec file only
./dist/package-name -s
```

### Dependencies
```bash
# Update dependencies
go mod tidy

# Vendor dependencies (if needed)
go mod vendor
```

### Testing
```bash
go test ./...
go test -v ./runtime
```

## Node Development Pattern

### Basic Node Structure
```go
type MyNode struct {
    runtime.Node `spec:"id=Namespace.MyNode,name=My Node,icon=mdiIcon,color=#3498db"`

    // Options (user-configurable in Designer)
    MyOption string `spec:"title=My Option,value=default,option"`

    // Inputs
    InData runtime.InVariable[string] `spec:"title=Input Data,type=string,scope=Message,name=data"`

    // Outputs
    OutResult runtime.OutVariable[string] `spec:"title=Result,type=string,scope=Message,name=result"`
}

// Required lifecycle methods
func (n *MyNode) OnCreate() error { return nil }
func (n *MyNode) OnMessage(ctx message.Context) error { return nil }
func (n *MyNode) OnClose() error { return nil }
```

### Node Registration
All nodes must be registered in `main.go`:
```go
func main() {
    runtime.RegisterNodes(
        &v1.MyNode{},
    )
    runtime.Start()
}
```

### AI Tool Support
Nodes can be exposed as AI tools for cross-language agent integration:
```go
type MyToolNode struct {
    runtime.Node `spec:"id=Namespace.MyTool,name=My Tool,icon=mdiRobot,color=#9b59b6"`
    runtime.Tool `tool:"name=my_tool,description=Does something useful"` // Marks as AI-callable

    // AI scope allows LLM to provide input values
    InQuery runtime.InVariable[string] `spec:"title=Query,type=string,scope=AI,name=query"`

    // Output is automatically collected for tool response
    OutResult runtime.OutVariable[string] `spec:"title=Result,type=string,scope=AI,name=result"`
}
```
The `ToolInterceptor` automatically wraps nodes with `runtime.Tool` and handles tool requests transparently.

## Key Conventions

### Spec Tags
- Node identification: `id=Namespace.NodeName` (must be unique)
- UI properties: `name`, `icon`, `color`, `inputs`, `outputs`
- Variable properties: `title`, `type`, `scope`, `name`, `messageScope`, `jsScope`, `customScope`
- Field types: `option` for user configuration
- Use Unicode comma (，U+FF0C) in descriptions to avoid parsing issues

### Variable Types and Scopes
| Type | Purpose | Access |
|------|---------|--------|
| `InVariable[T]` | Required input | `.Get(ctx)` returns `(T, error)` |
| `OptVariable[T]` | Optional input | `.Get(ctx)` returns `(T, error)` |
| `OutVariable[T]` | Output | `.Set(ctx, value)` |
| `Credential` | Vault item | `.Get(ctx)` returns `(map[string]interface{}, error)` |

**Scopes:**
- `Message` - Data from previous nodes in flow (message context)
- `Custom` - User-configured constants in Designer
- `JS` / `CS` - Dynamically controlled by expressions
- `AI` - Parameters provided by LLM agents (works like Message scope)

### Error Handling
- Return errors from lifecycle methods to stop flow execution
- Use `ContinueOnError: true` on Node to continue on errors
- Emit errors via `runtime.EmitError()`

## File Structure Conventions
```
package-root/
├── go.mod              # Module definition
├── config.json         # Package metadata and build scripts
├── main.go             # Node registration and entry point
├── v1/                 # Versioned node implementations
│   ├── node1.go
│   └── node2.go
├── icon.png            # Package icon
└── dist/               # Build outputs
```

## Common Patterns

### Accessing Runtime Services
```go
// Use global functions (preferred)
runtime.EmitOutput(n.GUID, data, portNumber)
runtime.GetVaultItem(scope, key)

// Or store RuntimeHelper from Init()
func (n *MyNode) Init(r runtime.RuntimeHelper) error {
    n.runtime = r
    return nil
}
```

### Large Data Handling
```go
// For payloads >256KB, LMO is handled automatically
// Manual packing if needed:
packed, _ := runtime.PackMessageBytes(largeData)
ctx.SetRaw(packed, runtime.WithPack())
```

### Custom Ports
```go
// Define named ports with directions
Files runtime.Port `direction="input" position="left" name="files"`
Done  runtime.Port `direction="output" position="right" name="done"`
```

### OAuth2 Authentication
```go
import "golang.org/x/oauth2"

func (n *MyNode) OnCreate() error {
    config := &oauth2.Config{
        ClientID:     "your-client-id",
        ClientSecret: "your-client-secret",
        Scopes:       []string{"scope1", "scope2"},
        Endpoint:     provider.Endpoint,
        RedirectURL:  runtime.OAuth2RedirectURL, // Fixed: http://localhost:9876/oauth2/callback
    }

    code, err := runtime.OpenOAuthDialog(config)
    if err != nil {
        return err
    }

    token, err := config.Exchange(context.Background(), code)
    // Store token for use in OnMessage...
    return nil
}
```
The OAuth dialog opens a browser, waits for authorization (up to 5 minutes), and handles port contention with automatic retry.

## Dependencies

Key external dependencies:
- `github.com/robomotionio/go-plugin` - Plugin framework
- `google.golang.org/grpc` - gRPC communication
- `google.golang.org/protobuf` - Protocol buffers
- `github.com/sirupsen/logrus` - Logging
- `github.com/tidwall/gjson/sjson` - JSON manipulation
- `golang.org/x/oauth2` - OAuth2 support

## Debugging

Use the attach mode for development:
1. Build: `roboctl package build`
2. Run: `./dist/package-name -a`
3. Execute flows in Designer - logs appear in console
4. Use standard Go logging (`log.Printf`) for debug output
