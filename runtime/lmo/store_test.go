package lmo

import (
	"encoding/json"
	"strconv"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

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
	// __len is the rune count (not byte count) of the value. For ASCII
	// content these are equal, but pin the rune-count invariant —
	// TestBlobRefStringLenIsRuneCount covers the multi-byte case.
	expectedLen := int64(utf8.RuneCountInString(big))
	if ln.Int() != expectedLen {
		t.Fatalf("expected __len %d (rune count of payload), got %d", expectedLen, ln.Int())
	}
	// __size is the UTF-8 byte count of the raw JSON-serialized value
	// (the string with its surrounding quotes). Wire contract across SDKs.
	expectedSize := int64(len(`"`+big+`"`))
	if size.Int() != expectedSize {
		t.Fatalf("expected __size %d (utf-8 bytes of raw JSON), got %d", expectedSize, size.Int())
	}
}

// Pin: BlobRef envelope metadata for non-Latin (multi-byte UTF-8) string
// content. The wire contract that every SDK + the robot must agree on:
//   - __len = rune count (Unicode code-point count), NOT byte count and
//     NOT UTF-16 code-unit count.
//   - __size = raw UTF-8 byte count of the JSON-serialized value (string
//     plus surrounding quotes), NOT a JSON-escaped form like İ.
// Multi-byte fixtures make the rune-vs-byte distinction visible. Customer
// payload is Turkish (Ihlas); Japanese covers 3-byte UTF-8 sequences.
func TestBlobRefStringNonLatinMetadata(t *testing.T) {
	cases := []struct {
		name      string
		rune      string // single character to repeat
		runeBytes int    // UTF-8 byte count for one rune
		count     int    // repetitions
	}{
		// Turkish "İ" (U+0130) — 2 bytes UTF-8. 2050 × 2 = 4100 > Threshold.
		{"Turkish_I_dotted", "İ", 2, 2050},
		// Japanese "日" (U+65E5) — 3 bytes UTF-8. 1400 × 3 = 4200 > Threshold.
		{"Japanese_Sun", "日", 3, 1400},
		// Emoji "😀" (U+1F600) — 4 bytes UTF-8, non-BMP. 1100 × 4 = 4400
		// > Threshold. Java's String.length() / .NET's string.Length would
		// return 2200 (UTF-16 surrogate pairs) here — but production uses
		// code-point count, which agrees with Go's rune count.
		{"Emoji_Grinning", "😀", 4, 1100},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newTestStore(t)

			big := strings.Repeat(tc.rune, tc.count)
			byteLen := len(big)
			runeLen := utf8.RuneCountInString(big)
			if byteLen != tc.runeBytes*tc.count || runeLen != tc.count {
				t.Fatalf("fixture invariant broken: bytes=%d expected=%d runes=%d expected=%d",
					byteLen, tc.runeBytes*tc.count, runeLen, tc.count)
			}
			if byteLen < Threshold {
				t.Fatalf("fixture too small: byte len %d < Threshold %d", byteLen, Threshold)
			}

			payload := []byte(`{"data":"` + big + `"}`)
			packed, err := s.Pack(payload)
			if err != nil {
				t.Fatal(err)
			}

			dataField := gjson.GetBytes(packed, "data")
			if !IsBlobRef(dataField) {
				t.Fatal("data field should be a BlobRef")
			}
			if got := gjson.Get(dataField.Raw, "__type").String(); got != "string" {
				t.Fatalf("expected type string, got %s", got)
			}
			if got := gjson.Get(dataField.Raw, "__len").Int(); got != int64(runeLen) {
				t.Fatalf("__len should be rune count %d, got %d (would be %d if byte-counting)",
					runeLen, got, byteLen)
			}
			// __size is bytes of raw JSON form: "İİİ..." = byteLen + 2 quotes.
			expectedSize := int64(byteLen + 2)
			if got := gjson.Get(dataField.Raw, "__size").Int(); got != expectedSize {
				t.Fatalf("__size should be UTF-8 byte count %d, got %d (would be %d if JSON-escape-counting)",
					expectedSize, got, 6*tc.count+2)
			}
		})
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

