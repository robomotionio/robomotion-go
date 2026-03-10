package runtime

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/pbkdf2"
)

// CLIVaultClient provides direct vault access for CLI mode without gRPC/robot runtime.
// It authenticates with the Robomotion platform and fetches/decrypts vault items.
type CLIVaultClient struct {
	apiBaseURL  string
	accessToken string
	masterKey   []byte
	keySet      *cliKeySet
}

type cliKeySet struct {
	PublicKey     string `json:"pub_key"`
	EncPrivateKey string `json:"enc_priv_key"`
	IV            string `json:"iv"`
}

// cliAuthConfig holds saved auth state from `robomotion login`.
type cliAuthConfig struct {
	APIEndpoint string `json:"api_endpoint"`
	AccessToken string `json:"access_token"`
	MasterKey   string `json:"master_key"`   // hex-encoded
	KeySet      *cliKeySet `json:"key_set"`
}

type vaultItemResponse struct {
	OK   bool   `json:"ok"`
	Data string `json:"data"` // hex-encoded encrypted data
	Item struct {
		ID       string `json:"id"`
		Category int    `json:"category"`
		Name     string `json:"name"`
		IV       string `json:"iv"` // hex-encoded IV
	} `json:"item"`
}

// NewCLIVaultClient creates a vault client from saved auth configuration.
// Auth config is read from ~/.robomotion/auth.json (saved by `robomotion login`).
func NewCLIVaultClient() (*CLIVaultClient, error) {
	configPath := defaultAuthConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("no saved auth found at %s; run 'robomotion login' first: %w", configPath, err)
	}

	var config cliAuthConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid auth config at %s: %w", configPath, err)
	}

	if config.AccessToken == "" {
		return nil, fmt.Errorf("no access token in auth config; run 'robomotion login' again")
	}

	client := &CLIVaultClient{
		apiBaseURL:  config.APIEndpoint,
		accessToken: config.AccessToken,
		keySet:      config.KeySet,
	}

	if config.MasterKey != "" {
		client.masterKey, _ = hex.DecodeString(config.MasterKey)
	}

	if client.apiBaseURL == "" {
		client.apiBaseURL = "https://api.robomotion.io"
	}

	return client, nil
}

// FetchVaultItem fetches and decrypts a vault item by vault ID and item ID.
func (c *CLIVaultClient) FetchVaultItem(vaultID, itemID string) (map[string]interface{}, error) {
	// 1. Fetch encrypted item from API
	endpoint := fmt.Sprintf("/v1/vaults.items.get?vault_id=%s&item_id=%s", vaultID, itemID)
	body, err := c.apiGet(endpoint)
	if err != nil {
		return nil, fmt.Errorf("vault API error: %w", err)
	}

	var resp vaultItemResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("invalid vault response: %w", err)
	}
	if !resp.OK {
		return nil, fmt.Errorf("vault item not found: vault=%s item=%s", vaultID, itemID)
	}

	// 2. Decode encrypted data from hex
	encData, err := hex.DecodeString(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("invalid encrypted data: %w", err)
	}

	// 3. Get vault key and decrypt
	vaultKey, err := c.getVaultKey(vaultID)
	if err != nil {
		return nil, fmt.Errorf("vault key error: %w", err)
	}

	iv := resp.Item.IV
	if iv == "" && c.keySet != nil {
		iv = c.keySet.IV // backward compatibility
	}

	plaintext, err := decryptAESCBC(vaultKey, encData, iv)
	if err != nil {
		return nil, fmt.Errorf("decryption error: %w", err)
	}

	// 4. Parse decrypted JSON into credential map
	var result map[string]interface{}
	if err := json.Unmarshal(plaintext, &result); err != nil {
		return nil, fmt.Errorf("invalid credential data: %w", err)
	}

	return result, nil
}

