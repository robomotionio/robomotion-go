package runtime

import (
	"encoding/json"
	"strings"

	robocapnp "github.com/robomotionio/robomotion-go/capnp"
	"github.com/robomotionio/robomotion-go/message"
)

func GetRaw(ctx message.Context) json.RawMessage {

	raw := ctx.GetRaw()

	if IsCapnpCapable() {
		var msg = make(map[string]interface{})
		err := json.Unmarshal(raw, &msg)
		if err != nil {
			//TODO HANDLE
		}
		for key, value := range msg {
			if capnp, ok := value.(map[string]interface{}); ok && capnp[robocapnp.ROBOMOTION_CAPNP_ID] != nil {

				capnp_id := capnp[robocapnp.ROBOMOTION_CAPNP_ID].(string)
				if strings.HasPrefix(capnp_id, robocapnp.ROBOMOTION_CAPNP_PREFIX) {
					result, err := robocapnp.Deserialize(capnp_id)
					if err != nil {
						//TODO handle
					}
					msg[key] = result

				}
			}

		}
		inMsg, err := json.Marshal(msg)
		if err != nil {
			//TODO Handle
		}
		return json.RawMessage(inMsg)
	}
	return raw
}

func SetRaw(ctx message.Context, data json.RawMessage) {
	if IsCapnpCapable() {
		var temp map[string]interface{}
		err := json.Unmarshal(data, &temp)
		if err != nil {
			//TODO HANDLE ERROR
		}
		robotInfo, err := GetRobotInfo()
		if err != nil {
			//TODO HANDLE ERROR
		}
		for key, value := range temp {
			value, err := robocapnp.Serialize(value, robotInfo, key)
			if err != nil {
				//TODO Handle error
			}
			temp[key] = value
		}
		data, err = json.Marshal(temp)
		if err != nil {
			//TODO Handle error
		}
	}

	ctx.SetRaw(data)
}
