package runtime

import "github.com/robomotionio/robomotion-go/proto"

// TestRuntimeHelper is a minimal interface for testing that only requires
// the methods commonly used in tests.
type TestRuntimeHelper interface {
	GetVaultItem(vaultID, itemID string) (map[string]interface{}, error)
	SetVaultItem(vaultID, itemID string, data []byte) (map[string]interface{}, error)
}

// testClient wraps a TestRuntimeHelper to implement the full RuntimeHelper interface.
type testClient struct {
	helper TestRuntimeHelper
}

// SetTestClient sets a test client for unit testing.
// This allows mocking credential and vault operations.
//
// Example:
//
//	type MockHelper struct{}
//	func (m *MockHelper) GetVaultItem(vaultID, itemID string) (map[string]interface{}, error) {
//	    return map[string]interface{}{"value": "test-api-key"}, nil
//	}
//	func (m *MockHelper) SetVaultItem(vaultID, itemID string, data []byte) (map[string]interface{}, error) {
//	    return nil, nil
//	}
//
//	runtime.SetTestClient(&MockHelper{})
func SetTestClient(helper TestRuntimeHelper) {
	client = &testClient{helper: helper}
}

// ClearTestClient clears the test client.
func ClearTestClient() {
	client = nil
}

func (t *testClient) Close() error { return nil }

func (t *testClient) Debug(string, string, interface{}) error { return nil }

func (t *testClient) EmitFlowEvent(string, string) error { return nil }

func (t *testClient) EmitInput(string, []byte) error { return nil }

func (t *testClient) EmitOutput(string, []byte, int32) error { return nil }

func (t *testClient) EmitError(string, string, string) error { return nil }

func (t *testClient) GetVaultItem(vaultID, itemID string) (map[string]interface{}, error) {
	return t.helper.GetVaultItem(vaultID, itemID)
}

func (t *testClient) SetVaultItem(vaultID, itemID string, data []byte) (map[string]interface{}, error) {
	return t.helper.SetVaultItem(vaultID, itemID, data)
}

func (t *testClient) GetVariable(v *variable) (interface{}, error) {
	return nil, nil
}

func (t *testClient) SetVariable(v *variable, value interface{}) error {
	return nil
}

func (t *testClient) GetRobotInfo() (map[string]interface{}, error) {
	return map[string]interface{}{
		"robotId":   "test-robot",
		"robotName": "Test Robot",
	}, nil
}

func (t *testClient) AppRequest([]byte, int32) ([]byte, error) { return nil, nil }

func (t *testClient) AppRequestV2([]byte) ([]byte, error) { return nil, nil }

func (t *testClient) AppPublish([]byte) error { return nil }

func (t *testClient) AppDownload(string, string, string) (string, error) { return "", nil }

func (t *testClient) AppUpload(string, string) (string, error) { return "", nil }

func (t *testClient) GatewayRequest(string, string, string, map[string]string) (*proto.GatewayRequestResponse, error) {
	return nil, nil
}

func (t *testClient) ProxyRequest(*proto.HttpRequest) (*proto.HttpResponse, error) {
	return nil, nil
}

func (t *testClient) IsRunning() (bool, error) { return true, nil }

func (t *testClient) GetPortConnections(string, int) ([]NodeInfo, error) { return nil, nil }

func (t *testClient) GetInstanceAccess() (*InstanceAccess, error) { return nil, nil }
