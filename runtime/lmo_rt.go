package runtime

import (
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"

	"github.com/robomotionio/robomotion-go/runtime/lmo"
	"github.com/robomotionio/robomotion-go/utils"
)

var lmoReader *lmo.Reader

// InitLMOReader creates a Reader using the platform config directory.
func InitLMOReader() error {
	if lmoReader != nil {
		lmoReader.Close()
		lmoReader = nil
	}
	r, err := lmo.NewReader(utils.ConfigDir())
	if err != nil {
		return err
	}
	lmoReader = r
	return nil
}

// CloseLMOReader releases the zstd decoder.
func CloseLMOReader() {
	if lmoReader == nil {
		return
	}
	lmoReader.Close()
	lmoReader = nil
}

// LMOResolve lazily resolves a BlobRef for a specific field path.
// Falls back to plain gjson if the reader is nil.
func LMOResolve(data []byte, key string) (gjson.Result, error) {
	if lmoReader == nil {
		return gjson.GetBytes(data, key), nil
	}
	return lmoReader.Resolve(data, key)
}

// LMOResolveAll eagerly resolves all BlobRefs in the payload.
// Returns data unchanged if the reader is nil.
func LMOResolveAll(data []byte) ([]byte, error) {
	if lmoReader == nil {
		return data, nil
	}
	return lmoReader.ResolveAll(data)
}

// ResolveBlobRefValue resolves a BlobRef from a map[string]interface{} value.
// Extracts __ref and __path, reads the blob, and unmarshals the result.
func ResolveBlobRefValue(m map[string]interface{}) (interface{}, error) {
	if lmoReader == nil {
		return nil, fmt.Errorf("lmo reader not initialised")
	}
	ref, _ := m["__ref"].(string)
	relPath, _ := m["__path"].(string)
	if ref == "" {
		return nil, fmt.Errorf("lmo: missing __ref")
	}
	data, err := lmoReader.GetBlob(ref, relPath)
	if err != nil {
		return nil, err
	}
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("lmo: unmarshal blob: %w", err)
	}
	return result, nil
}
