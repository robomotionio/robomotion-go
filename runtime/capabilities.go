package runtime

type Capability uint64

const (
	CapabilityLMO Capability = 1 << iota
	CapabilityIgnoreVersionCheck
	CapabilityTerminateOnStop
	CapabilityUseS3
	CapabilityChunkedTransfer
)

var packageCapabilities Capability = CapabilityLMO | CapabilityChunkedTransfer

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

// IsChunkedTransferCapable checks if both the robot and the package support chunked transfer.
// This is used to determine whether large messages should be chunked for transfer.
func IsChunkedTransferCapable() bool {
	robotInfo, err := GetRobotInfo()
	if err != nil {
		return false
	}

	robotCapabilities, ok := robotInfo["capabilities"].(map[string]interface{})
	if !ok {
		return false
	}

	chunked, _ := robotCapabilities["chunkedTransfer"].(bool)
	return chunked
}
