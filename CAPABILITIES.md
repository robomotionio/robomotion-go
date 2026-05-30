# Capability bits — canonical registry

Capabilities are a single `uint64` bitfield negotiated between the **robot**
(deskbot) and a **package** (built on this SDK). The value crosses the wire raw
(in `GetRobotInfo.capabilities` for robot→package, and the `GetCapabilities`
RPC for package→robot), so **a bit must mean the same thing on both sides.**

The enum is hand-duplicated in two repos with no compile-time link:

- `robomotion-go/runtime/capabilities.go` (this repo — packages)
- `robomotion-deskbot/runtime/capabilities.go` (the robot)

**Any change here MUST be mirrored there (and vice-versa).** This file is the
source of truth for the allocation.

## Allocation

| bit | value | name | direction | meaning |
|----:|------:|------|-----------|---------|
| 0 | 1 | _LMOv1Reserved | — | reserved, old file-based LMO; do not reuse |
| 1 | 2 | IgnoreVersionCheck | robot→package | skip node version gate |
| 2 | 4 | TerminateOnStop | robot→package | kill package process on stop |
| 3 | 8 | UseS3 | robot→package | flow assets/artifacts via S3 |
| 4 | 16 | LMO | both (AND) | content-addressed large-message blob store |
| 5 | 32 | Diagnostics | robot→package | deskbot stderr diagnostics |
| 6 | 64 | LazyMessage | robot→package | Go-backed lazy message bridge (Function node) |
| 7+ | — | (free) | — | next available |

Only **LMO (bit 4)** is intersected via `HasCapability` (robot AND package).
Everything else is one-directional (`HasRobotCapability`, or the runner reading
the package's advertised bits, which today only feeds a `[feat:…]` log label and
the LMO AND).

## Notes / history

- **bit 5 collision + `Setup` removal (v1.20.0).** robomotion-go v1.19.0 had a
  `Setup` bit at **bit 5** that collided with the deskbot's **`Diagnostics`
  (bit 5)** / **`LazyMessage` (bit 6)** — the two enums had drifted because the
  OnSetup work (SDK) and the diagnostics/lazy-message work (deskbot)
  independently grabbed the next free bit in separate repos. The `Setup` bit was
  **advertised but never consumed** — OnSetup is gated by the `SetupHandler`
  interface (at OnSetup-RPC time) + RPC-unimplemented for old packages, not by a
  capability bit. So v1.20.0 **drops `Setup` entirely** (rather than relocating
  dead weight) and the SDK adopts `Diagnostics`(5)/`LazyMessage`(6) to match the
  deskbot. Nets to: no `Setup` bit, enums aligned, bit 7+ free.

- **`setup_managed` is NOT a capability.** Whether interactive setup is
  host-managed for a run (Agent Hub hire vs custom flow) is run *context*, not a
  robot ability — it flips per run on the same robot. It travels as a field in
  the per-run `GetRobotInfo` payload (next to `flow_id`) and is read via
  `runtime.IsSetupManaged()`. Do **not** allocate a capability bit for it.