// getVaultKey derives the AES key for decrypting vault items.
// Flow: decrypt private key with master key → RSA-decrypt vault key → XOR with secret key.
func (c *CLIVaultClient) getVaultKey(vaultID string) ([]byte, error) {
	if c.masterKey == nil {
		return nil, errors.New("master key not available; run 'robomotion login' first")
	}
	if c.keySet == nil {
		return nil, errors.New("key set not available; run 'robomotion login' first")
	}

	// 1. Decrypt private key using master key + keySet IV
	privKeyPEM, err := c.decryptPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("private key decryption failed: %w", err)
	}

	// 2. Parse RSA private key
	block, _ := pem.Decode([]byte(privKeyPEM))
	if block == nil {
		return nil, errors.New("failed to parse private key PEM")
	}
	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// 3. Fetch vault metadata for encrypted vault key
	vault, err := c.fetchVault(vaultID)
	if err != nil {
		return nil, err
	}

	// 4. RSA-OAEP decrypt the vault key
	encVaultKey, err := base64.StdEncoding.DecodeString(vault.EncVaultKey)
	if err != nil {
		return nil, fmt.Errorf("invalid vault key encoding: %w", err)
	}

	hash := sha256.New()
	vaultKey, err := rsa.DecryptOAEP(hash, rand.Reader, privKey, encVaultKey, nil)
	if err != nil {
		return nil, fmt.Errorf("RSA decrypt vault key failed: %w", err)
	}

	// 5. Get and decrypt secret key
	encSecretKey, err := c.getSecretKey(vaultID)
	if err != nil {
		return nil, err
	}

	hash.Reset()
	secretKey, err := rsa.DecryptOAEP(hash, rand.Reader, privKey, encSecretKey, nil)
	if err != nil {
		return nil, fmt.Errorf("RSA decrypt secret key failed: %w", err)
	}

	// 6. XOR vault key with secret key
	if len(secretKey) != len(vaultKey) {
		return nil, errors.New("key length mismatch")
	}
	xored := make([]byte, len(secretKey))
	for i := range secretKey {
		xored[i] = secretKey[i] ^ vaultKey[i]
	}

	return xored, nil
}

// decryptPrivateKey decrypts the stored private key using the master key.
func (c *CLIVaultClient) decryptPrivateKey() (string, error) {
	iv, err := hex.DecodeString(c.keySet.IV)
	if err != nil {
		return "", err
	}
	data, err := hex.DecodeString(c.keySet.EncPrivateKey)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(c.masterKey)
	if err != nil {
		return "", err
	}

	cbc := cipher.NewCBCDecrypter(block, iv)
	cbc.CryptBlocks(data, data)

	return string(pkcs7Unpad(data)), nil
}

type vaultMetadata struct {
	EncVaultKey string `json:"enc_vault_key"`
	IV          string `json:"iv"`
}

type vaultsListResponse struct {
	OK     bool            `json:"ok"`
	Vaults []vaultMetadata `json:"vaults"`
}

// fetchVault gets vault metadata including the encrypted vault key.
func (c *CLIVaultClient) fetchVault(vaultID string) (*vaultMetadata, error) {
	body, err := c.apiGet("/v1/vaults.list")
	if err != nil {
		return nil, err
	}

	var resp vaultsListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	// Find vault by ID in the response
	// The vaults.list response includes vault objects with id fields
	var rawResp struct {
		OK     bool              `json:"ok"`
		Vaults []json.RawMessage `json:"vaults"`
	}
	if err := json.Unmarshal(body, &rawResp); err != nil {
		return nil, err
	}

	for _, raw := range rawResp.Vaults {
		var v struct {
			ID          string `json:"id"`
			EncVaultKey string `json:"enc_vault_key"`
			IV          string `json:"iv"`
		}
		if err := json.Unmarshal(raw, &v); err != nil {
			continue
		}
		if v.ID == vaultID {
			return &vaultMetadata{
				EncVaultKey: v.EncVaultKey,
				IV:          v.IV,
			}, nil
		}
	}

	return nil, fmt.Errorf("vault %s not found", vaultID)
}

// getSecretKey retrieves the encrypted secret key for a vault.
// In the full runtime this comes from the system keyring; for CLI mode
// it's saved in the auth config during `robomotion login`.
func (c *CLIVaultClient) getSecretKey(vaultID string) ([]byte, error) {
	configPath := defaultAuthConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read auth config: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Secret keys stored per vault in auth config
	secrets, ok := config["vault_secrets"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no vault secrets saved; run 'robomotion login' and open the vault first")
	}

	secretHex, ok := secrets[vaultID].(string)
	if !ok {
		return nil, fmt.Errorf("no secret key for vault %s; open the vault first", vaultID)
	}

	return hex.DecodeString(secretHex)
}

