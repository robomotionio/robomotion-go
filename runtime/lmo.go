package runtime

import (
	"bytes"
	"encoding/base32"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"

	"github.com/robomotionio/robomotion-go/utils"
)

const (
	LMO_MAGIC   = 0x1343B7E
	LMO_LIMIT   = 256 << 10 /** 256kb **/
	LMO_VERSION = 0x01
	LMO_HEAD    = 100
)

var (
	enableLMO = false
)

func SetLMOFlag(flag bool) {
	enableLMO = flag
}
func GetLMOFlag() bool {
	return enableLMO
}

type LargeMessageObject struct {
	Magic   int         `json:"magic"`
	Version int         `json:"version"`
	ID      string      `json:"id"`
	Head    string      `json:"head"`
	Size    int64       `json:"size"`
	Data    interface{} `json:"data"`
}

// Value extracts the underlying data from a LargeMessageObject after unmarshalling it.
func (lmo *LargeMessageObject) Value() interface{} {
	return lmo.Data
}

func NewId() string {
	var encoding = base32.NewEncoding("ybndrfg8ejkmcpqxot1uwisza345h769")
	var b bytes.Buffer
	encoder := base32.NewEncoder(encoding, &b)
	encoder.Write([]byte(uuid.New().String()))
	encoder.Close()
	b.Truncate(26) // removes the '==' padding
	return b.String()
}

// SerializeLMO converts a value to a LargeMessageObject if its size exceeds the LMO_LIMIT.
// It saves the serialized object into a file with a unique ID within a robot-specific directory.
func SerializeLMO(value interface{}) (*LargeMessageObject, error) {
	if !IsLMOCapable() {
		return nil, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	dataLen := len(data)
	if dataLen < LMO_LIMIT {
		return nil, nil
	}
	id := NewId()
	lmo := &LargeMessageObject{
		Magic:   LMO_MAGIC,
		Version: LMO_VERSION,
		ID:      id,
		Head:    string(data[0:LMO_HEAD]),
		Size:    int64(len(data)),
		Data:    value,
	}
	robotInfo, err := GetRobotInfo()
	if err != nil {
		//HANDLE ERROR
	}
	robotID := ""
	var ok bool
	if robotID, ok = robotInfo["id"].(string); !ok {
		return nil, nil
	}

	tempPath := utils.GetTempPath()
	dir := path.Join(tempPath, "robots", robotID)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}
	file, err := os.Create(path.Join(dir, id+".lmo"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lmoJSON, err := json.Marshal(lmo)
	if err != nil {
		return nil, err
	}

	_, err = file.Write(lmoJSON)
	if err != nil {
		return nil, err
	}

	lmo.Data = nil

	return lmo, nil

}

// DeserializeLMO reads a file identified by the given ID and unmarshals its content
// back into a LargeMessageObject.
func DeserializeLMO(id string) (*LargeMessageObject, error) {
	robotID := ""
	var ok bool
	if robotID, ok = robotInfo["id"].(string); !ok {
		return nil, nil
	}

	tempPath := utils.GetTempPath()
	dir := path.Join(tempPath, "robots", robotID)
	filePath := path.Join(dir, id+".lmo")

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileContent, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	lmo := &LargeMessageObject{}
	err = json.Unmarshal(fileContent, &lmo)
	if err != nil {
		return nil, err
	}

	return lmo, nil
}

// PackMessageBytes checks if the input message needs packing based on
// system capabilities and size constraints.
func PackMessageBytes(inMsg []byte) ([]byte, error) {
	if !IsLMOCapable() || len(inMsg) < LMO_LIMIT {
		return inMsg, nil
	}

	var msg map[string]interface{}
	err := json.Unmarshal(inMsg, &msg)
	if err != nil {
		return nil, err
	}

	err = PackMessage(msg)
	if err != nil {
		return nil, err
	}

	packed, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return packed, nil
}

// PackMessage iterates over a message map and serializes any values that qualify as
// LargeMessageObjects, replacing the original value in the map with the serialized object.
func PackMessage(msg map[string]interface{}) error {
	if !IsLMOCapable() {
		return nil
	}

	for key, value := range msg {
		lmo, e := SerializeLMO(value)
		if e != nil {
			return e
		}
		if lmo != nil {
			msg[key] = lmo
		} else {
			msg[key] = value
		}
	}

	return nil
}

// UnpackMessageBytes takes a byte slice (inMsg) containing a
// JSON-encoded message and a map (msg) to store the unmarshaled data.
func UnpackMessageBytes(inMsg []byte) ([]byte, error) {
	var msg = make(map[string]interface{})
	err := UnpackMessage(inMsg, msg)
	if err != nil {
		return nil, err
	}

	return json.Marshal(msg)
}

// UnpackMessage takes a byte slice of a message, unmarshals it into a map, and then
// deserializes any LargeMessageObjects within it, replacing the map entries with the
// deserialized values.
func UnpackMessage(inMsg []byte, msg map[string]interface{}) error {
	if !IsLMOCapable() {
		return nil
	}

	if err := json.Unmarshal(inMsg, &msg); err != nil {
		return err
	}

	for key, value := range msg {

		lmo, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		if magicValue, ok := lmo["magic"].(float64); !ok || int64(magicValue) != LMO_MAGIC {
			continue
		}

		idValue, ok := lmo["id"].(string)
		if !ok {
			continue
		}

		result, err := DeserializeLMO(idValue)
		if err != nil {
			return err
		}
		msg[key] = result.Data
	}

	return nil
}

// IsLMO checks if the provided gjson.Result represents a Large Message Object (LMO).
// It first determines if the system has the capability to handle LMOs and then verifies if the value
// is of JSON type with the correct "magic" number identifier specific to LMOs.
func IsLMO(value gjson.Result) bool {
	if !IsLMOCapable() || value.Type != gjson.JSON {
		return false
	}

	return int64(gjson.Get(value.Raw, "magic").Float()) == LMO_MAGIC
}

func DeleteLMObyID(id string) {
	robotID := ""
	var ok bool
	if robotID, ok = robotInfo["id"].(string); !ok {
		log.Println("id not found")
		return
	}
	tempPath := utils.GetTempPath()

	dir := path.Join(tempPath, "robots", robotID, id)
	os.Remove(dir)
}
