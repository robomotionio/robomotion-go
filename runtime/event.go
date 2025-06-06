package runtime

import (
	"fmt"

	"github.com/robomotionio/robomotion-go/proto"
)

func EmitDebug(guid, name string, message interface{}) error {
	if client == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return client.Debug(guid, name, message)
}
func EmitOutput(guid string, output []byte, port int32) error {
	if client == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return client.EmitOutput(guid, output, port)
}
func EmitInput(guid string, input []byte) error {
	if client == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return client.EmitInput(guid, input)
}
func EmitError(guid, name, message string) error {
	if client == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return client.EmitError(guid, name, message)
}
func EmitFlowEvent(guid, name string) error {
	if client == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return client.EmitFlowEvent(guid, name)
}
func AppRequest(request []byte, timeout int32) ([]byte, error) {
	if client == nil {
		return nil, fmt.Errorf("Runtime was not initialized")
	}

	return client.AppRequest(request, timeout)
}
func AppRequestV2(request []byte) ([]byte, error) {
	if client == nil {
		return nil, fmt.Errorf("Runtime was not initialized")
	}

	return client.AppRequestV2(request)
}
func AppPublish(request []byte) error {
	if client == nil {
		return fmt.Errorf("Runtime was not initialized")
	}

	return client.AppPublish(request)
}
func AppDownload(id, dir, file string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("Runtime was not initialized")
	}

	return client.AppDownload(id, dir, file)
}
func AppUpload(id, path string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("Runtime was not initialized")
	}

	return client.AppUpload(id, path)
}
func GatewayRequest(method, endpoint, body string, headers map[string]string) (*proto.GatewayRequestResponse, error) {
	if client == nil {
		return nil, fmt.Errorf("Runtime was not initialized")
	}

	return client.GatewayRequest(method, endpoint, body, headers)
}

func ProxyRequest(req *proto.HttpRequest) (*proto.HttpResponse, error) {
	if client == nil {
		return nil, fmt.Errorf("Runtime was not initialized")
	}

	return client.ProxyRequest(req)
}

func IsRunning() (bool, error) {
	if client == nil {
		return false, fmt.Errorf("Runtime was not initialized")
	}

	return client.IsRunning()
}

func GetPortConnections(guid string, port int) ([]NodeInfo, error) {
	if client == nil {
		return nil, fmt.Errorf("Runtime was not initialized")
	}

	return client.GetPortConnections(guid, port)
}
