package runtime

import "fmt"

type InstanceAccess struct {
	AmqEndpoint string `json:"amq_endpoint"`
	ApiEndpoint string `json:"api_endpoint"`
	AccessToken string `json:"access_token"`
}

func GetInstanceAccess() (*InstanceAccess, error) {
	if client == nil {
		return nil, fmt.Errorf("Runtime was not initialized")
	}

	return client.GetInstanceAccess()
}
