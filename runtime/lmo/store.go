package lmo

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/klauspost/compress/zstd"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/zeebo/xxh3"
)

const (
	Magic     = 20260301
	Threshold = 4096 // 4KB
)

// BlobRef replaces a large field value in the message.
type BlobRef struct {
	Ref   string `json:"__ref"`
	Magic int    `json:"__magic"`
	Size  int    `json:"__size"`
	Path  string `json:"__path"`
	Type  string `json:"__type"`
	Len   int    `json:"__len,omitempty"`
}

// Store manages content-addressed, zstd-compressed blobs on disk.
// It supports both reading blobs (from any relPath via BlobRef.__path) and
// writing blobs (to its own configured relPath).
type Store struct {
	configDir string
	relPath   string // set when known; required for Pack/PutBlob
	enc       *zstd.Encoder
	dec       *zstd.Decoder
}

// NewStore creates a Store with both encoder and decoder.
// relPath can be set later via SetRelPath once the store path is known.
func NewStore(configDir string) (*Store, error) {
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return nil, fmt.Errorf("lmo: zstd encoder: %w", err)
	}

	dec, err := zstd.NewReader(nil)
	if err != nil {
		enc.Close()
		return nil, fmt.Errorf("lmo: zstd decoder: %w", err)
	}

	return &Store{configDir: configDir, enc: enc, dec: dec}, nil
}

// SetRelPath sets the store's relative path and creates the blobs directory.
// relPath is embedded in BlobRefs (e.g. "robots/{id}/flows/{flowID}").
func (s *Store) SetRelPath(relPath string) error {
	s.relPath = relPath
	blobDir := filepath.Join(s.configDir, "store", relPath, "blobs")
	if err := os.MkdirAll(blobDir, 0755); err != nil {
		return fmt.Errorf("lmo: create blobs dir: %w", err)
	}
	return nil
}

// RelPath returns the current relative path.
func (s *Store) RelPath() string {
	return s.relPath
}

// Close releases zstd resources.
func (s *Store) Close() {
	if s.enc != nil {
		s.enc.Close()
	}
	if s.dec != nil {
		s.dec.Close()
	}
}

// --- Read operations (use relPath from BlobRef.__path) ---

// GetBlob reads and decompresses a blob given its ref and relPath.
func (s *Store) GetBlob(ref, relPath string) ([]byte, error) {
	h := strings.TrimPrefix(ref, "xxh3:")
	if len(h) < 3 {
		return nil, fmt.Errorf("lmo: invalid ref %q: hex too short", ref)
	}
	p := filepath.Join(s.configDir, "store", relPath, "blobs", h[:2], h[2:])

	compressed, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("lmo: read blob %s: %w", ref, err)
	}

	data, err := s.dec.DecodeAll(compressed, nil)
	if err != nil {
		return nil, fmt.Errorf("lmo: decompress blob %s: %w", ref, err)
	}

	return data, nil
}

// Resolve lazily resolves a BlobRef for a specific field path.
//
// Each `resolveRef` call is wrapped in `fullyResolveRef`, which loops while
// the result is itself a BlobRef envelope. This handles double-nested
// envelopes — same pathology as ResolveAll's recursive fix, applied to the
// singular path-based variant.
func (s *Store) Resolve(data []byte, key string) (gjson.Result, error) {
	value := gjson.GetBytes(data, key)
	if value.Exists() {
		if IsBlobRef(value) {
			return s.fullyResolveRef(value)
		}
		return value, nil
	}

	// Walk path segments to find an intermediate BlobRef
	parts := splitPath(key)
	for i := 1; i <= len(parts); i++ {
		prefix := strings.Join(parts[:i], ".")
		seg := gjson.GetBytes(data, prefix)
		if !seg.Exists() {
			continue
		}
		if IsBlobRef(seg) {
			resolved, err := s.fullyResolveRef(seg)
			if err != nil {
				return gjson.Result{}, err
			}
			if i == len(parts) {
				return resolved, nil
			}
			remaining := strings.Join(parts[i:], ".")
			inner := gjson.Get(resolved.Raw, remaining)
			if inner.Exists() {
				if IsBlobRef(inner) {
					return s.fullyResolveRef(inner)
				}
				return inner, nil
			}
			return gjson.Result{}, nil
		}
	}

	return gjson.Result{}, nil
}

