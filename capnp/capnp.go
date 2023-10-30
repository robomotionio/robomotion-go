package robocapnp

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"capnproto.org/go/capnp/v3"
)

const ROBOMOTION_CAPNP_PREFIX = "robomotion-capnp-"

var CAPNP_LIMIT = 4 << 10 //4KB
func WriteToFile(value interface{}) (interface{}, error) {

	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	if len(data) < CAPNP_LIMIT {
		return value, nil
	}

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

	file, err := os.CreateTemp("", "robomotion-capnproto")
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

func isGreaterThan4KB(data interface{}, limit int) bool {
	// Use reflection to get the underlying value of the interface
	value := reflect.ValueOf(data)

	// Check if the underlying type is a slice or an array and it's larger than 4KB
	if value.Kind() == reflect.Array || value.Kind() == reflect.Slice {
		byteSlice, ok := data.([]byte)
		if ok {
			// Check the length of the byte slice
			return len(byteSlice) > limit // 4KB
		}
	}

	return false
}
