package runtime

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/robomotionio/robomotion-go/proto"
)

// CLIRuntimeHelper implements RuntimeHelper for CLI mode without gRPC.
// In CLI mode, all variables use Message scope and resolve from message.Context,
// so GetVariable/SetVariable are no-ops. Credentials come from vault flags.
type CLIRuntimeHelper struct {
	credentials map[string]interface{} // populated from vault fetch
}

// NewCLIRuntimeHelper creates a CLIRuntimeHelper.
func NewCLIRuntimeHelper() *CLIRuntimeHelper {
	return &CLIRuntimeHelper{}
}

// SetCredentials sets the credential map (e.g. from vault fetch).
func (c *CLIRuntimeHelper) SetCredentials(creds map[string]interface{}) {
	c.credentials = creds
}

// --- RuntimeHelper interface implementation ---

func (c *CLIRuntimeHelper) Close() error { return nil }

func (c *CLIRuntimeHelper) Debug(guid, name string, message interface{}) error {
	// Log debug messages to stderr in CLI mode
	data, _ := json.Marshal(message)
	fmt.Fprintf(os.Stderr, "[debug] %s: %s\n", name, string(data))
	return nil
}

func (c *CLIRuntimeHelper) EmitFlowEvent(guid, name string) error { return nil }

func (c *CLIRuntimeHelper) EmitInput(guid string, input []byte) error { return nil }

func (c *CLIRuntimeHelper) EmitOutput(guid string, output []byte, port int32) error { return nil }

func (c *CLIRuntimeHelper) EmitError(guid, name, message string) error {
	fmt.Fprintf(os.Stderr, "[error] %s: %s\n", name, message)
	return nil
}

func (c *CLIRuntimeHelper) GetVaultItem(vaultID, itemID string) (map[string]interface{}, error) {
	if c.credentials != nil {
		return c.credentials, nil
	}
	return nil, fmt.Errorf("no credentials available: use --vault-id/--item-id flags")
}

func (c *CLIRuntimeHelper) SetVaultItem(vaultID, itemID string, data []byte) (map[string]interface{}, error) {
	return nil, fmt.Errorf("SetVaultItem not supported in CLI mode")
}

// GetVariable is a no-op in CLI mode — variables resolve from message.Context directly.
func (c *CLIRuntimeHelper) GetVariable(v *variable) (interface{}, error) {
	return nil, nil
}

// SetVariable is a no-op in CLI mode — variables resolve from message.Context directly.
func (c *CLIRuntimeHelper) SetVariable(v *variable, val interface{}) error {
	return nil
}

func (c *CLIRuntimeHelper) GetRobotInfo() (map[string]interface{}, error) {
	return map[string]interface{}{
		"id":      "cli",
		"flow_id": "cli",
	}, nil
}

func (c *CLIRuntimeHelper) AppRequest(request []byte, timeout int32) ([]byte, error) {
	return nil, fmt.Errorf("AppRequest not supported in CLI mode")
}

func (c *CLIRuntimeHelper) AppRequestV2(request []byte) ([]byte, error) {
	return nil, fmt.Errorf("AppRequestV2 not supported in CLI mode")
}

func (c *CLIRuntimeHelper) AppPublish(request []byte) error {
	return fmt.Errorf("AppPublish not supported in CLI mode")
}

func (c *CLIRuntimeHelper) AppDownload(id, dir, file string) (string, error) {
	return "", fmt.Errorf("AppDownload not supported in CLI mode")
}

func (c *CLIRuntimeHelper) AppUpload(id, path string) (string, error) {
	return "", fmt.Errorf("AppUpload not supported in CLI mode")
}

func (c *CLIRuntimeHelper) GatewayRequest(method, endpoint, body string, headers map[string]string) (*proto.GatewayRequestResponse, error) {
	return nil, fmt.Errorf("GatewayRequest not supported in CLI mode")
}

func (c *CLIRuntimeHelper) ProxyRequest(req *proto.HttpRequest) (*proto.HttpResponse, error) {
	return nil, fmt.Errorf("ProxyRequest not supported in CLI mode")
}

func (c *CLIRuntimeHelper) IsRunning() (bool, error) {
	return true, nil
}

func (c *CLIRuntimeHelper) GetPortConnections(guid string, port int) ([]NodeInfo, error) {
	return nil, nil
}

func (c *CLIRuntimeHelper) GetInstanceAccess() (*InstanceAccess, error) {
	return nil, fmt.Errorf("GetInstanceAccess not supported in CLI mode")
}