// fullyResolveRef wraps resolveRef in a depth-bounded loop so that a blob
// whose content is itself a BlobRef envelope (or a chain of them) gets fully
// unwrapped. Naturally-produced chains via Pack are bounded; the limit is a
// defensive guard against pathological inputs.
func (s *Store) fullyResolveRef(value gjson.Result) (gjson.Result, error) {
	const maxResolveDepth = 32
	current := value
	for depth := 0; depth < maxResolveDepth; depth++ {
		if !IsBlobRef(current) {
			return current, nil
		}
		resolved, err := s.resolveRef(current)
		if err != nil {
			return gjson.Result{}, err
		}
		current = resolved
	}
	return gjson.Result{}, fmt.Errorf("lmo: BlobRef chain exceeded max resolve depth %d (possible cycle or pathological input)", maxResolveDepth)
}

// ResolveAll eagerly resolves every BlobRef in the payload.
func (s *Store) ResolveAll(data []byte) ([]byte, error) {
	if !gjson.ValidBytes(data) {
		return data, nil
	}

	result := gjson.ParseBytes(data)
	if result.Type != gjson.JSON {
		return data, nil
	}

	modified := false
	out := data

	var walkErr error
	result.ForEach(func(key, value gjson.Result) bool {
		fieldKey := key.String()

		newRaw, changed, err := s.resolveValue(value)
		if err != nil {
			walkErr = err
			return false
		}
		if changed {
			out, err = sjson.SetRawBytes(out, escape(fieldKey), newRaw)
			if err != nil {
				walkErr = fmt.Errorf("lmo: sjson set %s: %w", fieldKey, err)
				return false
			}
			modified = true
		}
		return true
	})

	if walkErr != nil {
		return nil, walkErr
	}
	if !modified {
		return data, nil
	}

	return out, nil
}

// --- Write operations (use store's own relPath) ---

// PutBlob stores data as a zstd-compressed blob and returns its XXH3-128 ref.
// If a non-empty blob already exists at the target path, dedup skips writing.
//
// Atomic write: compressed bytes go to a unique tmp file then rename onto the
// final path. This eliminates the race where a concurrent reader's os.Stat
// sees the file mid-write and dedup-skips with partial bytes, surfacing
// upstream as a zstd "unexpected EOF" decompress error.
//
// Dedup verifies the existing file is non-empty before trusting it. A
// zero-byte file (left by a prior kill before this atomic-write fix, or by
// an unrelated tool) is rewritten rather than honored.
func (s *Store) PutBlob(data []byte) (string, error) {
	ref := hashRef(data)

	p := s.blobPathLocal(ref)
	if info, err := os.Stat(p); err == nil && info.Size() > 0 {
		return ref, nil // already exists, non-empty — trust it
	}

	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("lmo: mkdir blob: %w", err)
	}

	compressed := s.enc.EncodeAll(data, nil)

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return "", fmt.Errorf("lmo: create tmp blob: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(compressed); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("lmo: write tmp blob: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("lmo: close tmp blob: %w", err)
	}
	if err := os.Rename(tmpPath, p); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("lmo: rename blob: %w", err)
	}

	return ref, nil
}

// Pack walks the JSON payload, extracts fields >= Threshold as blobs, and
// replaces them with BlobRef markers. Returns the original if nothing extracted.
// Requires relPath to be set via SetRelPath.
func (s *Store) Pack(payload []byte) ([]byte, error) {
	if s.relPath == "" {
		return payload, nil // no store path, can't pack
	}

	if !gjson.ValidBytes(payload) {
		return payload, nil
	}

	result := gjson.ParseBytes(payload)
	if result.Type != gjson.JSON {
		return payload, nil
	}

	modified := false
	out := payload

	var walkErr error
	result.ForEach(func(key, value gjson.Result) bool {
		fieldKey := key.String()

		newRaw, changed, err := s.extractField(value)
		if err != nil {
			walkErr = err
			return false
		}
		if changed {
			out, err = sjson.SetRawBytes(out, escape(fieldKey), newRaw)
			if err != nil {
				walkErr = fmt.Errorf("lmo: sjson set %s: %w", fieldKey, err)
				return false
			}
			modified = true
		}
		return true
	})

	if walkErr != nil {
		return nil, walkErr
	}
	if !modified {
		return payload, nil
	}

	return out, nil
}

