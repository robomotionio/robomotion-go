package runtime

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/robomotionio/robomotion-go/message"
)

type Credential struct {
	Scope string      `json:"scope"`
	Name  interface{} `json:"name"`

	// Deprecated
	VaultID string `json:"vaultId,omitempty"`
	// Deprecated
	ItemID string `json:"itemId,omitempty"`
}

type credential struct {
	VaultID string `json:"vaultId"`
	ItemID  string `json:"itemId"`
}

func (c *Credential) Set(ctx message.Context, data []byte) (map[string]interface{}, error) {
	if client == nil {
		return nil, fmt.Errorf("Runtime was not initialized")
	}

	if c.VaultID != "" && c.ItemID != "" {
		return client.SetVaultItem(c.VaultID, c.ItemID, data)
	}

	var (
		err   error
		ci    interface{} = c.Name
		creds credential
	)

	if c.Scope == "Message" {
		v := &InVariable{Variable: Variable{Scope: c.Scope, Name: c.Name.(string)}}
		ci, err = v.Get(ctx)
		if err != nil {
			return nil, err
		}
	}

	err = mapstructure.Decode(ci, &creds)
	if err != nil {
		return nil, err
	}

	return client.SetVaultItem(creds.VaultID, creds.ItemID, data)
}

func (c *Credential) Get(ctx message.Context) (map[string]interface{}, error) {
	if client == nil {
		return nil, fmt.Errorf("Runtime was not initialized")
	}

	if c.VaultID != "" && c.ItemID != "" {
		return client.GetVaultItem(c.VaultID, c.ItemID)
	}

	var (
		err   error
		ci    interface{} = c.Name
		creds credential
	)

	if c.Scope == "Message" {
		v := &InVariable{Variable: Variable{Scope: c.Scope, Name: c.Name.(string)}}
		ci, err = v.Get(ctx)
		if err != nil {
			return nil, err
		}
	}

	err = mapstructure.Decode(ci, &creds)
	if err != nil {
		return nil, err
	}

	return client.GetVaultItem(creds.VaultID, creds.ItemID)
}
