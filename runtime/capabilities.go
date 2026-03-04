package runtime

type Capability uint64

const (
	_CapabilityLMOv1Reserved Capability = 1 << iota // bit 0: RESERVED — old file-based LMO, do not reuse
	CapabilityIgnoreVersionCheck                     // bit 1
	CapabilityTerminateOnStop                        // bit 2
	CapabilityUseS3                                  // bit 3
	CapabilityLMO                                    // bit 4: content-addressed blob store
)

var (
	robotCapabilities   uint64     = 0
	packageCapabilities Capability = CapabilityLMO
)

// GetCapabilities returns the intersection of robot and package capabilities.
func GetCapabilities() uint64 {
	return robotCapabilities & uint64(packageCapabilities)
}

// HasCapability returns true when both robot and package support the capability.
func HasCapability(capability Capability) bool {
	return (GetCapabilities() & uint64(capability)) > 0
}

func HasRobotCapability(capability Capability) bool {
	return (robotCapabilities & uint64(capability)) > 0
}

func GetRobotCapabilities() uint64 {
	return robotCapabilities
}

func SetRobotCapabilities(cap uint64) {
	robotCapabilities = cap
}

func SetPackageCapabilities(cap uint64) {
	packageCapabilities = Capability(cap)
}

func SetPackageCapability(cap Capability) {
	packageCapabilities |= cap
}

func SetRobotCapability(cap Capability) {
	robotCapabilities |= uint64(cap)
}
