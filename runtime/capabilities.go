package runtime

import (
	"math"

	robocapnp "github.com/robomotionio/robomotion-go/capnp"
	"github.com/robomotionio/robomotion-go/utils"
)

type Capability uint64

const (
	CapabilityCapnp Capability = (1 << iota)
)

var (
	capabilities = []Capability{}
	robotInfo    map[string]interface{}
)

func IsCapnpCapable() bool {
	if len(robotInfo) == 0 {
		var err error
		robotInfo, err = GetRobotInfo()
		if err != nil {
			return false
		}
	}
	if version, ok := robotInfo["version"].(string); ok {
		if !utils.IsVersionLessThan(version, robocapnp.MINIMUM_ROBOT_VERSION) {
			return true
		}
	}
	return false
}
func AddCapability(capability Capability) {
	capabilities = append(capabilities, capability)
}

func init() {
	AddCapability(CapabilityCapnp)

}

func Capabilites() uint64 {
	var _capabilities uint64 = math.MaxUint64
	for _, cap := range capabilities {
		_capabilities &= uint64(cap)
	}
	return _capabilities
}
