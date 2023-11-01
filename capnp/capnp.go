package robocapnp

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"capnproto.org/go/capnp/v3"
)

const ROBOMOTION_CAPNP_PREFIX = "robomotion-capnp-"

// var CAPNP_LIMIT = 4 << 10 //4KB
var CAPNP_LIMIT = 5

func WriteToFile(value interface{}, robotInfo map[string]interface{}) (interface{}, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if len(data) < CAPNP_LIMIT {
		return value, nil
	}
	robotID := robotInfo["id"].(string)
	cacheDir := robotInfo["cache_dir"].(string)

	msg, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}
	// Create a new Book struct.  Every message must have a root struct.
	nodeMessage, err := NewRootNodeMessage(seg)
	if err != nil {
		return nil, err
	}

	err = nodeMessage.SetContent(data)
	if err != nil {
		return nil, err
	}
	dir := path.Join(cacheDir, "temp", "robots", robotID)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}
	file, err := os.CreateTemp(dir, "robomotion-capnp")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	err = capnp.NewEncoder(file).Encode(msg)
	if err != nil {
		return nil, err
	}
	result := fmt.Sprintf("%s%s", ROBOMOTION_CAPNP_PREFIX, hex.EncodeToString([]byte(file.Name())))
	return result, nil

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
