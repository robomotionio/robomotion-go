package lmo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const Magic = 20260301

// BlobRef replaces a large field value in the message.
type BlobRef struct {
	Ref   string `json:"__ref"`
	Magic int    `json:"__magic"`
	Size  int    `json:"__size"`
	Path  string `json:"__path"`
	Type  string `json:"__type"`
	Len   int    `json:"__len,omitempty"`
}

// Reader holds a zstd decoder and the config dir for locating blobs.
type Reader struct {
	configDir string
	dec       *zstd.Decoder
}

// NewReader creates a Reader with a zstd decoder.
func NewReader(configDir string) (*Reader, error) {
	dec, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("lmo: zstd decoder: %w", err)
	}
	return &Reader{configDir: configDir, dec: dec}, nil
}

// Close releases zstd resources.
func (r *Reader) Close() {
	if r.dec != nil {
		r.dec.Close()
	}
}

// GetBlob reads and decompresses a blob given its ref and relPath from BlobRef.__path.
func (r *Reader) GetBlob(ref, relPath string) ([]byte, error) {
	h := strings.TrimPrefix(ref, "xxh3:")
	p := filepath.Join(r.configDir, "store", relPath, "blobs", h[:2], h[2:])

	compressed, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("lmo: read blob %s: %w", ref, err)
	}

	data, err := r.dec.DecodeAll(compressed, nil)
	if err != nil {
		return nil, fmt.Errorf("lmo: decompress blob %s: %w", ref, err)
	}

	return data, nil
}

// Resolve lazily resolves a BlobRef for a specific field path.
func (r *Reader) Resolve(data []byte, key string) (gjson.Result, error) {
	// Try direct access
	value := gjson.GetBytes(data, key)
	if value.Exists() {
		if IsBlobRef(value) {
			return r.resolveRef(value)
		}
		return value, nil
	}

	// Walk path segments to find a BlobRef
	parts := strings.Split(key, ".")
	for i := 1; i <= len(parts); i++ {
		prefix := strings.Join(parts[:i], ".")
		seg := gjson.GetBytes(data, prefix)
		if !seg.Exists() {
			continue
		}
		if IsBlobRef(seg) {
			resolved, err := r.resolveRef(seg)
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
					return r.resolveRef(inner)
				}
				return inner, nil
			}
			return gjson.Result{}, nil
		}
	}

	return gjson.Result{}, nil
}

// ResolveAll eagerly resolves every BlobRef in the payload.
// Returns the original data unchanged if no BlobRefs are found.
func (r *Reader) ResolveAll(data []byte) ([]byte, error) {
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

		newRaw, changed, err := r.resolveValue(value)
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

func (r *Reader) resolveRef(value gjson.Result) (gjson.Result, error) {
	ref := gjson.Get(value.Raw, "__ref").String()
	relPath := gjson.Get(value.Raw, "__path").String()
	data, err := r.GetBlob(ref, relPath)
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.ParseBytes(data), nil
}

func (r *Reader) resolveValue(value gjson.Result) ([]byte, bool, error) {
	if IsBlobRef(value) {
		resolved, err := r.resolveRef(value)
		if err != nil {
			return nil, false, err
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

			newRaw, changed, err := r.resolveValue(child)
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

	return nil, false, nil
}

func escape(key string) string {
	return strings.ReplaceAll(key, ".", `\.`)
}
