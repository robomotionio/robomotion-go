package runtime

type Capability uint64

const (
	CapabilityLMO Capability = 1 << iota
)

var packageCapabilities Capability = CapabilityLMO

func IsLMOCapable() (isCapable bool) {

	robotInfo, err := GetRobotInfo()
	if err != nil {
		return false
	}

	robotCapabilities, ok := robotInfo["capabilities"].(map[string]interface{})
	if !ok {
		return false
	}

	lmo, _ := robotCapabilities["lmo"].(bool)
	return lmo
}
