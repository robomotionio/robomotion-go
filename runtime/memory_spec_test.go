package runtime

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/robomotionio/robomotion-go/message"
)

// fxMemory declares an agent memory; the spec must carry `memory`.
type fxMemory struct {
	Node `spec:"id=Test.Memory,name=Test Memory,icon=,color=#000"`
}

func (n *fxMemory) OnCreate() error                   { return nil }
func (n *fxMemory) OnMessage(_ message.Context) error { return nil }
func (n *fxMemory) OnClose() error                    { return nil }
func (n *fxMemory) Memory() MemoryDecl {
	return MemoryDecl{Layout: "v1", Roots: []string{"skills/learned", "skills/reviewed", "notes", "usage"}, Sync: true}
}

func TestSpec_MemoryProviderEmitsMemory(t *testing.T) {
	pspec := captureSpec(t, &fxMemory{})
	node := nodeByID(t, pspec, "Test.Memory")
	mem, ok := node["memory"].(map[string]interface{})
	if !ok {
		t.Fatalf("memory missing: %v", node)
	}
	if mem["layout"] != "v1" || mem["sync"] != true {
		t.Fatalf("memory = %v", mem)
	}
	roots, _ := mem["roots"].([]interface{})
	if !reflect.DeepEqual(roots, []interface{}{"skills/learned", "skills/reviewed", "notes", "usage"}) {
		t.Fatalf("roots = %v", roots)
	}
}

// A node without Memory() must produce exactly the pspec it produced before:
// no `memory` key at all, and nothing else moved.
func TestSpec_WithoutMemoryIsByteIdentical(t *testing.T) {
	pspec := captureSpec(t, &fxPlain{})
	node := nodeByID(t, pspec, "Test.Plain")
	if _, present := node["memory"]; present {
		t.Fatalf("a node without Memory() emitted memory: %v", node["memory"])
	}
	raw, _ := json.Marshal(node)
	if got := string(raw); containsKey(got, `"memory"`) {
		t.Fatalf("memory key leaked into %s", got)
	}
}

func containsKey(s, key string) bool {
	return len(s) > 0 && len(key) > 0 && (indexOf(s, key) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestMemoryRepoName(t *testing.T) {
	if MemoryRepoName("flow-1", "abc123") != "agent-flow-1-abc123" {
		t.Fatal(MemoryRepoName("flow-1", "abc123"))
	}
}

func TestMemoryRemoteWithoutRuntimeIsNoRemote(t *testing.T) {
	if _, err := MemoryRemote("flow", "abc123"); err == nil {
		t.Fatal("expected no remote without an initialised runtime")
	}
	if _, err := MemoryRemote("", "abc123"); err == nil {
		t.Fatal("expected no remote without a flow id")
	}
}
