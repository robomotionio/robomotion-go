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
//
// Two auth paths are supported:
//  1. User auth: from ~/.robomotion/auth.json (via `robomotion login`)
//  2. Robot auth: from env vars ROBOMOTION_API_TOKEN + ROBOMOTION_ROBOT_ID
//     (token inherited from runner, private key from ~/.config/robomotion/keys/)
type CLIVaultClient struct {
	apiBaseURL  string
	accessToken string

	// User auth (from auth.json)
	masterKey []byte
	keySet    *cliKeySet

	// Robot auth (from env + keys dir)
	robotID         string // used to request robot-specific enc_vault_key from API
	robotPrivateKey *rsa.PrivateKey
	vaultSecrets    map[string][]byte // vault ID → RSA-encrypted secret
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

// NewCLIVaultClient creates a vault client using the best available auth source:
//  1. Environment (ROBOMOTION_API_TOKEN + ROBOMOTION_ROBOT_ID) — for processes spawned by the runner
//  2. Auth config (~/.robomotion/auth.json) — from `robomotion login`
func NewCLIVaultClient() (*CLIVaultClient, error) {
	// Try env-based robot auth first (runner → llm_agent → package flow)
	if token := os.Getenv("ROBOMOTION_API_TOKEN"); token != "" {
		return newCLIVaultClientFromEnv(token)
	}

	// Fall back to saved auth config
	return newCLIVaultClientFromConfig()
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

	fmt.Fprintf(os.Stderr, "[vault-debug] auth=env robotID=%s apiURL=%s token=%s...\n", robotID, apiURL, token[:8])

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
	fmt.Fprintf(os.Stderr, "[vault-debug] private key loaded: bits=%d\n", privKey.N.BitLen())

	// Load vault secrets from ~/.config/robomotion/keys/*.vaults
	c.vaultSecrets = loadVaultSecrets()
	fmt.Fprintf(os.Stderr, "[vault-debug] vault secrets loaded: %d vaults\n", len(c.vaultSecrets))

	return c, nil
}

// newCLIVaultClientFromConfig creates a vault client from ~/.robomotion/auth.json.
func newCLIVaultClientFromConfig() (*CLIVaultClient, error) {
	configPath := defaultAuthConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("no auth available: set ROBOMOTION_API_TOKEN env var, or run 'robomotion login': %w", err)
	}

	var config cliAuthConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid auth config at %s: %w", configPath, err)
	}

	if config.AccessToken == "" {
		return nil, fmt.Errorf("no access token in auth config; run 'robomotion login' again")
	}

	c := &CLIVaultClient{
		apiBaseURL:  config.APIEndpoint,
		accessToken: config.AccessToken,
		keySet:      config.KeySet,
	}

	if config.MasterKey != "" {
		c.masterKey, _ = hex.DecodeString(config.MasterKey)
	}

	if c.apiBaseURL == "" {
		c.apiBaseURL = "https://api.robomotion.io"
	}

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

