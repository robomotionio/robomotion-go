package runtime

import "fmt"

type Credential struct {
	VaultID string `json:"vaultId"`
	ItemID  string `json:"itemId"`
}

func (c *Credential) Get() (map[string]interface{}, error) {
	if client == nil {
		return nil, fmt.Errorf("Runtime was not initialized")
	}

	return client.GetVaultItem(c.VaultID, c.ItemID)
}