// --- Detection ---

// IsBlobRef checks if a gjson.Result is a BlobRef marker.
func IsBlobRef(value gjson.Result) bool {
	if value.Type != gjson.JSON {
		return false
	}
	raw := value.Raw
	if !strings.Contains(raw, "__magic") || !strings.Contains(raw, "__ref") {
		return false
	}
	magic := gjson.Get(raw, "__magic")
	ref := gjson.Get(raw, "__ref")
	return magic.Int() == Magic && ref.Exists() && ref.String() != ""
}

// IsBlobRefMap checks if a map[string]interface{} is a BlobRef marker.
func IsBlobRefMap(val interface{}) bool {
	m, ok := val.(map[string]interface{})
	if !ok {
		return false
	}
	magic, _ := m["__magic"].(float64)
	ref, _ := m["__ref"].(string)
	return int64(magic) == Magic && ref != ""
}

// --- internal helpers ---

func hashRef(data []byte) string {
	h := xxh3.Hash128(data)
	var buf [16]byte
	buf[0] = byte(h.Hi >> 56)
	buf[1] = byte(h.Hi >> 48)
	buf[2] = byte(h.Hi >> 40)
	buf[3] = byte(h.Hi >> 32)
	buf[4] = byte(h.Hi >> 24)
	buf[5] = byte(h.Hi >> 16)
	buf[6] = byte(h.Hi >> 8)
	buf[7] = byte(h.Hi)
	buf[8] = byte(h.Lo >> 56)
	buf[9] = byte(h.Lo >> 48)
	buf[10] = byte(h.Lo >> 40)
	buf[11] = byte(h.Lo >> 32)
	buf[12] = byte(h.Lo >> 24)
	buf[13] = byte(h.Lo >> 16)
	buf[14] = byte(h.Lo >> 8)
	buf[15] = byte(h.Lo)
	return "xxh3:" + hex.EncodeToString(buf[:])
}

// blobPathLocal returns the blob path under the store's own relPath.
func (s *Store) blobPathLocal(ref string) string {
	h := strings.TrimPrefix(ref, "xxh3:")
	return filepath.Join(s.configDir, "store", s.relPath, "blobs", h[:2], h[2:])
}

func (s *Store) resolveRef(value gjson.Result) (gjson.Result, error) {
	ref := gjson.Get(value.Raw, "__ref").String()
	relPath := gjson.Get(value.Raw, "__path").String()
	data, err := s.GetBlob(ref, relPath)
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.ParseBytes(data), nil
}

func (s *Store) resolveValue(value gjson.Result) ([]byte, bool, error) {
	if IsBlobRef(value) {
		resolved, err := s.resolveRef(value)
		if err != nil {
			return nil, false, err
		}
		// Recurse into resolved bytes so nested BlobRefs (from
		// Pack's extractObject !modified whole-pack branch) are
		// also unwrapped. Without this, an inner BlobRef envelope
		// leaks to user code as a stub object and crashes with
		// "TypeError: <field>.push is not a function".
		nestedFixed, changed, err := s.resolveValue(resolved)
		if err != nil {
			return nil, false, err
		}
		if changed {
			return nestedFixed, true, nil
		}
		return []byte(resolved.Raw), true, nil
	}

	raw := value.Raw

	// Object: recurse into children
	if value.Type == gjson.JSON && strings.HasPrefix(raw, "{") {
		inner := gjson.Parse(raw)
		modified := false
		out := []byte(raw)

		var walkErr error
		inner.ForEach(func(key, child gjson.Result) bool {
			childKey := key.String()

			newRaw, changed, err := s.resolveValue(child)
			if err != nil {
				walkErr = err
				return false
			}
			if changed {
				out, err = sjson.SetRawBytes(out, escape(childKey), newRaw)
				if err != nil {
					walkErr = fmt.Errorf("lmo: sjson set %s: %w", childKey, err)
					return false
				}
				modified = true
			}
			return true
		})

		if walkErr != nil {
			return nil, false, walkErr
		}
		if !modified {
			return nil, false, nil
		}
		return out, true, nil
	}

	// Array: recurse into elements so a BlobRef envelope that happens to
	// sit as an array element (e.g. user code did
	// `arr.push(msg.alreadyPackedField)` and arr later crossed the LMO
	// threshold and was packed whole) is also unwrapped. Without this,
	// the inner envelope survives ResolveAll and reaches caller code as
	// a stub object.
	if value.Type == gjson.JSON && strings.HasPrefix(raw, "[") {
		inner := gjson.Parse(raw)
		modified := false
		out := []byte(raw)
		idx := 0

		var walkErr error
		inner.ForEach(func(_, child gjson.Result) bool {
			newRaw, changed, err := s.resolveValue(child)
			if err != nil {
				walkErr = err
				return false
			}
			if changed {
				out, err = sjson.SetRawBytes(out, strconv.Itoa(idx), newRaw)
				if err != nil {
					walkErr = fmt.Errorf("lmo: sjson set [%d]: %w", idx, err)
					return false
				}
				modified = true
			}
			idx++
			return true
		})

		if walkErr != nil {
			return nil, false, walkErr
		}
		if !modified {
			return nil, false, nil
		}
		return out, true, nil
	}

	return nil, false, nil
}

