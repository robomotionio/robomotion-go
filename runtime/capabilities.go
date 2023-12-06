package runtime

import (
	"math"

	"github.com/robomotionio/robomotion-go/utils"
)

type Capability uint64

const MINIMUM_ROBOT_VERSION = "23.12.0"
const (
	CapabilityLMO Capability = (1 << iota)
)

var (
	capabilities = []Capability{}
	//robotInfo    map[string]interface{}
)

func IsLMOCapable() bool {
	robotInfo, err := GetRobotInfo()
	if err != nil {
		return false
	}
	if lmoFlag, _ := robotInfo["lmo_enabled"].(bool); lmoFlag {
		if version, ok := robotInfo["version"].(string); ok {
			if !utils.IsVersionLessThan(version, MINIMUM_ROBOT_VERSION) {
				return true
			}
		}
	}

	return false
}
func AddCapability(capability Capability) {
	capabilities = append(capabilities, capability)
}

func init() {
	AddCapability(CapabilityLMO)

}

func Capabilites() uint64 {
	var _capabilities uint64 = math.MaxUint64
	for _, cap := range capabilities {
		_capabilities &= uint64(cap)
	}
	return _capabilities
}
