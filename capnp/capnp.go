package robocapnp

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"capnproto.org/go/capnp/v3"
	"github.com/robomotionio/robomotion-go/utils"
)

const ROBOMOTION_CAPNP_PREFIX = "robomotion-capnp"
const ROBOMOTION_CAPNP_ID = "robomotion_capnp_id"

var CAPNP_LIMIT = 4 << 10 //4KB

const MINIMUM_ROBOT_VERSION = "23.11.1"

func Serialize(value interface{}, robotInfo map[string]interface{}, varName string) (interface{}, error) {

	var (
		version  string
		robotID  string
		cacheDir string
		ok       bool
	)

	//For backward compatibility, we dont serialize in the older robots
	version, ok = robotInfo["version"].(string)
	if !ok || utils.IsVersionLessThan(version, MINIMUM_ROBOT_VERSION) {
		return value, nil
	}

	if robotID, ok = robotInfo["id"].(string); !ok {
		return value, nil
	}
	if cacheDir, ok = robotInfo["cache_dir"].(string); !ok {
		return value, nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return value, err
	}

	if len(data) < CAPNP_LIMIT { //If the value is small enough, dont serialize it
		return value, nil
	}

	//Set content for capnp
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return value, err
	}
	nodeMessage, err := NewRootNodeMessage(seg)
	if err != nil {
		return value, err
	}

	err = nodeMessage.SetContent(data)
	if err != nil {
		return value, err
	}

	//Prepare path
	dir := path.Join(cacheDir, "temp", "robots", robotID, varName)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return value, err
	}
	file, err := os.CreateTemp(dir, ROBOMOTION_CAPNP_PREFIX)
	if err != nil {
		return value, err
	}
	defer file.Close()

	//Write to the path
	err = capnp.NewEncoder(file).Encode(msg)
	if err != nil {
		return value, err
	}

	dataLen := len(data)
	//bu kısım küçük datalarla çalışmak için ekledin. Burası silinecek
	dataLimit := 100
	if dataLen < 100 {
		dataLimit = dataLen
	}
	//silinecek

	_data := fmt.Sprintf("%+s...+ %dkb", string(data[0:dataLimit]), len(data)/10)               //user will show only some part
	id := fmt.Sprintf("%s%s", ROBOMOTION_CAPNP_PREFIX, hex.EncodeToString([]byte(file.Name()))) //Points the place whole body is stored
	result := map[string]interface{}{
		ROBOMOTION_CAPNP_ID: id,
		"data":              _data}
	return result, nil

}

func Deserialize(id string) (interface{}, error) {

	//Obtain the path that the data serialized
	id = strings.TrimPrefix(id, ROBOMOTION_CAPNP_PREFIX)
	temp, err := hex.DecodeString(id)
	if err != nil {
		return nil, err
	}
	path := string(temp)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	//Read message
	msg, err := capnp.NewDecoder(file).Decode()
	if err != nil {
		return nil, err
	}

	nodeMessage, err := ReadRootNodeMessage(msg)
	if err != nil {
		return nil, err
	}

	data, err := nodeMessage.Content()
	if err != nil {
		return nil, err
	}
	var result interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}
