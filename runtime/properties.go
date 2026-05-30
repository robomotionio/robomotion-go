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

// IsSetupManaged reports whether the host manages interactive setup for THIS
// run — i.e. a setup surface (the Agent Hub hire wizard / OnSetup) exists, so a
// node should NOT fall back to in-flow pairing (e.g. the WhatsApp/Telegram
// Receive flow-run QR/token prompt). The deskbot sets it (from AgentHubSlug)
// in the per-run GetRobotInfo payload, alongside flow_id. This is run CONTEXT,
// not a capability: it flips per run on the same robot (hub hire vs custom
// flow). False when absent (older robots / custom & designer flows) so the
// legacy in-flow pairing remains the default there.
func IsSetupManaged() bool {
	info, err := GetRobotInfo()
	if err != nil {
		return false
	}
	switch v := info["setup_managed"].(type) {
	case bool:
		return v
	case float64: // protobuf Struct encodes some scalars as float64
		return v != 0
	default:
		return false
	}
}
