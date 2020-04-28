package runtime

type Variable struct {
	Scope string `json:"scope"`
	Name  string `json:"name"`
}

type Credentials struct {
	VaultID string `json:"vaultId"`
	ItemID  string `json:"itemId"`
}
