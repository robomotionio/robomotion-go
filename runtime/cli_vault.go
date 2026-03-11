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
	goruntime "runtime"
	"strings"

)

// CLIVaultClient provides direct vault access for CLI mode without gRPC/robot runtime.
// It authenticates with the Robomotion platform and fetches/decrypts vault items.
// Auth via env vars ROBOMOTION_API_TOKEN + ROBOMOTION_ROBOT_ID
// (token inherited from runner, private key from keys dir).
type CLIVaultClient struct {
	apiBaseURL      string
	accessToken     string
	robotID         string // used to request robot-specific enc_vault_key from API
	robotPrivateKey *rsa.PrivateKey
	vaultSecrets    map[string][]byte // vault ID → RSA-encrypted secret
}

type vaultItemResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Data  string `json:"data"` // hex-encoded encrypted data
	Item  struct {
		ID       string `json:"id"`
		Category int    `json:"category"`
		Name     string `json:"name"`
		IV       string `json:"iv"` // hex-encoded IV
	} `json:"item"`
}

// NewCLIVaultClient creates a vault client from env vars ROBOMOTION_API_TOKEN + ROBOMOTION_ROBOT_ID.
func NewCLIVaultClient() (*CLIVaultClient, error) {
	token := os.Getenv("ROBOMOTION_API_TOKEN")
	if token == "" {
		return nil, errors.New("ROBOMOTION_API_TOKEN env var is required")
	}
	return newCLIVaultClientFromEnv(token)
}

// newCLIVaultClientFromEnv creates a vault client from runner-inherited env vars.
// The runner has already authenticated; we reuse its session token and the robot's
// local keys for vault decryption.
func newCLIVaultClientFromEnv(token string) (*CLIVaultClient, error) {
	robotID := os.Getenv("ROBOMOTION_ROBOT_ID")
	if robotID == "" {
		return nil, errors.New("ROBOMOTION_API_TOKEN set but ROBOMOTION_ROBOT_ID is missing")
	}

	apiURL := os.Getenv("ROBOMOTION_API_URL")
	if apiURL == "" {
		apiURL = "https://api.robomotion.io"
	}

	c := &CLIVaultClient{
		apiBaseURL:  apiURL,
		accessToken: token,
		robotID:     robotID,
	}

	// Load robot's private key from ~/.config/robomotion/keys/<robotID>
	privKey, err := loadRobotPrivateKey(robotID)
	if err != nil {
		return nil, fmt.Errorf("robot private key: %w", err)
	}
	c.robotPrivateKey = privKey

	// Load vault secrets from ~/.config/robomotion/keys/<robotID>.vaults
	c.vaultSecrets = loadVaultSecrets(robotID)

	return c, nil
}

// loadRobotPrivateKey reads the robot's RSA private key from the keys directory.
// Path: ~/.config/robomotion/keys/<robotID>
func loadRobotPrivateKey(robotID string) (*rsa.PrivateKey, error) {
	keysDir := defaultKeysDir()
	privPath := filepath.Join(keysDir, robotID)

	pemData, err := os.ReadFile(privPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", privPath, err)
	}

	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in %s", privPath)
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// loadVaultSecrets reads .vaults files from the keys directory for a specific robot.
// Looks for <robotID>.vaults first, then falls back to all .vaults files.
// Each file is YAML: "vaults:\n  <vaultID>: <base64-encoded RSA-encrypted secret>\n"
// Returns a merged map of vault ID → RSA-encrypted secret bytes.
func loadVaultSecrets(robotID string) map[string][]byte {
	secrets := make(map[string][]byte)
	keysDir := defaultKeysDir()

	// Try robot-specific file first
	robotFile := filepath.Join(keysDir, robotID+".vaults")
	if data, err := os.ReadFile(robotFile); err == nil {
		parseVaultsFile(data, secrets)
		return secrets
	}

	// Fallback: read all .vaults files
	entries, err := os.ReadDir(keysDir)
	if err != nil {
		return secrets
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".vaults") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(keysDir, e.Name()))
		if err != nil {
			continue
		}
		parseVaultsFile(data, secrets)
	}

	return secrets
}

