package runtime

import (
	"encoding/json"

	"github.com/robomotionio/robomotion-go/message"
)

func init() {
	message.GetRaw = getRaw
	message.SetRaw = setRaw
}

func getRaw(raw json.RawMessage) (json.RawMessage, error) {
	if IsLMOCapable() {
		return raw, nil

		var err error
		raw, err = UnpackMessageBytes(raw)
		if err != nil {
			return nil, err
		}
	}
	return raw, nil
}

func setRaw(data json.RawMessage) (json.RawMessage, error) {
	if IsLMOCapable() {
		var err error
		data, err = PackMessageBytes(data)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}
