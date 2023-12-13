package runtime

import (
	"encoding/json"

	"github.com/robomotionio/robomotion-go/message"
)

func init() {
	message.GetRaw = getRaw
	message.SetRaw = setRaw
}

// DeserializeLMO for all data
func WithUnpack() message.GetOption {
	return func(raw json.RawMessage) (json.RawMessage, error) {
		if IsLMOCapable() {
			var err error
			raw, err = UnpackMessageBytes(raw)
			if err != nil {
				return nil, err
			}
		}
		return raw, nil
	}
}

// SerializeLMO for all data
func WithPack() message.SetOption {
	return func(raw json.RawMessage) (json.RawMessage, error) {
		if IsLMOCapable() {
			var err error
			raw, err = PackMessageBytes(raw)
			if err != nil {
				return nil, err
			}
		}

		return raw, nil
	}
}

func getRaw(raw json.RawMessage, options ...message.GetOption) (json.RawMessage, error) {
	var err error
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
