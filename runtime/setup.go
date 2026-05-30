package runtime

import (
	"context"
	"encoding/json"
	"fmt"
)

// SetupHandler is the OPTIONAL lifecycle a node implements to support an
// interactive setup action — e.g. WhatsApp QR pairing — driven from the UI
// during hire setup, OUTSIDE any flow run. The host (runner) instantiates the
// node from its config (exactly like OnCreate) and then invokes OnSetup via
// the Node.OnSetup RPC. OnSetup may block for the setup's whole duration while
// pushing prompts with ctx.Emit and (optionally) blocking for user input with
// ctx.Await, then returns. Nodes that don't implement this interface report an
// "unsupported" error and the host falls back.
type SetupHandler interface {
	OnSetup(ctx SetupContext) error
}

// SetupEvent is one prompt/status frame a node pushes to the UI during setup.
// Kind tells the UI how to render; Step mirrors the node's setup state
// machine. The shape is channel-agnostic so future setups (OAuth redirect,
// device code, test-connection) reuse it without a wire change.
type SetupEvent struct {
	Kind      string `json:"kind"`            // "qr" | "code" | "oauth" | "status" | "prompt"
	Step      string `json:"step"`            // "pending" | "completed" | "timeout" | "error"
	Image     string `json:"image,omitempty"` // data URL, e.g. a QR PNG
	Text      string `json:"text,omitempty"`  // code / OAuth URL / value to display
	Message   string `json:"message,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// SetupInputSpec describes input the node is waiting for in Await. Optional —
// no-input setups (WhatsApp QR, where the scan is detected out-of-band) never
// call Await.
type SetupInputSpec struct {
	Prompt string   `json:"prompt,omitempty"`
	Fields []string `json:"fields,omitempty"`
}

// SetupInput is the user-supplied input returned by Await.
type SetupInput struct {
	Values    map[string]string `json:"values,omitempty"`
	Cancelled bool              `json:"cancelled,omitempty"`
}

// SetupContext is handed to OnSetup. It exposes the node config, lets the node
// Emit prompts and optionally Await input, and carries a cancellation Context
// that fires on host timeout/abort.
type SetupContext interface {
	GUID() string
	SessionID() string
	Config() []byte
	Emit(SetupEvent) error
	Await(spec SetupInputSpec, timeoutSec int32) (SetupInput, error)
	Context() context.Context
}

type setupContext struct {
	guid      string
	sessionID string
	config    []byte
	ctx       context.Context
}

func (s *setupContext) GUID() string              { return s.guid }
func (s *setupContext) SessionID() string         { return s.sessionID }
func (s *setupContext) Config() []byte            { return s.config }
func (s *setupContext) Context() context.Context { return s.ctx }

func (s *setupContext) Emit(ev SetupEvent) error {
	if client == nil {
		return fmt.Errorf("runtime not initialized")
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return client.SetupEmit(s.guid, s.sessionID, b)
}

func (s *setupContext) Await(spec SetupInputSpec, timeoutSec int32) (SetupInput, error) {
	var in SetupInput
	if client == nil {
		return in, fmt.Errorf("runtime not initialized")
	}
	b, err := json.Marshal(spec)
	if err != nil {
		return in, err
	}
	raw, timedOut, err := client.SetupAwait(s.guid, s.sessionID, b, timeoutSec)
	if err != nil {
		return in, err
	}
	if timedOut {
		return in, fmt.Errorf("setup input timed out")
	}
	if len(raw) > 0 {
		if uerr := json.Unmarshal(raw, &in); uerr != nil {
			return in, uerr
		}
	}
	return in, nil
}

// AsSetupHandler returns the SetupHandler for a stored handler, unwrapping the
// ToolInterceptor that AddNodeHandler always applies. Returns nil when the
// node doesn't support setup.
func AsSetupHandler(h MessageHandler) SetupHandler {
	if sh, ok := h.(SetupHandler); ok {
		return sh
	}
	if ti, ok := h.(*ToolInterceptor); ok {
		if sh, ok := ti.Unwrap().(SetupHandler); ok {
			return sh
		}
	}
	return nil
}
