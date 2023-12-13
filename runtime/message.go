package runtime

import (
	"encoding/json"
	"log"

	"github.com/robomotionio/robomotion-go/message"
)

func init() {
	message.GetRaw = getRaw
	message.SetRaw = setRaw
}

// DeserializeLMO for all data
func WithUnpack() message.Option {
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
func WithPack() message.Option {
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

func getRaw(raw json.RawMessage, options ...message.Option) (json.RawMessage, error) {
	for _, opt := range options {
		_raw, err := opt(raw)
		if err != nil {
			log.Printf("Option could not be applied %+v \n", err)
			continue
		}
		raw = _raw
	}
	return raw, nil
}

func setRaw(raw json.RawMessage, options ...message.Option) (json.RawMessage, error) {

	for _, opt := range options {
		_raw, err := opt(raw)
		if err != nil {
			log.Printf("Option could not be applied %+v \n", err)
			continue
		}
		raw = _raw
	}
	return raw, nil
}
