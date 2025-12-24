// Package testing provides utilities for unit testing Robomotion nodes.
//
// This package allows testing of node OnMessage methods without requiring
// the full Robomotion runtime, GRPC, or NATS infrastructure. It provides
// mock implementations of the message.Context interface and helpers for
// configuring node variables.
//
// # Basic Usage
//
// There are two main ways to use this package:
//
// 1. Using the Harness API for full control:
//
//	func TestMyNode(t *testing.T) {
//	    node := &mypackage.MyNode{}
//
//	    h := testing.NewHarness(node).
//	        ConfigureInVariable(&node.InData, "Message", "data").
//	        ConfigureOutVariable(&node.OutResult, "Message", "result").
//	        WithInput("data", "input value")
//
//	    err := h.Run()
//	    if err != nil {
//	        t.Fatalf("Expected no error, got: %v", err)
//	    }
//
//	    result := h.GetOutput("result")
//	    // ... assertions
//	}
//
// 2. Using the Quick API for simpler tests (auto-configures from spec tags):
//
//	func TestMyNode_Quick(t *testing.T) {
//	    node := &mypackage.MyNode{}
//
//	    q := testing.NewQuick(node).
//	        SetInput("data", "input value").
//	        SetCustom("InOption", "option value")
//
//	    err := q.Run()
//	    // ... assertions using q.Output("result")
//	}
//
// # Variable Configuration
//
// Robomotion nodes use InVariable, OutVariable, and OptVariable types
// to read from and write to the message context. These variables have
// a Scope and Name that determine how they access data:
//
//   - Message scope: Reads/writes from the message context using the Name as a JSON path
//   - Custom scope: The Name field contains the actual value (for user-provided constants)
//   - Other scopes (Global, Flow, etc.): Require runtime support (not available in tests)
//
// For testing, you typically configure variables to use Message scope and
// then set the corresponding values in the mock context:
//
//	h.ConfigureInVariable(&node.InData, "Message", "data")
//	h.WithInput("data", "the input value")
//
// For Custom scope variables (options/parameters), use ConfigureCustomInput:
//
//	h.ConfigureCustomInput(&node.InKey, "propertyName")
//
// # MockContext
//
// MockContext implements message.Context and stores all data in memory.
// It uses gjson/sjson for JSON path operations, matching the behavior
// of the real Robomotion context.
//
// You can create a MockContext directly if needed:
//
//	ctx := testing.NewMockContext(map[string]interface{}{
//	    "key1": "value1",
//	    "nested": map[string]interface{}{"deep": "value"},
//	})
//
//	value := ctx.Get("nested.deep")  // Returns "value"
//
// # Testing Patterns
//
// Testing error conditions:
//
//	func TestMyNode_InvalidInput(t *testing.T) {
//	    node := &mypackage.MyNode{}
//	    h := testing.NewHarness(node).
//	        ConfigureInVariable(&node.InData, "Message", "data")
//	    // Note: not setting "data" input
//
//	    err := h.Run()
//	    if err == nil {
//	        t.Fatal("Expected error for missing input")
//	    }
//	}
//
// Testing workflows (multiple nodes):
//
//	func TestWorkflow(t *testing.T) {
//	    // Step 1
//	    node1 := &package.Node1{}
//	    h1 := testing.NewHarness(node1).
//	        ConfigureInVariable(&node1.InData, "Message", "input").
//	        ConfigureOutVariable(&node1.OutData, "Message", "output").
//	        WithInput("input", "data")
//	    h1.Run()
//
//	    // Pass output to step 2
//	    intermediate := h1.GetOutput("output")
//
//	    node2 := &package.Node2{}
//	    h2 := testing.NewHarness(node2).
//	        ConfigureInVariable(&node2.InData, "Message", "input").
//	        WithInput("input", intermediate)
//	    h2.Run()
//	}
//
// # Limitations
//
// This testing package has some limitations:
//
//   - Global, Flow, and other non-Message scopes require the actual runtime
//   - Credential access (runtime.Credential) is not supported
//   - LMO (Large Message Object) serialization is not fully supported
//   - Tool request/response flows require additional setup
//
// For integration tests that require these features, consider using the
// full Robomotion runtime in a test environment.
package testing
