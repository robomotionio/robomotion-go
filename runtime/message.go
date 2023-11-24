package runtime

import (
	"encoding/json"
	"strings"

	robocapnp "github.com/robomotionio/robomotion-go/capnp"
	"github.com/robomotionio/robomotion-go/message"
)

func GetRaw(ctx message.Context) (json.RawMessage, error) {

	raw := ctx.GetRaw()

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

func SetRaw(ctx message.Context, data json.RawMessage) error {
	if IsCapnpCapable() {
		var temp map[string]interface{}
		err := json.Unmarshal(data, &temp)
		if err != nil {
			return err
		}
		robotInfo, err := GetRobotInfo()
		if err != nil {
			return err
		}
		for key, value := range temp {
			value, err := robocapnp.Serialize(value, robotInfo, key)
			if err != nil {
				return err
			}
			temp[key] = value
		}
		data, err = json.Marshal(temp)
		if err != nil {
			return err
		}
	}

	ctx.SetRaw(data)
	return nil
}
