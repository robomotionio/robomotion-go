package robocapnp

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"capnproto.org/go/capnp/v3"
	"github.com/robomotionio/robomotion-go/runtime"
	"github.com/robomotionio/robomotion-go/utils"
)

const ROBOMOTION_CAPNP_PREFIX = "robomotion-capnp"

// var CAPNP_LIMIT = 4 << 10 //4KB
var CAPNP_LIMIT = 50

const MINIMUM_ROBOT_VERSION = "23.10.2"

func WriteToFile(value interface{}, robotInfo map[string]interface{}, varName string) (interface{}, error) {
	info, err := runtime.GetRobotInfo()
	if err != nil {
		return value, nil
	}

	v, ok := info["version"].(string)
	if !ok || utils.IsVersionLessThan(v, MINIMUM_ROBOT_VERSION) {
		return value, nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	if len(data) < CAPNP_LIMIT {
		return value, nil
	}

	robotID := robotInfo["id"].(string)
	cacheDir := robotInfo["cache_dir"].(string)

	//Set content for capnp
	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}
	nodeMessage, err := NewRootNodeMessage(seg)
	if err != nil {
		return nil, err
	}

	err = nodeMessage.SetContent(data)
	if err != nil {
		return nil, err
	}

	//Prepare path
	dir := path.Join(cacheDir, "temp", "robots", robotID, varName)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}
	file, err := os.CreateTemp(dir, ROBOMOTION_CAPNP_PREFIX)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	//Write to the path
	err = capnp.NewEncoder(file).Encode(msg)
	if err != nil {
		return nil, err
	}

	dataLen := len(data)
	//bu kısım küçük datalarla çalışmak için ekledin. Burası silinecek
	dataLimit := 100
	if dataLen < 100 {
		dataLimit = dataLen
	}
	//silinecek

	cut_result := fmt.Sprintf("%+s...+ %dkb", string(data[0:dataLimit]), len(data)/10)          //user will show only some part
	id := fmt.Sprintf("%s%s", ROBOMOTION_CAPNP_PREFIX, hex.EncodeToString([]byte(file.Name()))) //Points the place whole body is stored
	return map[string]interface{}{"robomotion_capnp_id": id, "is_data_cut": true, "cut_result": cut_result}, nil

}

func ReadFromFile(id string) (interface{}, error) {
	id = strings.TrimPrefix(id, ROBOMOTION_CAPNP_PREFIX)
	temp, err := hex.DecodeString(id)
	if err != nil {
		return "", err
	}
	path := string(temp)
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}

	msg, err := capnp.NewDecoder(file).Decode()
	if err != nil {
		return "", err
	}

	nodeMessage, err := ReadRootNodeMessage(msg)
	if err != nil {
		return "", err
	}

	data, err := nodeMessage.Content()
	if err != nil {
		return "", nil
	}
	var result interface{}
	json.Unmarshal(data, &result)
	return result, nil
}