// parseVaultsFile parses a simple YAML vaults file into the secrets map.
// Format:
//
//	vaults:
//	  <uuid>: <base64>
func parseVaultsFile(data []byte, secrets map[string][]byte) {
	inVaults := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed == "---" {
			continue
		}
		if trimmed == "vaults:" {
			inVaults = true
			continue
		}
		if !inVaults {
			continue
		}
		// Entries are indented: "  <vaultID>: <base64>"
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			break // new top-level key, done with vaults section
		}
		idx := strings.Index(trimmed, ": ")
		if idx < 0 {
			continue
		}
		vaultID := trimmed[:idx]
		b64Value := trimmed[idx+2:]
		decoded, err := base64.StdEncoding.DecodeString(b64Value)
		if err != nil {
			continue
		}
		secrets[vaultID] = decoded
	}
}

// defaultKeysDir returns the path to the Robomotion keys directory.
func defaultKeysDir() string {
	return filepath.Join(configDir(), "keys")
}

// configDir returns the platform-specific Robomotion config directory.
func configDir() string {
	if goruntime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return filepath.Join(home, "AppData", "Local", "Robomotion")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "robomotion")
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
		if resp.Error != "" {
			return nil, fmt.Errorf("vault error: %s (vault=%s item=%s)", resp.Error, vaultID, itemID)
		}
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

	plaintext, err := decryptAESCBC(vaultKey, encData, resp.Item.IV)
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
// Two paths:
//   - Robot auth: private key loaded directly from keys dir
//   - User auth:  private key decrypted from keySet with master key
func (c *CLIVaultClient) getVaultKey(vaultID string) ([]byte, error) {
	// Get the RSA private key (robot or user path)
	privKey, err := c.getPrivateKey()
	if err != nil {
		return nil, err
	}

	// Fetch vault metadata for encrypted vault key
	vault, err := c.fetchVault(vaultID)
	if err != nil {
		return nil, err
	}

	// RSA-OAEP decrypt the vault key
	encVaultKey, err := base64.StdEncoding.DecodeString(vault.EncVaultKey)
	if err != nil {
		return nil, fmt.Errorf("invalid vault key encoding: %w", err)
	}

	hash := sha256.New()
	vaultKey, err := rsa.DecryptOAEP(hash, rand.Reader, privKey, encVaultKey, nil)
	if err != nil {
		return nil, fmt.Errorf("RSA decrypt vault key failed: %w", err)
	}

	// Get and decrypt secret key
	encSecretKey, err := c.getSecretKey(vaultID)
	if err != nil {
		return nil, err
	}

	hash.Reset()
	secretKey, err := rsa.DecryptOAEP(hash, rand.Reader, privKey, encSecretKey, nil)
	if err != nil {
		return nil, fmt.Errorf("RSA decrypt secret key failed: %w", err)
	}

	// XOR vault key with secret key
	if len(secretKey) != len(vaultKey) {
		return nil, errors.New("key length mismatch")
	}
	xored := make([]byte, len(secretKey))
	for i := range secretKey {
		xored[i] = secretKey[i] ^ vaultKey[i]
	}

	return xored, nil
}

// getPrivateKey returns the robot's RSA private key.
func (c *CLIVaultClient) getPrivateKey() (*rsa.PrivateKey, error) {
	if c.robotPrivateKey == nil {
		return nil, errors.New("no private key loaded")
	}
	return c.robotPrivateKey, nil
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
	endpoint := "/v1/vaults.list"
	if c.robotID != "" {
		endpoint += "?robot_id=" + c.robotID
	}
	body, err := c.apiGet(endpoint)
	if err != nil {
		return nil, err
	}

	// Find vault by ID in the response
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

// getSecretKey retrieves the encrypted secret key for a vault from .vaults files.
func (c *CLIVaultClient) getSecretKey(vaultID string) ([]byte, error) {
	if c.vaultSecrets != nil {
		if secret, ok := c.vaultSecrets[vaultID]; ok {
			return secret, nil
		}
	}
	return nil, fmt.Errorf("no vault secret for %s in .vaults files", vaultID)
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
	endpoint := "/v1/vaults.list"
	if c.robotID != "" {
		endpoint += "?robot_id=" + c.robotID
	}
	body, err := c.apiGet(endpoint)
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

