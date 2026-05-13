package lmo

import (
	"strconv"
	"strings"
	"testing"
)

// TestResolve_SingularRecursive_DirectAccess_DoubleNested pins the gap in
// Store.Resolve (the path-based singular variant). A payload field that's a
// BlobRef envelope whose blob content is ITSELF another BlobRef envelope
// (double-nested) was returned unchanged pre-fix. Post-fix the singular
// Resolve fully unwraps it.
//
// Same pathology as the ResolveAll recursive fix shipped in this PR — but
// the singular Resolve path is used for var binding lookups
// (varhelper.go / SetRaw with key) and was a separate call site.
func TestResolve_SingularRecursive_DirectAccess_DoubleNested(t *testing.T) {
	s, _ := newTestStore(t)

	realData := []byte(`"hello from the deepest blob"`)
	realRef, err := s.PutBlob(realData)
	if err != nil {
		t.Fatalf("PutBlob real: %v", err)
	}
	realEnvelope := `{"__ref":"` + realRef + `","__magic":20260301,` +
		`"__size":` + strconv.Itoa(len(realData)) + `,"__path":"test/flow","__type":"string"}`

	wrapperRef, err := s.PutBlob([]byte(realEnvelope))
	if err != nil {
		t.Fatalf("PutBlob wrapper: %v", err)
	}
	wrapperEnvelope := `{"__ref":"` + wrapperRef + `","__magic":20260301,` +
		`"__size":` + strconv.Itoa(len(realEnvelope)) + `,"__path":"test/flow","__type":"string"}`

	payload := []byte(`{"key":` + wrapperEnvelope + `}`)

	result, err := s.Resolve(payload, "key")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if IsBlobRef(result) {
		t.Fatalf("Resolve returned a BlobRef envelope instead of unwrapping it.\n"+
			"Got: %s\nExpected: \"hello from the deepest blob\"", result.Raw)
	}
	if result.String() != "hello from the deepest blob" {
		t.Fatalf("expected resolved content, got: %s (raw: %s)", result.String(), result.Raw)
	}
}

// TestResolve_SingularRecursive_PathWalk_DoubleNested covers the inner-BlobRef
// branch: walk hits a BlobRef mid-path, the path-remainder inside the
// resolved content is itself a BlobRef whose content is ANOTHER BlobRef.
// The existing `if IsBlobRef(inner)` branch resolves once but doesn't loop.
func TestResolve_SingularRecursive_PathWalk_DoubleNested(t *testing.T) {
	s, _ := newTestStore(t)

	realData := []byte(`"deep value"`)
	realRef, err := s.PutBlob(realData)
	if err != nil {
		t.Fatalf("PutBlob real: %v", err)
	}
	realEnvelope := `{"__ref":"` + realRef + `","__magic":20260301,` +
		`"__size":` + strconv.Itoa(len(realData)) + `,"__path":"test/flow","__type":"string"}`

	wrapperRef, err := s.PutBlob([]byte(realEnvelope))
	if err != nil {
		t.Fatalf("PutBlob wrapper: %v", err)
	}
	wrapperEnvelope := `{"__ref":"` + wrapperRef + `","__magic":20260301,` +
		`"__size":` + strconv.Itoa(len(realEnvelope)) + `,"__path":"test/flow","__type":"string"}`

	outerContent := []byte(`{"inner":` + wrapperEnvelope + `,"sibling":"keep"}`)
	outerRef, err := s.PutBlob(outerContent)
	if err != nil {
		t.Fatalf("PutBlob outer: %v", err)
	}
	outerEnvelope := `{"__ref":"` + outerRef + `","__magic":20260301,` +
		`"__size":` + strconv.Itoa(len(outerContent)) + `,"__path":"test/flow","__type":"object"}`

	payload := []byte(`{"container":` + outerEnvelope + `}`)

	result, err := s.Resolve(payload, "container.inner")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if IsBlobRef(result) {
		t.Fatalf("Resolve returned a BlobRef envelope instead of fully unwrapping the path-walk + nested case.\n"+
			"Got: %s\nExpected: \"deep value\"", result.Raw)
	}
	if result.String() != "deep value" {
		t.Fatalf("expected resolved content, got: %s (raw: %s)", result.String(), result.Raw)
	}
}

// TestResolve_SingularRecursive_NoOpForUnnested guards the common case.
func TestResolve_SingularRecursive_NoOpForUnnested(t *testing.T) {
	s, _ := newTestStore(t)

	payload := []byte(`{"name":"plain","count":42}`)
	result, err := s.Resolve(payload, "name")
	if err != nil {
		t.Fatalf("Resolve plain: %v", err)
	}
	if result.String() != "plain" {
		t.Fatalf("expected plain, got %s", result.String())
	}

	realData := []byte(`"single-level"`)
	realRef, err := s.PutBlob(realData)
	if err != nil {
		t.Fatalf("PutBlob: %v", err)
	}
	realEnvelope := `{"__ref":"` + realRef + `","__magic":20260301,` +
		`"__size":` + strconv.Itoa(len(realData)) + `,"__path":"test/flow","__type":"string"}`
	payload2 := []byte(`{"k":` + realEnvelope + `}`)

	result2, err := s.Resolve(payload2, "k")
	if err != nil {
		t.Fatalf("Resolve single-level: %v", err)
	}
	if result2.String() != "single-level" {
		t.Fatalf("expected single-level, got %s", result2.String())
	}
}

// TestResolve_SingularRecursive_DepthLimit_Pathological pins the defensive
// depth-limit guard against pathological chains.
func TestResolve_SingularRecursive_DepthLimit_Pathological(t *testing.T) {
	s, _ := newTestStore(t)

	const chainDepth = 64
	innerRef, err := s.PutBlob([]byte(`"bottom"`))
	if err != nil {
		t.Fatalf("PutBlob bottom: %v", err)
	}
	currentEnvelope := `{"__ref":"` + innerRef + `","__magic":20260301,` +
		`"__size":8,"__path":"test/flow","__type":"string"}`
	for i := 0; i < chainDepth; i++ {
		ref, err := s.PutBlob([]byte(currentEnvelope))
		if err != nil {
			t.Fatalf("PutBlob link %d: %v", i, err)
		}
		currentEnvelope = `{"__ref":"` + ref + `","__magic":20260301,` +
			`"__size":` + strconv.Itoa(len(currentEnvelope)) + `,"__path":"test/flow","__type":"string"}`
	}

	payload := []byte(`{"k":` + currentEnvelope + `}`)
	result, err := s.Resolve(payload, "k")
	// Chain depth 64 exceeds maxResolveDepth (32), so the loop MUST exhaust
	// and surface a depth-limit error. Asserts the FIX is present (full
	// unwrap or loud error) and rejects the silent-failure mode where a
	// missing fullyResolveRef would return the inner envelope unchanged.
	if err == nil {
		if IsBlobRef(result) {
			t.Fatalf("Resolve returned a BlobRef envelope without error — "+
				"fullyResolveRef appears to be missing.\nGot: %s", result.Raw)
		}
		if result.String() != "bottom" {
			t.Fatalf("expected full unwrap to 'bottom', got: %s", result.String())
		}
		return
	}
	if !strings.Contains(err.Error(), "depth") {
		t.Fatalf("expected depth-limit error containing 'depth', got: %v", err)
	}
}
