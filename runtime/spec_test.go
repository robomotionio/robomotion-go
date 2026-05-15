package runtime

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/robomotionio/robomotion-go/message"
)

// These tests drive generateSpecFile against fixture node types and
// assert the JSON it prints. The fixtures cover the three spec-emission
// paths the toolkit work touches:
//
//   • runtime.Tool      → node.tool: {name, description}
//   • runtime.Toolkit + ToolkitProvider.Tools() → node.tools: [...]
//   • runtime.SkillProvider.Skill() → node.skill: "..."
//   • OptVariable[[]string] + enum  → multi-select shape with
//     items.enum/enumNames and ui:field=multiSelectCheckbox
//
// generateSpecFile prints to stdout. We pipe stdout through an os.Pipe
// to capture the output, restore the original FD after, and unmarshal
// the JSON to make structured assertions.

// ---------------------------------------------------------------------------
// Fixture node types
// ---------------------------------------------------------------------------

// fxToolkit exercises the multi-tool path. It embeds runtime.Toolkit and
// implements ToolkitProvider + SkillProvider so the spec generator emits
// `tools[]` and `skill`. The OptEnabled field uses OptVariable[[]string]
// + enum to drive the multi-select checkbox path.
type fxToolkit struct {
	Node    `spec:"id=Test.Toolkit,name=Test Toolkit,icon=,color=#000"`
	Toolkit `toolkit:""`

	OptEnabled runtime_OptVariableStringSlice `spec:"title=Enabled,type=array,scope=Custom,name=,customScope,enum=a|b|c,enumNames=A|B|C"`
}

// runtime_OptVariableStringSlice exists because the test fixture cannot
// inline `OptVariable[[]string]` as a struct field tag without pulling
// the generic instantiation into the type definition. Aliasing keeps
// the spec generator's reflection path happy (it inspects the type's
// name prefix via `isGenericType`).
type runtime_OptVariableStringSlice = OptVariable[[]string]

func (n *fxToolkit) OnCreate() error                     { return nil }
func (n *fxToolkit) OnMessage(_ message.Context) error   { return nil }
func (n *fxToolkit) OnClose() error                      { return nil }
func (n *fxToolkit) Tools() []ToolDef {
	return []ToolDef{
		{Name: "one", Description: "first", Schema: map[string]interface{}{"type": "object"}},
		{Name: "two", Description: "second", Schema: map[string]interface{}{
			"type":     "object",
			"required": []interface{}{"x"},
		}},
	}
}
func (n *fxToolkit) Skill() string { return "use this toolkit wisely" }

// fxSingleTool exercises the legacy single-Tool path so the test suite
// also pins the back-compat behavior — `tool: {name, description}` must
// still emit unchanged.
type fxSingleTool struct {
	Node `spec:"id=Test.SingleTool,name=Test Single,icon=,color=#000"`
	Tool `tool:"name=do_thing,description=does a thing"`
}

func (n *fxSingleTool) OnCreate() error                   { return nil }
func (n *fxSingleTool) OnMessage(_ message.Context) error { return nil }
func (n *fxSingleTool) OnClose() error                    { return nil }

// fxPlain has neither Tool nor Toolkit — its emitted spec must carry
// neither `tool` nor `tools` nor `skill`.
type fxPlain struct {
	Node `spec:"id=Test.Plain,name=Test Plain,icon=,color=#000"`
}

func (n *fxPlain) OnCreate() error                   { return nil }
func (n *fxPlain) OnMessage(_ message.Context) error { return nil }
func (n *fxPlain) OnClose() error                    { return nil }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// captureSpec registers `nodes` with the SDK's handler list, runs the
// spec generator, captures the printed JSON, and unmarshals it. The
// previous handlerList is restored on exit so concurrent test packages
// remain isolated.
func captureSpec(t *testing.T, nodes ...MessageHandler) map[string]interface{} {
	t.Helper()

	prev := handlerList
	t.Cleanup(func() { handlerList = prev })
	RegisterNodes(nodes...)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	doneCh := make(chan []byte, 1)
	go func() {
		buf, _ := io.ReadAll(r)
		doneCh <- buf
	}()

	generateSpecFile("test-plugin", "0.0.1")
	w.Close()
	raw := <-doneCh

	var pspec map[string]interface{}
	if err := json.Unmarshal(raw, &pspec); err != nil {
		t.Fatalf("spec JSON decode failed (output=%q): %v", string(raw), err)
	}
	return pspec
}

