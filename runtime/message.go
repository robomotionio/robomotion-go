package runtime

import (
	"encoding/json"
	"strings"

	robocapnp "github.com/robomotionio/robomotion-go/capnp"
	"github.com/robomotionio/robomotion-go/message"
)

func init() {
	message.GetRaw = getRaw
	message.SetRaw = setRaw
}

func getRaw(raw json.RawMessage) (json.RawMessage, error) {

	if IsCapnpCapable() {
		var msg = make(map[string]interface{})
		err := json.Unmarshal(raw, &msg)
		if err != nil {
			return nil, err
		}
		for key, value := range msg {
			if capnp, ok := value.(map[string]interface{}); ok && capnp[robocapnp.ROBOMOTION_CAPNP_ID] != nil {

				capnp_id := capnp[robocapnp.ROBOMOTION_CAPNP_ID].(string)
				if strings.HasPrefix(capnp_id, robocapnp.ROBOMOTION_CAPNP_PREFIX) {
					result, err := robocapnp.Deserialize(capnp_id)
					if err != nil {
						return nil, err
					}
					msg[key] = result

				}
			}

		}
		inMsg, err := json.Marshal(msg)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(inMsg), nil
	}
	return raw, nil
}

func setRaw(data json.RawMessage) (json.RawMessage, error) {
	if IsCapnpCapable() {
		var temp map[string]interface{}
		err := json.Unmarshal(data, &temp)
		if err != nil {
			return data, err
		}
		robotInfo, err := GetRobotInfo()
		if err != nil {
			return data, err
		}
		for key, value := range temp {
			value, err := robocapnp.Serialize(value, robotInfo, key)
			if err != nil {
				return data, err
			}
			temp[key] = value
		}
		data, err = json.Marshal(temp)
		if err != nil {
			return data, err
		}
	}

	return data, nil
}
