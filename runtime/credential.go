package runtime

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/robomotionio/robomotion-go/message"
)

type Credential struct {
	Scope string `json:"scope"`
	Name  string `json:"name"`

	// Deprecated
	VaultID string `json:"vaultId,omitempty"`
	// Deprecated
	ItemID string `json:"itemId,omitempty"`
}

type credential struct {
	VaultID string `json:"vaultId"`
	ItemID  string `json:"itemId"`
}

func (c *Credential) Get(ctx message.Context) (map[string]interface{}, error) {
	if client == nil {
		return nil, fmt.Errorf("Runtime was not initialized")
	}

	if c.VaultID != "" && c.ItemID != "" {
		return client.GetVaultItem(c.VaultID, c.ItemID)
	}

	v := &InVariable{Variable: Variable{Scope: c.Scope, Name: c.Name}}
	cr, err := v.Get(ctx)
	if err != nil {
		return nil, err
	}

	var creds credential
	err = mapstructure.Decode(cr, &creds)
	if err != nil {
		return nil, err
	}

	return client.GetVaultItem(creds.VaultID, creds.ItemID)
}
