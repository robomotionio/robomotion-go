package runtime

import (
	"fmt"

	"github.com/magiconair/properties"
)

var (
	Props     = &properties.Properties{}
	robotInfo map[string]interface{}
)

func GetProps() {
	props, _ := properties.LoadFile("${HOME}/.config/robomotion/config.properties", properties.UTF8)
	if props != nil {
		Props = props
	}
}

func GetRobotInfo() (map[string]interface{}, error) {
	if client == nil {
		return nil, fmt.Errorf("Runtime was not initialized")
	}
	var err error
	if len(robotInfo) == 0 {
		robotInfo, err = client.GetRobotInfo()
	}
	return robotInfo, err
}

func GetRobotVersion() (string, error) {
	info, err := GetRobotInfo()
	if err != nil {
		return "", nil
	}

	v, ok := info["version"].(string)
	if !ok {
		return "", fmt.Errorf("robot version not found")
	}

	return v, nil
}

func GetRobotID() (string, error) {
	info, err := GetRobotInfo()
	if err != nil {
		return "", nil
	}

	v, ok := info["id"].(string)
	if !ok {
		return "", fmt.Errorf("robot id not found")
	}

	return v, nil
}