// TestNestedBlobRef_SurvivesResolveAll pins the customer's iter-17 bug
// pattern: when extractObject's !modified branch packs a whole container
// as a single blob, the blob bytes preserve any pre-existing BlobRef
// envelopes inside. Without recursive resolveValue, those nested
// BlobRefs survive ResolveAll and surface upstream as stub objects.
func TestNestedBlobRef_SurvivesResolveAll(t *testing.T) {
	s, _ := newTestStore(t)

	// Pack a "response" array (16 SOAP-ish entries, ~12 KB).
	respElements := make([]string, 16)
	for i := range respElements {
		respElements[i] = `{"raw":"` + strings.Repeat("X", 680) +
			`","sgkSicil":"s","sirketAdi":"ihl","sonucKod":"0"}`
	}
	respArr := []byte("[" + strings.Join(respElements, ",") + "]")
	respRef, err := s.PutBlob(respArr)
	if err != nil {
		t.Fatalf("PutBlob: %v", err)
	}
	respEnv := `{"__ref":"` + respRef + `","__magic":20260301,"__size":12500,"__path":"test/flow","__type":"array","__len":16}`

	// Build msg shape that triggers !modified whole-pack on api: api > 4 KB
	// but no individual child reaches 4 KB.
	loginEntries := make([]string, 16)
	for i := range loginEntries {
		loginEntries[i] = `{"sirketAdi":"İHLAS HABER AJANSI A.Ş.","sgkSicil":"2.6031.01.03","isyeriKodu":"x","kullaniciAdi":"u","isyeriSifresi":"p","token":"02bf0810-6d0e-4197-ad80-0dcb7fe5b962"}`
	}
	loginSuccess := "[" + strings.Join(loginEntries, ",") + "]"
	wsLogin := `{"response":` + respEnv + `,"loginSuccess":` + loginSuccess + `,"loginFailed":[]}`

	medium := func() string {
		entries := make([]string, 4)
		for i := range entries {
			entries[i] = `{"sirketAdi":"İHLAS","sgkSicil":"2.6031","note":"` + strings.Repeat("y", 80) + `"}`
		}
		return "[" + strings.Join(entries, ",") + "]"
	}
	api := `{"wsLogin":` + wsLogin +
		`,"raporAramaTarihile":{"response":` + medium() + `,"failedResponses":[],"noReports":[],"Reports":[]}` +
		`,"raporOnay":{"response":` + medium() + `,"confirmedReports":[],"reportsNotConfirmed":[]}` +
		`,"raporOkunduKapat":{"response":` + medium() + `,"reportsNotClosed":[]}` + `}`

	t.Logf("api size: %d (threshold %d)  wsLogin size: %d", len(api), Threshold, len(wsLogin))

	msg := []byte(`{"constants":{"api":` + api + `,"urls":{}}}`)

	packed, err := s.Pack(msg)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	apiAfter := gjson.GetBytes(packed, "constants.api")
	t.Logf("after Pack: api isBlobRef=%v", IsBlobRef(apiAfter))

	resolved, err := s.ResolveAll(packed)
	if err != nil {
		t.Fatalf("ResolveAll: %v", err)
	}

	// Scan for any surviving BlobRef envelope.
	survRef, survPath := findAnyBlobRef(resolved, "")
	if survRef != "" {
		t.Fatalf("NESTED BLOBREF SURVIVED RESOLVEALL — bug reproduced!\n"+
			"surviving ref: %s\nsurviving path: %s\n", survRef, survPath)
	}
	respCheck := gjson.GetBytes(resolved, "constants.api.wsLogin.response")
	if !respCheck.IsArray() {
		t.Fatalf("expected resolved response to be an array, got: %s", respCheck.Raw[:200])
	}
}

func findAnyBlobRef(data []byte, prefix string) (string, string) {
	if !gjson.ValidBytes(data) {
		return "", ""
	}
	return scan(gjson.ParseBytes(data), prefix)
}
func scan(n gjson.Result, prefix string) (string, string) {
	if IsBlobRef(n) {
		return gjson.Get(n.Raw, "__ref").String(), prefix
	}
	if n.Type != gjson.JSON {
		return "", ""
	}
	var foundRef, foundPath string
	if strings.HasPrefix(n.Raw, "{") {
		n.ForEach(func(k, v gjson.Result) bool {
			p := k.String()
			if prefix != "" {
				p = prefix + "." + p
			}
			if r, pp := scan(v, p); r != "" {
				foundRef, foundPath = r, pp
				return false
			}
			return true
		})
		return foundRef, foundPath
	}
	if strings.HasPrefix(n.Raw, "[") {
		i := 0
		n.ForEach(func(_, v gjson.Result) bool {
			p := prefix
			if p != "" {
				p += "."
			}
			p += strconv.Itoa(i)
			i++
			if r, pp := scan(v, p); r != "" {
				foundRef, foundPath = r, pp
				return false
			}
			return true
		})
	}
	return foundRef, foundPath
}

