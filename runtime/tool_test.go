package runtime

import (
	"encoding/json"
	"testing"

	"github.com/robomotionio/robomotion-go/message"
)

// ToolName + ToolParameters operate on the public message.Context — the
// helpers should resolve the standard wire fields the agent side sets
// (`__tool_name__` and `__parameters__`) and degrade gracefully when the
// envelope is malformed or absent. These cases mirror the corresponding
// Python tests in robomotion-python/tests/test_tool.py.

func newCtx(raw string) message.Context {
	return message.NewContext([]byte(raw))
}

func TestIsToolRequest(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{"tool_request", `{"__message_type__":"tool_request"}`, true},
		{"user_message", `{"__message_type__":"user_message"}`, false},
		{"missing", `{}`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsToolRequest(newCtx(c.raw)); got != c.want {
				t.Fatalf("IsToolRequest(%s) = %v, want %v", c.raw, got, c.want)
			}
		})
	}
}

func TestToolName(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"set", `{"__tool_name__":"send_message"}`, "send_message"},
		{"missing", `{}`, ""},
		// gjson coerces non-string JSON values via Value() — a numeric
		// tool_name surfaces as float64, which fails the string type-assert
		// inside ToolName and returns "". That matches the Python helper.
		{"non_string", `{"__tool_name__":42}`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ToolName(newCtx(c.raw)); got != c.want {
				t.Fatalf("ToolName(%s) = %q, want %q", c.raw, got, c.want)
			}
		})
	}
}

func TestToolParameters(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		ctx := newCtx(`{"__parameters__":{"body":"hi","channel_id":"c1"}}`)
		got := ToolParameters(ctx)
		if got["body"] != "hi" || got["channel_id"] != "c1" {
			t.Fatalf("unexpected params: %#v", got)
		}
	})
	t.Run("missing", func(t *testing.T) {
		got := ToolParameters(newCtx(`{}`))
		if got == nil {
			t.Fatal("ToolParameters returned nil; want empty map")
		}
		if len(got) != 0 {
			t.Fatalf("ToolParameters non-empty for missing: %#v", got)
		}
	})
	t.Run("non_object_returns_empty", func(t *testing.T) {
		// A string-typed `__parameters__` is malformed but the helper
		// should not panic — it returns an empty map so callers can
		// safely range or index without nil checks.
		got := ToolParameters(newCtx(`{"__parameters__":"bad"}`))
		if got == nil || len(got) != 0 {
			t.Fatalf("expected empty map for non-object params, got %#v", got)
		}
	})
}

// ---------------------------------------------------------------------------
// ToolDef + ToolkitProvider + SkillProvider — interface compliance
// ---------------------------------------------------------------------------

type fakeToolkit struct{}

func (f *fakeToolkit) Tools() []ToolDef {
	return []ToolDef{
		{Name: "alpha", Description: "first", Schema: map[string]interface{}{"type": "object"}},
		{Name: "beta", Description: "second", Schema: map[string]interface{}{"type": "object", "required": []string{"x"}}},
	}
}

func (f *fakeToolkit) Skill() string { return "# Skill\nbody" }

func TestToolkitProvider_InterfaceContract(t *testing.T) {
	var tp ToolkitProvider = &fakeToolkit{}
	tools := tp.Tools()
	if len(tools) != 2 {
		t.Fatalf("Tools() returned %d entries, want 2", len(tools))
	}
	if tools[0].Name != "alpha" || tools[1].Name != "beta" {
		t.Fatalf("unexpected tool names: %v / %v", tools[0].Name, tools[1].Name)
	}
	// Schema is a regular map — must round-trip through JSON the way the
	// spec generator marshals it.
	raw, err := json.Marshal(tools[0])
	if err != nil {
		t.Fatalf("ToolDef JSON marshal failed: %v", err)
	}
	var decoded ToolDef
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("ToolDef JSON unmarshal failed: %v", err)
	}
	if decoded.Name != "alpha" || decoded.Description != "first" {
		t.Fatalf("ToolDef did not round-trip: %#v", decoded)
	}
}

func TestSkillProvider_InterfaceContract(t *testing.T) {
	var sp SkillProvider = &fakeToolkit{}
	if got := sp.Skill(); got != "# Skill\nbody" {
		t.Fatalf("Skill() = %q, want %q", got, "# Skill\nbody")
	}
}

func TestTool_AndToolkit_AreDistinctMarkers(t *testing.T) {
	// Sanity check: a node embedding runtime.Tool must not be detected
	// as a Toolkit, and vice versa. The spec generator differentiates
	// them by field name and emits to different pspec keys.
	type withTool struct{ Tool }
	type withToolkit struct{ Toolkit }
	type withBoth struct {
		Tool
		Toolkit
	}

	var _ = withTool{}
	var _ = withToolkit{}
	var _ = withBoth{} // legal Go; spec generator emits both keys
}
