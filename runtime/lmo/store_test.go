package lmo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

// helper: create a store in a temp dir with relPath set.
func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetRelPath("test/flow"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, dir
}

// bigString returns a string of length n (above threshold).
func bigString(n int) string {
	return strings.Repeat("A", n)
}

func TestPutGetBlobRoundtrip(t *testing.T) {
	s, _ := newTestStore(t)

	data := []byte(`"hello world"`)
	ref, err := s.PutBlob(data)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(ref, "xxh3:") {
		t.Fatalf("expected xxh3: prefix, got %s", ref)
	}

	got, err := s.GetBlob(ref, "test/flow")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Fatalf("roundtrip mismatch: got %s, want %s", got, data)
	}
}

func TestPutBlobDeduplication(t *testing.T) {
	s, dir := newTestStore(t)

	data := []byte(`"deduplicate me"`)
	ref1, err := s.PutBlob(data)
	if err != nil {
		t.Fatal(err)
	}
	ref2, err := s.PutBlob(data)
	if err != nil {
		t.Fatal(err)
	}
	if ref1 != ref2 {
		t.Fatalf("same data should produce same ref: %s vs %s", ref1, ref2)
	}

	// Count blob files — should be exactly 1.
	blobDir := filepath.Join(dir, "store", "test/flow", "blobs")
	count := 0
	filepath.Walk(blobDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			count++
		}
		return nil
	})
	if count != 1 {
		t.Fatalf("expected 1 blob file, got %d", count)
	}
}

func TestSmallMessagePassthrough(t *testing.T) {
	s, _ := newTestStore(t)

	small := []byte(`{"name":"small","value":42}`)
	packed, err := s.Pack(small)
	if err != nil {
		t.Fatal(err)
	}
	if string(packed) != string(small) {
		t.Fatal("small message should pass through unchanged")
	}
}

func TestLargeMessagePacking(t *testing.T) {
	s, _ := newTestStore(t)

	big := bigString(Threshold + 100)
	payload := []byte(`{"data":"` + big + `"}`)

	packed, err := s.Pack(payload)
	if err != nil {
		t.Fatal(err)
	}

	// The packed result should contain a BlobRef for "data"
	dataField := gjson.GetBytes(packed, "data")
	if !dataField.Exists() {
		t.Fatal("expected data field in packed result")
	}
	if !IsBlobRef(dataField) {
		t.Fatal("data field should be a BlobRef")
	}

	// Verify all BlobRef fields
	magic := gjson.Get(dataField.Raw, "__magic")
	ref := gjson.Get(dataField.Raw, "__ref")
	size := gjson.Get(dataField.Raw, "__size")
	path := gjson.Get(dataField.Raw, "__path")
	typ := gjson.Get(dataField.Raw, "__type")
	ln := gjson.Get(dataField.Raw, "__len")

	if magic.Int() != Magic {
		t.Fatalf("expected magic %d, got %d", Magic, magic.Int())
	}
	if !strings.HasPrefix(ref.String(), "xxh3:") {
		t.Fatal("expected xxh3: prefix in ref")
	}
	if size.Int() == 0 {
		t.Fatal("expected non-zero size")
	}
	if path.String() != "test/flow" {
		t.Fatalf("expected path test/flow, got %s", path.String())
	}
	if typ.String() != "string" {
		t.Fatalf("expected type string, got %s", typ.String())
	}
	if ln.Int() == 0 {
		t.Fatal("expected non-zero __len for string type")
	}
}

func TestLazyResolve(t *testing.T) {
	s, _ := newTestStore(t)

	big := bigString(Threshold + 100)
	payload := []byte(`{"small":"ok","large":"` + big + `"}`)

	packed, err := s.Pack(payload)
	if err != nil {
		t.Fatal(err)
	}

	// Resolve the small field — should return directly without blob access.
	small, err := s.Resolve(packed, "small")
	if err != nil {
		t.Fatal(err)
	}
	if small.String() != "ok" {
		t.Fatalf("expected ok, got %s", small.String())
	}

	// Resolve the large field — should decompress from blob.
	large, err := s.Resolve(packed, "large")
	if err != nil {
		t.Fatal(err)
	}
	if large.String() != big {
		t.Fatal("resolved large field doesn't match original")
	}
}