// TestArrayNestedBlobRef_SurvivesResolveAll pins the follow-up class:
// a BlobRef envelope sitting inside an array element survives ResolveAll
// unless resolveValue also recurses into arrays.
func TestArrayNestedBlobRef_SurvivesResolveAll(t *testing.T) {
	s, _ := newTestStore(t)

	innerData := []byte(`"the inner blob content"`)
	innerRef, err := s.PutBlob(innerData)
	if err != nil {
		t.Fatalf("PutBlob: %v", err)
	}
	innerEnv := `{"__ref":"` + innerRef + `","__magic":20260301,` +
		`"__size":` + strconv.Itoa(len(innerData)) +
		`,"__path":"test/flow","__type":"string","__len":22}`

	var elements []string
	for i := 0; i < 30; i++ {
		elements = append(elements, `{"i":`+strconv.Itoa(i)+`,"pad":"`+strings.Repeat("x", 150)+`"}`)
	}
	envIndex := 15
	elements = append(elements[:envIndex], append([]string{innerEnv}, elements[envIndex:]...)...)
	bigArray := "[" + strings.Join(elements, ",") + "]"

	msg := []byte(`{"bigArray":` + bigArray + `,"other":"fluff"}`)

	packed, err := s.Pack(msg)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if !IsBlobRef(gjson.GetBytes(packed, "bigArray")) {
		t.Fatalf("bigArray should have been packed as BlobRef")
	}

	resolved, err := s.ResolveAll(packed)
	if err != nil {
		t.Fatalf("ResolveAll: %v", err)
	}

	if ref, path := scanForBlobRef(resolved); ref != "" {
		t.Fatalf("array-nested BlobRef survived: ref=%s path=%s", ref, path)
	}

	got := gjson.GetBytes(resolved, "bigArray."+strconv.Itoa(envIndex))
	if got.Type != gjson.String || got.String() != "the inner blob content" {
		t.Fatalf("inner not unwrapped, got type=%d raw=%q", got.Type, got.Raw)
	}
}

func scanForBlobRef(data []byte) (string, string) {
	if !gjson.ValidBytes(data) {
		return "", ""
	}
	return scanRefRec(gjson.ParseBytes(data), "")
}
func scanRefRec(n gjson.Result, prefix string) (string, string) {
	if IsBlobRef(n) {
		return gjson.Get(n.Raw, "__ref").String(), prefix
	}
	if n.Type != gjson.JSON {
		return "", ""
	}
	var ref, path string
	if strings.HasPrefix(n.Raw, "{") {
		n.ForEach(func(k, v gjson.Result) bool {
			p := k.String()
			if prefix != "" {
				p = prefix + "." + p
			}
			if r, pp := scanRefRec(v, p); r != "" {
				ref, path = r, pp
				return false
			}
			return true
		})
	} else if strings.HasPrefix(n.Raw, "[") {
		i := 0
		n.ForEach(func(_, v gjson.Result) bool {
			p := prefix
			if p != "" {
				p += "."
			}
			p += strconv.Itoa(i)
			i++
			if r, pp := scanRefRec(v, p); r != "" {
				ref, path = r, pp
				return false
			}
			return true
		})
	}
	return ref, path
}

// TestPutBlob_ConcurrentSameContent_NoShortReads pins the atomic-write
// guarantee: when N goroutines race to PutBlob the same content, no concurrent
// reader observes a partial file. Pre-fix (non-atomic os.WriteFile) this
// surfaced as zstd "unexpected EOF" decompress errors at low rates
// (~115/5000). Post-fix (tmp + rename) every reader sees the full payload.
func TestPutBlob_ConcurrentSameContent_NoShortReads(t *testing.T) {
	s, _ := newTestStore(t)

	// Payload large enough that compressed bytes don't fit in one write syscall
	// and partial reads have visible effect on decompression.
	data := []byte(bigString(64 * 1024))

	const writers = 16
	const iterations = 200

	var wg sync.WaitGroup
	errs := make(chan error, writers*iterations)

	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				ref, err := s.PutBlob(data)
				if err != nil {
					errs <- err
					return
				}
				got, err := s.GetBlob(ref, "test/flow")
				if err != nil {
					errs <- err
					return
				}
				if len(got) != len(data) {
					errs <- &shortReadErr{got: len(got), want: len(data)}
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent PutBlob/GetBlob race produced an error: %v", err)
	}
}

type shortReadErr struct{ got, want int }

func (e *shortReadErr) Error() string {
	return "short read: got " + strconv.Itoa(e.got) + " bytes, want " + strconv.Itoa(e.want)
}

// TestPutBlob_RewritesEmptyLeftover pins the dedup gate: a zero-byte file at
// the target path (e.g. left by a crashed writer pre-fix) must NOT be honored
// by dedup. PutBlob must rewrite it with the real compressed bytes.
func TestPutBlob_RewritesEmptyLeftover(t *testing.T) {
	s, dir := newTestStore(t)

	data := []byte(`"recover from leftover"`)
	ref := hashRef(data)

	// Plant a zero-byte file at the destination, simulating a crashed prior
	// PutBlob from before the atomic-write fix.
	p := filepath.Join(dir, "store", "test/flow", "blobs", ref[5:7], ref[7:])
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, nil, 0644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(p)
	if err != nil || info.Size() != 0 {
		t.Fatalf("setup: expected zero-byte planted file, got size=%d err=%v", info.Size(), err)
	}

	got, err := s.PutBlob(data)
	if err != nil {
		t.Fatalf("PutBlob over zero-byte file: %v", err)
	}
	if got != ref {
		t.Fatalf("ref mismatch: got %s, want %s", got, ref)
	}

	// Read back and verify content is real, not empty.
	roundtrip, err := s.GetBlob(ref, "test/flow")
	if err != nil {
		t.Fatalf("GetBlob after PutBlob over leftover: %v", err)
	}
	if string(roundtrip) != string(data) {
		t.Fatalf("content mismatch: got %q, want %q", roundtrip, data)
	}
}