// vaultEntry represents a vault with its ID and name.
type vaultEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// itemEntry represents a vault item with its ID and name.
type itemEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ResolveVaultByName lists all vaults and finds one matching the given name.
// Returns the vault ID or an error if no match, or ambiguous (multiple matches).
func (c *CLIVaultClient) ResolveVaultByName(name string) (string, error) {
	body, err := c.apiGet("/v1/vaults.list")
	if err != nil {
		return "", fmt.Errorf("failed to list vaults: %w", err)
	}

	var resp struct {
		OK     bool              `json:"ok"`
		Vaults []json.RawMessage `json:"vaults"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}

	var matches []vaultEntry
	for _, raw := range resp.Vaults {
		var v vaultEntry
		if err := json.Unmarshal(raw, &v); err != nil {
			continue
		}
		if strings.EqualFold(v.Name, name) {
			matches = append(matches, v)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no vault found with name %q", name)
	}
	if len(matches) > 1 {
		lines := []string{fmt.Sprintf("ambiguous vault name %q matches %d vaults:", name, len(matches))}
		for _, m := range matches {
			lines = append(lines, fmt.Sprintf("  --vault-id=%s  (%s)", m.ID, m.Name))
		}
		return "", fmt.Errorf("%s", strings.Join(lines, "\n"))
	}
	return matches[0].ID, nil
}

// ResolveItemByName lists items in a vault and finds one matching the given name.
// Returns the item ID or an error if no match, or ambiguous (multiple matches).
func (c *CLIVaultClient) ResolveItemByName(vaultID, name string) (string, error) {
	endpoint := fmt.Sprintf("/v1/vaults.items.list?vault_id=%s", vaultID)
	body, err := c.apiGet(endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to list vault items: %w", err)
	}

	var resp struct {
		OK    bool              `json:"ok"`
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}

	var matches []itemEntry
	for _, raw := range resp.Items {
		var item itemEntry
		if err := json.Unmarshal(raw, &item); err != nil {
			continue
		}
		if strings.EqualFold(item.Name, name) {
			matches = append(matches, item)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no item found with name %q in vault %s", name, vaultID)
	}
	if len(matches) > 1 {
		lines := []string{fmt.Sprintf("ambiguous item name %q matches %d items:", name, len(matches))}
		for _, m := range matches {
			lines = append(lines, fmt.Sprintf("  --item-id=%s  (%s)", m.ID, m.Name))
		}
		return "", fmt.Errorf("%s", strings.Join(lines, "\n"))
	}
	return matches[0].ID, nil
}

// apiGet makes an authenticated GET request to the Robomotion API.
func (c *CLIVaultClient) apiGet(endpoint string) ([]byte, error) {
	url := strings.TrimRight(c.apiBaseURL, "/") + endpoint

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("authentication expired; run 'robomotion login' again")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// decryptAESCBC decrypts data using AES-256-CBC with PKCS7 padding.
func decryptAESCBC(key, data []byte, ivHex string) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(data) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	iv, err := hex.DecodeString(ivHex)
	if err != nil {
		return nil, fmt.Errorf("invalid IV: %w", err)
	}

	cbc := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(data))
	cbc.CryptBlocks(plaintext, data)

	return pkcs7Unpad(plaintext), nil
}

// pkcs7Unpad removes PKCS7 padding from decrypted data.
func pkcs7Unpad(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	padding := int(data[len(data)-1])
	if padding > len(data) || padding == 0 {
		return data
	}
	return data[:len(data)-padding]
}

// SetMasterKeyCLI derives the master key from identity, password, and salt.
// This mirrors the deskbot SetMasterKey function.
func SetMasterKeyCLI(identity, password, salt string) []byte {
	saltBytes, _ := hex.DecodeString(salt)

	// HKDF with SHA256
	h := hkdf.New(sha256.New, []byte(identity), saltBytes, nil)
	derivedSalt := make([]byte, 32)
	io.ReadFull(h, derivedSalt)

	// Convert to hex string (matches deskbot behavior)
	derivedSaltHex := []byte(hex.EncodeToString(derivedSalt))

	// PBKDF2: 100,000 iterations, 32-byte key
	return pbkdf2.Key([]byte(password), derivedSaltHex, 100000, 32, sha256.New)
}

// defaultAuthConfigPath returns the path to the CLI auth config file.
func defaultAuthConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".robomotion", "auth.json")
}