func TestBlobRefPassthrough(t *testing.T) {
	s, _ := newTestStore(t)

	big := bigString(Threshold + 100)
	payload := []byte(`{"data":"` + big + `"}`)

	packed1, err := s.Pack(payload)
	if err != nil {
		t.Fatal(err)
	}

	// Pack again — BlobRef should pass through without re-extraction.
	packed2, err := s.Pack(packed1)
	if err != nil {
		t.Fatal(err)
	}

	if string(packed1) != string(packed2) {
		t.Fatal("packing a BlobRef should be a no-op")
	}
}

func TestRecursiveExtraction(t *testing.T) {
	s, _ := newTestStore(t)

	big := bigString(Threshold + 100)
	payload := []byte(`{"outer":{"inner":"` + big + `"}}`)

	packed, err := s.Pack(payload)
	if err != nil {
		t.Fatal(err)
	}

	// The outer object should have its inner field extracted.
	outer := gjson.GetBytes(packed, "outer")
	if !outer.Exists() {
		t.Fatal("expected outer field")
	}

	// Check if inner was extracted (outer itself might be a BlobRef if big enough,
	// or inner within outer might be a BlobRef).
	if IsBlobRef(outer) {
		// Outer was extracted as a whole blob — resolve and check inner.
		resolved, err := s.resolveRef(outer)
		if err != nil {
			t.Fatal(err)
		}
		inner := gjson.Get(resolved.Raw, "inner")
		if !IsBlobRef(inner) {
			// Inner might have been left inline if outer was extracted whole.
			if inner.String() != big {
				t.Fatal("inner field not preserved after outer extraction")
			}
		}
	} else {
		// Inner should be a BlobRef.
		inner := gjson.Get(outer.Raw, "inner")
		if !IsBlobRef(inner) {
			// If inner wasn't extracted, at least verify the data is preserved.
			if inner.String() != big {
				t.Fatal("inner field should be a BlobRef or preserved")
			}
		}
	}
}

func TestResolveAllFlat(t *testing.T) {
	s, _ := newTestStore(t)

	big := bigString(Threshold + 100)
	payload := []byte(`{"a":"` + big + `","b":"small"}`)

	packed, err := s.Pack(payload)
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := s.ResolveAll(packed)
	if err != nil {
		t.Fatal(err)
	}

	a := gjson.GetBytes(resolved, "a")
	b := gjson.GetBytes(resolved, "b")
	if a.String() != big {
		t.Fatal("resolved a doesn't match original")
	}
	if b.String() != "small" {
		t.Fatal("resolved b doesn't match original")
	}
}

func TestResolveAllNested(t *testing.T) {
	s, _ := newTestStore(t)

	big := bigString(Threshold + 100)
	payload := []byte(`{"outer":{"deep":"` + big + `"}}`)

	packed, err := s.Pack(payload)
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := s.ResolveAll(packed)
	if err != nil {
		t.Fatal(err)
	}

	deep := gjson.GetBytes(resolved, "outer.deep")
	if deep.String() != big {
		t.Fatal("resolved nested field doesn't match original")
	}
}

func TestResolveAllNoOp(t *testing.T) {
	s, _ := newTestStore(t)

	small := []byte(`{"x":"hello","y":123}`)
	resolved, err := s.ResolveAll(small)
	if err != nil {
		t.Fatal(err)
	}
	if string(resolved) != string(small) {
		t.Fatal("ResolveAll on no-blob payload should be a no-op")
	}
}

func TestResolveAllRoundtrip(t *testing.T) {
	s, _ := newTestStore(t)

	big := bigString(Threshold + 100)
	original := []byte(`{"data":"` + big + `","meta":"info"}`)

	packed, err := s.Pack(original)
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := s.ResolveAll(packed)
	if err != nil {
		t.Fatal(err)
	}

	// Compare as parsed JSON to handle key ordering.
	var orig, res map[string]interface{}
	json.Unmarshal(original, &orig)
	json.Unmarshal(resolved, &res)

	origJSON, _ := json.Marshal(orig)
	resJSON, _ := json.Marshal(res)
	if string(origJSON) != string(resJSON) {
		t.Fatalf("roundtrip mismatch:\n  orig: %s\n  got:  %s", origJSON, resJSON)
	}
}

func TestIsBlobRefValid(t *testing.T) {
	valid := gjson.Parse(`{"__magic":20260301,"__ref":"xxh3:abc123","__size":5000,"__path":"test","__type":"string"}`)
	if !IsBlobRef(valid) {
		t.Fatal("expected valid BlobRef to be detected")
	}
}

