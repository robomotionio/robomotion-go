package runtime

import (
	"encoding/json"

	"github.com/robomotionio/robomotion-go/message"
	"github.com/tidwall/gjson"
)

func init() {
	message.GetRaw = getRaw
	message.SetRaw = setRaw
	message.Resolve = func(data []byte, key string) (gjson.Result, error) {
		return LMOResolveSubtree(data, key)
	}
}

// WithUnpack resolves all BlobRefs in the message payload.
func WithUnpack() message.GetOption {
	return func(raw json.RawMessage) (json.RawMessage, error) {
		return LMOResolveAll(raw)
	}
}

// WithPack extracts large fields from the message payload as blobs.
func WithPack() message.SetOption {
	return func(raw json.RawMessage) (json.RawMessage, error) {
		return LMOPack(raw)
	}
}

func getRaw(raw json.RawMessage, options ...message.GetOption) (json.RawMessage, error) {
	// Auto-resolve all BlobRefs before returning to the caller.
	resolved, err := LMOResolveAll(raw)
	if err == nil {
		raw = resolved
	}
	for _, opt := range options {
		raw, err = opt(raw)
		if err != nil {
			return nil, err
		}
	}
	return raw, nil
}


func setRaw(raw json.RawMessage, options ...message.SetOption) (json.RawMessage, error) {
	var err error
	for _, opt := range options {
		raw, err = opt(raw)
		if err != nil {
			return nil, err
		}
	}
	return raw, nil
}