func nodeByID(t *testing.T, pspec map[string]interface{}, id string) map[string]interface{} {
	t.Helper()
	nodes, ok := pspec["nodes"].([]interface{})
	if !ok {
		t.Fatalf("pspec[nodes] missing or wrong type: %T", pspec["nodes"])
	}
	for _, n := range nodes {
		m, ok := n.(map[string]interface{})
		if !ok {
			continue
		}
		if m["id"] == id {
			return m
		}
	}
	t.Fatalf("node id=%q not found in pspec (got %d nodes)", id, len(nodes))
	return nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestSpec_ToolkitEmitsToolsAndSkill(t *testing.T) {
	pspec := captureSpec(t, &fxToolkit{})
	node := nodeByID(t, pspec, "Test.Toolkit")

	tools, ok := node["tools"].([]interface{})
	if !ok {
		t.Fatalf("node.tools missing or wrong type: %#v", node["tools"])
	}
	if len(tools) != 2 {
		t.Fatalf("node.tools has %d entries, want 2: %#v", len(tools), tools)
	}
	first := tools[0].(map[string]interface{})
	if first["name"] != "one" || first["description"] != "first" {
		t.Fatalf("tools[0] = %#v", first)
	}
	if _, ok := first["schema"].(map[string]interface{}); !ok {
		t.Fatalf("tools[0].schema not an object: %#v", first["schema"])
	}

	if skill, _ := node["skill"].(string); skill != "use this toolkit wisely" {
		t.Fatalf("node.skill = %q, want %q", skill, "use this toolkit wisely")
	}

	// Toolkit nodes must NOT also carry a `tool` (singular) entry — the
	// two keys are mutually exclusive by convention.
	if _, has := node["tool"]; has {
		t.Fatalf("toolkit node unexpectedly has `tool` key: %#v", node["tool"])
	}
}

func TestSpec_SingleToolUnchanged(t *testing.T) {
	// Back-compat regression: a node embedding runtime.Tool (not Toolkit)
	// must still emit `tool: {name, description}` and must not carry
	// `tools` or `skill` unless it also implements SkillProvider.
	pspec := captureSpec(t, &fxSingleTool{})
	node := nodeByID(t, pspec, "Test.SingleTool")

	tool, ok := node["tool"].(map[string]interface{})
	if !ok {
		t.Fatalf("node.tool missing or wrong type: %#v", node["tool"])
	}
	if tool["name"] != "do_thing" || tool["description"] != "does a thing" {
		t.Fatalf("node.tool = %#v", tool)
	}
	if _, has := node["tools"]; has {
		t.Fatalf("single-tool node unexpectedly has `tools` key: %#v", node["tools"])
	}
	if _, has := node["skill"]; has {
		t.Fatalf("single-tool node unexpectedly has `skill` key: %#v", node["skill"])
	}
}

func TestSpec_PlainNodeHasNeitherToolNorToolsNorSkill(t *testing.T) {
	pspec := captureSpec(t, &fxPlain{})
	node := nodeByID(t, pspec, "Test.Plain")
	for _, k := range []string{"tool", "tools", "skill"} {
		if _, has := node[k]; has {
			t.Fatalf("plain node unexpectedly carries key %q: %#v", k, node[k])
		}
	}
}

func TestSpec_OptVariableEnumEmitsMultiSelectCheckbox(t *testing.T) {
	// OptVariable[[]string] + enum must hit the new isVar+isEnum+isOptVar
	// branch in spec.go: the property's UISchema gets
	// `ui:field=multiSelectCheckbox`, the schema becomes a `type=array`
	// with `items.enum`/`items.enumNames`, and formData defaults to [].
	pspec := captureSpec(t, &fxToolkit{})
	node := nodeByID(t, pspec, "Test.Toolkit")

	properties, ok := node["properties"].([]interface{})
	if !ok || len(properties) == 0 {
		t.Fatalf("node.properties missing or empty: %#v", node["properties"])
	}

	// The OptVariable lives under the Options property group. Find the
	// group whose schema.title == "Options".
	var optsGroup map[string]interface{}
	for _, p := range properties {
		pm, _ := p.(map[string]interface{})
		schema, _ := pm["schema"].(map[string]interface{})
		if schema["title"] == "Options" {
			optsGroup = pm
			break
		}
	}
	if optsGroup == nil {
		t.Fatalf("Options property group not found: %#v", properties)
	}

	schemaProps, _ := optsGroup["schema"].(map[string]interface{})["properties"].(map[string]interface{})
	uiSchema, _ := optsGroup["uiSchema"].(map[string]interface{})
	formData, _ := optsGroup["formData"].(map[string]interface{})

	optEnabled, ok := schemaProps["optEnabled"].(map[string]interface{})
	if !ok {
		t.Fatalf("schema.properties.optEnabled missing: %#v", schemaProps)
	}
	if optEnabled["type"] != "array" {
		t.Fatalf("optEnabled.type = %v, want \"array\"", optEnabled["type"])
	}
	if mult, _ := optEnabled["multiple"].(bool); !mult {
		t.Fatalf("optEnabled.multiple = %v, want true", optEnabled["multiple"])
	}
	items, ok := optEnabled["items"].(map[string]interface{})
	if !ok {
		t.Fatalf("optEnabled.items missing: %#v", optEnabled)
	}
	enum, ok := items["enum"].([]interface{})
	if !ok || len(enum) != 3 {
		t.Fatalf("optEnabled.items.enum = %#v, want length 3", items["enum"])
	}
	got := make([]string, 0, len(enum))
	for _, v := range enum {
		got = append(got, v.(string))
	}
	if strings.Join(got, ",") != "a,b,c" {
		t.Fatalf("items.enum order = %v, want [a b c]", got)
	}

	if ui, _ := uiSchema["optEnabled"].(map[string]interface{}); ui["ui:field"] != "multiSelectCheckbox" {
		t.Fatalf("uiSchema.optEnabled.ui:field = %v, want \"multiSelectCheckbox\"", ui["ui:field"])
	}

	// formData should default to an empty array — when the user picks
	// nothing, the agent-side reads empty and exposes every tool.
	arr, ok := formData["optEnabled"].([]interface{})
	if !ok {
		t.Fatalf("formData.optEnabled missing or wrong type: %#v", formData["optEnabled"])
	}
	if len(arr) != 0 {
		t.Fatalf("formData.optEnabled = %v, want []", arr)
	}
}