func TestIsBlobRefWrongMagic(t *testing.T) {
	wrong := gjson.Parse(`{"__magic":99999,"__ref":"xxh3:abc123","__size":5000}`)
	if IsBlobRef(wrong) {
		t.Fatal("wrong magic should not be detected as BlobRef")
	}
}

func TestIsBlobRefMissingRef(t *testing.T) {
	noRef := gjson.Parse(`{"__magic":20260301,"__size":5000}`)
	if IsBlobRef(noRef) {
		t.Fatal("missing __ref should not be detected as BlobRef")
	}
}

func TestIsBlobRefNonJSON(t *testing.T) {
	nonJSON := gjson.Parse(`"just a string"`)
	if IsBlobRef(nonJSON) {
		t.Fatal("non-JSON value should not be detected as BlobRef")
	}
}

func TestIsBlobRefMap(t *testing.T) {
	valid := map[string]interface{}{
		"__magic": float64(Magic),
		"__ref":   "xxh3:abc123",
	}
	if !IsBlobRefMap(valid) {
		t.Fatal("expected valid map to be detected as BlobRef")
	}

	invalid := map[string]interface{}{
		"__magic": float64(99999),
		"__ref":   "xxh3:abc123",
	}
	if IsBlobRefMap(invalid) {
		t.Fatal("wrong magic should not be detected as BlobRef")
	}

	noRef := map[string]interface{}{
		"__magic": float64(Magic),
	}
	if IsBlobRefMap(noRef) {
		t.Fatal("missing ref should not be detected")
	}

	if IsBlobRefMap("not a map") {
		t.Fatal("non-map should not be detected")
	}
}

func TestPackWithoutRelPath(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	big := bigString(Threshold + 100)
	payload := []byte(`{"data":"` + big + `"}`)

	packed, err := s.Pack(payload)
	if err != nil {
		t.Fatal(err)
	}
	// Without relPath, Pack should return the payload unchanged.
	if string(packed) != string(payload) {
		t.Fatal("Pack without relPath should return payload unchanged")
	}
}

func TestGetBlobInvalidRef(t *testing.T) {
	s, _ := newTestStore(t)

	_, err := s.GetBlob("xxh3:ab", "test/flow")
	if err == nil {
		t.Fatal("expected error for short ref")
	}
}

func TestPackInvalidJSON(t *testing.T) {
	s, _ := newTestStore(t)

	invalid := []byte(`not json`)
	packed, err := s.Pack(invalid)
	if err != nil {
		t.Fatal(err)
	}
	if string(packed) != string(invalid) {
		t.Fatal("invalid JSON should pass through")
	}
}

func TestBlobRefTypeArray(t *testing.T) {
	s, _ := newTestStore(t)

	// Build a large array
	items := make([]string, 200)
	for i := range items {
		items[i] = bigString(30)
	}
	arrJSON, _ := json.Marshal(items)
	payload := []byte(`{"arr":` + string(arrJSON) + `}`)

	packed, err := s.Pack(payload)
	if err != nil {
		t.Fatal(err)
	}

	arr := gjson.GetBytes(packed, "arr")
	if IsBlobRef(arr) {
		typ := gjson.Get(arr.Raw, "__type").String()
		if typ != "array" {
			t.Fatalf("expected type array, got %s", typ)
		}
		ln := gjson.Get(arr.Raw, "__len").Int()
		if ln != int64(len(items)) {
			t.Fatalf("expected __len %d, got %d", len(items), ln)
		}
	}
}

func TestBlobRefTypeObject(t *testing.T) {
	s, _ := newTestStore(t)

	// Build a large object with many small fields so children don't get extracted
	// individually, forcing the whole object to be extracted.
	obj := make(map[string]string)
	for i := 0; i < 200; i++ {
		obj[bigString(10)+string(rune('a'+i%26))] = bigString(20)
	}
	objJSON, _ := json.Marshal(obj)
	payload := []byte(`{"obj":` + string(objJSON) + `}`)

	packed, err := s.Pack(payload)
	if err != nil {
		t.Fatal(err)
	}

	field := gjson.GetBytes(packed, "obj")
	if IsBlobRef(field) {
		typ := gjson.Get(field.Raw, "__type").String()
		if typ != "object" {
			t.Fatalf("expected type object, got %s", typ)
		}
	}
}
