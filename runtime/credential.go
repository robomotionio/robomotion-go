package runtime

import "fmt"

type Credential struct {
	VaultID string `json:"vaultId"`
	ItemID  string `json:"itemId"`
}

func (c *Credential) Get() (map[string]interface{}, error) {
	if runtimeHelper == nil {
		return nil, fmt.Errorf("Runtime was not initialized")
	}

	return runtimeHelper.GetVaultItem(c.VaultID, c.ItemID)
}
