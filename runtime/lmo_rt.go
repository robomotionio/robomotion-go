package runtime

import (
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"

	"github.com/robomotionio/robomotion-go/runtime/lmo"
	"github.com/robomotionio/robomotion-go/utils"
)

var lmoStore *lmo.Store

// InitLMOStore creates a Store using the platform config directory.
func InitLMOStore() error {
	if lmoStore != nil {
		lmoStore.Close()
		lmoStore = nil
	}
	s, err := lmo.NewStore(utils.ConfigDir())
	if err != nil {
		return err
	}
	lmoStore = s
	return nil
}

// SetLMOStorePath sets the relative path for the blob store.
// Must be called before Pack/PutBlob can work (e.g. "robots/{id}/flows/{flowID}").
func SetLMOStorePath(relPath string) error {
	if lmoStore == nil {
		return fmt.Errorf("lmo store not initialised")
	}
	return lmoStore.SetRelPath(relPath)
}

// CloseLMOStore releases zstd resources.
func CloseLMOStore() {
	if lmoStore == nil {
		return
	}
	lmoStore.Close()
	lmoStore = nil
}

// LMOResolve lazily resolves a BlobRef for a specific field path.
// Falls back to plain gjson if the store is nil.
func LMOResolve(data []byte, key string) (gjson.Result, error) {
	if lmoStore == nil {
		return gjson.GetBytes(data, key), nil
	}
	return lmoStore.Resolve(data, key)
}

// LMOResolveAll eagerly resolves all BlobRefs in the payload.
// Returns data unchanged if the store is nil.
func LMOResolveAll(data []byte) ([]byte, error) {
	if lmoStore == nil {
		return data, nil
	}
	return lmoStore.ResolveAll(data)
}

// LMOPack extracts large fields from the payload as blobs.
// Returns payload unchanged if the store is nil or relPath is not set.
func LMOPack(payload []byte) ([]byte, error) {
	if lmoStore == nil {
		return payload, nil
	}
	return lmoStore.Pack(payload)
}

// ResolveBlobRefValue resolves a BlobRef from a map[string]interface{} value.
// Extracts __ref and __path, reads the blob, and unmarshals the result.
func ResolveBlobRefValue(m map[string]interface{}) (interface{}, error) {
	if lmoStore == nil {
		return nil, fmt.Errorf("lmo store not initialised")
	}
	ref, _ := m["__ref"].(string)
	relPath, _ := m["__path"].(string)
	if ref == "" {
		return nil, fmt.Errorf("lmo: missing __ref")
	}

	// Learn the store relPath from the first BlobRef we encounter.
	if lmoStore.RelPath() == "" && relPath != "" {
		_ = lmoStore.SetRelPath(relPath)
	}

	data, err := lmoStore.GetBlob(ref, relPath)
	if err != nil {
		return nil, err
	}
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("lmo: unmarshal blob: %w", err)
	}
	return result, nil
}

// PackValue marshals a Go value to JSON and packs it into the blob store
// if it exceeds the threshold. Returns (blobRefMap, true) if packed,
// or (nil, false) if the value is small enough to send inline.
func PackValue(value interface{}) (interface{}, bool, error) {
	if lmoStore == nil || lmoStore.RelPath() == "" {
		return nil, false, nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return nil, false, err
	}

	if len(data) < lmo.Threshold {
		return nil, false, nil
	}

	packed, err := lmoStore.Pack(data)
	if err != nil {
		return nil, false, err
	}

	// If Pack produced a BlobRef at the top level, return it as a map
	var result interface{}
	if err := json.Unmarshal(packed, &result); err != nil {
		return nil, false, err
	}

	// Check if the result itself is a BlobRef
	if lmo.IsBlobRefMap(result) {
		return result, true, nil
	}

	// Pack may have replaced children but not the root — return the packed map
	if len(packed) < len(data) {
		return result, true, nil
	}

	return nil, false, nil
}