func (s *Store) extractField(value gjson.Result) ([]byte, bool, error) {
	if IsBlobRef(value) {
		return nil, false, nil // already a BlobRef
	}

	raw := value.Raw

	// Object: recurse into children
	if value.Type == gjson.JSON && strings.HasPrefix(raw, "{") {
		return s.extractObject(value)
	}

	// Array or scalar: extract if large
	if len(raw) >= Threshold {
		ref, err := s.PutBlob([]byte(raw))
		if err != nil {
			return nil, false, err
		}

		blobRef := s.buildBlobRef(ref, raw, value)
		refJSON := marshalBlobRef(blobRef)
		return refJSON, true, nil
	}

	return nil, false, nil
}

func (s *Store) extractObject(value gjson.Result) ([]byte, bool, error) {
	raw := value.Raw

	if len(raw) < Threshold {
		return nil, false, nil
	}

	modified := false
	out := []byte(raw)

	var walkErr error
	value.ForEach(func(key, child gjson.Result) bool {
		childKey := key.String()

		newRaw, changed, err := s.extractField(child)
		if err != nil {
			walkErr = err
			return false
		}
		if changed {
			out, err = sjson.SetRawBytes(out, escape(childKey), newRaw)
			if err != nil {
				walkErr = fmt.Errorf("lmo: sjson set %s: %w", childKey, err)
				return false
			}
			modified = true
		}
		return true
	})

	if walkErr != nil {
		return nil, false, walkErr
	}

	if !modified {
		ref, err := s.PutBlob([]byte(raw))
		if err != nil {
			return nil, false, err
		}
		blobRef := s.buildBlobRef(ref, raw, value)
		refJSON := marshalBlobRef(blobRef)
		return refJSON, true, nil
	}

	return out, true, nil
}

func (s *Store) buildBlobRef(ref, raw string, value gjson.Result) BlobRef {
	br := BlobRef{
		Ref:   ref,
		Magic: Magic,
		Size:  len(raw),
		Path:  s.relPath,
	}

	switch {
	case value.Type == gjson.JSON && strings.HasPrefix(raw, "["):
		br.Type = "array"
		br.Len = len(value.Array())
	case value.Type == gjson.JSON && strings.HasPrefix(raw, "{"):
		br.Type = "object"
	case value.Type == gjson.String:
		br.Type = "string"
		br.Len = utf8.RuneCountInString(value.String())
	case value.Type == gjson.Number:
		br.Type = "number"
	case value.Type == gjson.True || value.Type == gjson.False:
		br.Type = "boolean"
	}

	return br
}

func marshalBlobRef(br BlobRef) []byte {
	var out []byte
	out, _ = sjson.SetBytes(out, "__ref", br.Ref)
	out, _ = sjson.SetBytes(out, "__magic", br.Magic)
	out, _ = sjson.SetBytes(out, "__size", br.Size)
	out, _ = sjson.SetBytes(out, "__path", br.Path)
	out, _ = sjson.SetBytes(out, "__type", br.Type)
	if br.Len > 0 {
		out, _ = sjson.SetBytes(out, "__len", br.Len)
	}
	return out
}

func escape(key string) string {
	return strings.ReplaceAll(key, ".", `\.`)
}

func splitPath(path string) []string {
	return strings.Split(path, ".")
}
