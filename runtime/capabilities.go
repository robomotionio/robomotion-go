package runtime

type Capability uint64

// Capability bit allocation. THIS MUST STAY IN SYNC WITH
// robomotion-deskbot/runtime/capabilities.go — the deskbot and SDK each keep a
// hand-maintained copy and the value crosses the wire as a raw uint64, so a bit
// that means one thing here and another there silently mis-negotiates. See
// CAPABILITIES.md for the canonical registry.
//
// bit-5 history: v1.19.0 had a `Setup` bit at bit 5 that collided with the
// deskbot's Diagnostics(5)/LazyMessage(6). It was DROPPED in v1.20.0 — it was
// advertised but never consumed (OnSetup is gated by the SetupHandler interface
// + RPC-unimplemented, not a bit), so it was pure dead weight. The SDK now
// adopts the deskbot's Diagnostics(5)/LazyMessage(6) so the enums align; bit 7+
// are free.
const (
	_CapabilityLMOv1Reserved Capability = 1 << iota // bit 0: RESERVED — old file-based LMO, do not reuse
	CapabilityIgnoreVersionCheck                     // bit 1
	CapabilityTerminateOnStop                        // bit 2
	CapabilityUseS3                                  // bit 3
	CapabilityLMO                                    // bit 4: content-addressed blob store
	CapabilityDiagnostics                            // bit 5: deskbot stderr diagnostics (robot→package; defined for parity)
	CapabilityLazyMessage                            // bit 6: deskbot Go-backed lazy msg bridge (robot→package; defined for parity)
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
