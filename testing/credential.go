package testing

import (
	"bufio"
	"os"
	"strings"
	"sync"

	"github.com/robomotionio/robomotion-go/runtime"
)

// CredentialStore holds mock credentials for testing.
// Credentials are stored by a key and can be retrieved during tests.
type CredentialStore struct {
	mu          sync.RWMutex
	credentials map[string]map[string]interface{}
}

// NewCredentialStore creates a new credential store.
func NewCredentialStore() *CredentialStore {
	return &CredentialStore{
		credentials: make(map[string]map[string]interface{}),
	}
}

// SetAPIKey stores an API key credential (category 4).
// The returned map will have the structure: {"value": apiKey}
func (s *CredentialStore) SetAPIKey(name, apiKey string) *CredentialStore {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials[name] = map[string]interface{}{
		"value": apiKey,
	}
	return s
}

// SetLogin stores a login credential (category 1).
// The returned map will have the structure: {"username": username, "password": password}
func (s *CredentialStore) SetLogin(name, username, password string) *CredentialStore {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials[name] = map[string]interface{}{
		"username": username,
		"password": password,
	}
	return s
}

// SetDatabase stores a database credential (category 5).
func (s *CredentialStore) SetDatabase(name string, config DatabaseCredential) *CredentialStore {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials[name] = map[string]interface{}{
		"server":   config.Server,
		"port":     config.Port,
		"database": config.Database,
		"username": config.Username,
		"password": config.Password,
	}
	return s
}

// SetDocument stores a document credential (category 6).
// The returned map will have the structure: {"content": content}
func (s *CredentialStore) SetDocument(name, content string) *CredentialStore {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials[name] = map[string]interface{}{
		"content": content,
	}
	return s
}

// SetCustom stores a custom credential with arbitrary fields.
func (s *CredentialStore) SetCustom(name string, data map[string]interface{}) *CredentialStore {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials[name] = data
	return s
}

// Get retrieves a credential by name.
func (s *CredentialStore) Get(name string) (map[string]interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.credentials[name]
	return data, ok
}

// DatabaseCredential holds database connection details.
type DatabaseCredential struct {
	Server   string
	Port     string
	Database string
	Username string
	Password string
}

// LoadFromEnv loads credentials from environment variables.
// It looks for variables with the given prefix and maps them to credential fields.
//
// Example:
//
//	// .env file:
//	// GEMINI_API_KEY=AIza...
//	// MYSQL_SERVER=localhost
//	// MYSQL_PORT=3306
//
//	store.LoadFromEnv("GEMINI", "gemini_cred")  // Creates {"value": "AIza..."}
func (s *CredentialStore) LoadFromEnv(prefix, credName string) *CredentialStore {
	s.mu.Lock()
	defer s.mu.Unlock()

	data := make(map[string]interface{})
	prefix = strings.ToUpper(prefix) + "_"

	// Common field mappings
	fieldMappings := map[string]string{
		"API_KEY":  "value",
		"KEY":      "value",
		"TOKEN":    "value",
		"VALUE":    "value",
		"USERNAME": "username",
		"USER":     "username",
		"PASSWORD": "password",
		"PASS":     "password",
		"SERVER":   "server",
		"HOST":     "server",
		"PORT":     "port",
		"DATABASE": "database",
		"DB":       "database",
		"CONTENT":  "content",
	}

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]

		if !strings.HasPrefix(key, prefix) {
			continue
		}

		// Extract field name from env var
		fieldName := strings.TrimPrefix(key, prefix)

		// Map to standard field name if possible
		if mapped, ok := fieldMappings[fieldName]; ok {
			data[mapped] = value
		} else {
			// Use lowercase field name
			data[strings.ToLower(fieldName)] = value
		}
	}

	if len(data) > 0 {
		s.credentials[credName] = data
	}

	return s
}

// LoadDotEnv loads environment variables from a .env file.
func LoadDotEnv(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		os.Setenv(key, value)
	}

	return scanner.Err()
}

// mockHelper implements runtime.TestRuntimeHelper for credential testing.
type mockHelper struct {
	store *CredentialStore
}

// GetVaultItem retrieves a credential from the store.
// It tries to match by vaultID first, then by itemID.
func (m *mockHelper) GetVaultItem(vaultID, itemID string) (map[string]interface{}, error) {
	// Try vaultID as credential name
	if data, ok := m.store.Get(vaultID); ok {
		return data, nil
	}
	// Try itemID as credential name
	if data, ok := m.store.Get(itemID); ok {
		return data, nil
	}
	// Try combined key
	key := vaultID + ":" + itemID
	if data, ok := m.store.Get(key); ok {
		return data, nil
	}
	return nil, nil // Return nil without error to allow optional credentials
}

// SetVaultItem is a no-op for testing.
func (m *mockHelper) SetVaultItem(vaultID, itemID string, data []byte) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

// InitCredentials initializes the runtime with mock credentials.
// This must be called before running tests that use credentials.
//
// Example:
//
//	func TestMain(m *testing.M) {
//	    store := rtesting.NewCredentialStore()
//	    rtesting.LoadDotEnv(".env")
//	    store.LoadFromEnv("GEMINI", "api_key")
//	    rtesting.InitCredentials(store)
//
//	    os.Exit(m.Run())
//	}
func InitCredentials(store *CredentialStore) {
	runtime.SetTestClient(&mockHelper{store: store})
}

// ClearCredentials clears the mock credentials.
// Call this in test cleanup if needed.
func ClearCredentials() {
	runtime.ClearTestClient()
}