// loadVaultSecrets reads all .vaults files from the keys directory.
// Each file is YAML: "vaults:\n  <vaultID>: <base64-encoded RSA-encrypted secret>\n"
// Returns a merged map of vault ID → RSA-encrypted secret bytes.
func loadVaultSecrets() map[string][]byte {
	secrets := make(map[string][]byte)
	keysDir := defaultKeysDir()

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
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "robomotion", "keys")
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
// Two paths:
//   - Robot auth: private key loaded directly from keys dir
//   - User auth:  private key decrypted from keySet with master key
func (c *CLIVaultClient) getVaultKey(vaultID string) ([]byte, error) {
	// Get the RSA private key (robot or user path)
	privKey, err := c.getPrivateKey()
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "[vault-debug] getVaultKey: have private key, bits=%d\n", privKey.N.BitLen())

	// Fetch vault metadata for encrypted vault key
	vault, err := c.fetchVault(vaultID)
	if err != nil {
		return nil, err
	}

	// RSA-OAEP decrypt the vault key
	fmt.Fprintf(os.Stderr, "[vault-debug] getVaultKey: enc_vault_key base64 len=%d\n", len(vault.EncVaultKey))
	encVaultKey, err := base64.StdEncoding.DecodeString(vault.EncVaultKey)
	if err != nil {
		return nil, fmt.Errorf("invalid vault key encoding: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[vault-debug] getVaultKey: enc_vault_key bytes len=%d\n", len(encVaultKey))

	hash := sha256.New()
	vaultKey, err := rsa.DecryptOAEP(hash, rand.Reader, privKey, encVaultKey, nil)
	if err != nil {
		return nil, fmt.Errorf("RSA decrypt vault key failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[vault-debug] getVaultKey: vaultKey decrypted, len=%d\n", len(vaultKey))

	// Get and decrypt secret key
	encSecretKey, err := c.getSecretKey(vaultID)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "[vault-debug] getVaultKey: encSecretKey len=%d\n", len(encSecretKey))

	hash.Reset()
	secretKey, err := rsa.DecryptOAEP(hash, rand.Reader, privKey, encSecretKey, nil)
	if err != nil {
		return nil, fmt.Errorf("RSA decrypt secret key failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[vault-debug] getVaultKey: secretKey decrypted, len=%d\n", len(secretKey))

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

// getPrivateKey returns the RSA private key from the appropriate source.
func (c *CLIVaultClient) getPrivateKey() (*rsa.PrivateKey, error) {
	// Robot auth: key already loaded from keys dir
	if c.robotPrivateKey != nil {
		return c.robotPrivateKey, nil
	}

	// User auth: decrypt from keySet with master key
	if c.masterKey == nil {
		return nil, errors.New("no private key available: set ROBOMOTION_API_TOKEN + ROBOMOTION_ROBOT_ID, or run 'robomotion login'")
	}
	if c.keySet == nil {
		return nil, errors.New("key set not available; run 'robomotion login' first")
	}

	privKeyPEM, err := c.decryptPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("private key decryption failed: %w", err)
	}

	block, _ := pem.Decode([]byte(privKeyPEM))
	if block == nil {
		return nil, errors.New("failed to parse private key PEM")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
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
	endpoint := "/v1/vaults.list"
	if c.robotID != "" {
		endpoint += "?robot_id=" + c.robotID
	}
	fmt.Fprintf(os.Stderr, "[vault-debug] fetchVault: endpoint=%s\n", endpoint)
	body, err := c.apiGet(endpoint)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "[vault-debug] fetchVault: response=%s\n", string(body))

	// Find vault by ID in the response
	var rawResp struct {
		OK     bool              `json:"ok"`
		Vaults []json.RawMessage `json:"vaults"`
	}
	if err := json.Unmarshal(body, &rawResp); err != nil {
		return nil, err
	}

	fmt.Fprintf(os.Stderr, "[vault-debug] fetchVault: found %d vaults, looking for %s\n", len(rawResp.Vaults), vaultID)

	for _, raw := range rawResp.Vaults {
		var v struct {
			ID          string `json:"id"`
			EncVaultKey string `json:"enc_vault_key"`
			IV          string `json:"iv"`
		}
		if err := json.Unmarshal(raw, &v); err != nil {
			continue
		}
		fmt.Fprintf(os.Stderr, "[vault-debug] fetchVault: vault id=%s enc_vault_key_len=%d iv=%s\n", v.ID, len(v.EncVaultKey), v.IV)
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
// Tries robot auth (.vaults files) first, then user auth (auth.json vault_secrets).
func (c *CLIVaultClient) getSecretKey(vaultID string) ([]byte, error) {
	// Robot auth: vault secrets loaded from .vaults files in keys dir
	if c.vaultSecrets != nil {
		if secret, ok := c.vaultSecrets[vaultID]; ok {
			return secret, nil
		}
	}

	// User auth: vault secrets saved in auth.json
	configPath := defaultAuthConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("no vault secret for %s: not in .vaults files, cannot read auth config: %w", vaultID, err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	secrets, ok := config["vault_secrets"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no vault secret for %s: not in .vaults files or auth config", vaultID)
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
