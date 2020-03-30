package runtime

import (
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
