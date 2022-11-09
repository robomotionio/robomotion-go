package runtime

import (
	"fmt"

	"github.com/magiconair/properties"
)

var (
	Props = &properties.Properties{}
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

	return client.GetRobotInfo()
}
