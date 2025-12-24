# Robomotion Go Package Testing Guide

*Last updated: December 24, 2025*

This guide covers writing **integration tests** for Robomotion packages using the `rtesting` framework. These tests execute real `OnMessage()` methods with real API calls.

---

## 1. Overview

The `rtesting` package provides tools to test Robomotion nodes without needing the full runtime environment:

| Component | Purpose |
|-----------|---------|
| `Quick` | Harness that configures nodes and runs `OnMessage()` |
| `CredentialStore` | Mock credential vault that loads from `.env` files |
| `MockContext` | Lightweight message context for helper function tests |
| `LoadDotEnv()` | Loads environment variables from `.env` files |
| `InitCredentials()` | Initializes the runtime with mock credentials |

**These are integration tests** — they call real APIs with real credentials. The mock is only for the runtime plumbing, not the business logic.

---

## 2. Test File Structure

Follow Go convention: one test file per source file, plus a shared setup file.

```
v1/
├── common.go              # Shared utilities
├── connect.go             # Connect node
├── GenerateText.go        # GenerateText node
├── Embeddings.go          # Embeddings node
│
├── common_test.go         # TestMain, shared setup ←
├── connect_test.go        # Tests for Connect
├── GenerateText_test.go   # Tests for GenerateText
├── Embeddings_test.go     # Tests for Embeddings
├── helpers_test.go        # Tests for utility functions
│
├── .env                   # Credentials (git-ignored!)
└── TESTING.md             # Requirements to run tests
```

---

## 3. Basic Setup (common_test.go)

Every package needs a `common_test.go` with `TestMain` for credential setup:

```go
package v1

import (
    "os"
    "testing"

    rtesting "github.com/robomotionio/robomotion-go/testing"
)

var credStore *rtesting.CredentialStore

func TestMain(m *testing.M) {
    // Initialize credential store
    credStore = rtesting.NewCredentialStore()

    // Load .env file (won't fail if missing)
    rtesting.LoadDotEnv(".env")

    // Load credentials from environment
    // Pattern: PREFIX_API_KEY, PREFIX_KEY, or PREFIX_VALUE → "credential_id"
    credStore.LoadFromEnv("GEMINI", "api_key")
    credStore.LoadFromEnv("OPENAI", "openai_key")

    // Initialize runtime with mock credentials
    rtesting.InitCredentials(credStore)

    code := m.Run()

    // Cleanup
    rtesting.ClearCredentials()
    os.Exit(code)
}

// Helper to check if credentials are available
func hasCredentials() bool {
    _, ok := credStore.Get("api_key")
    return ok
}
```

---

## 4. Writing Node Tests

### 4.1 Basic Test Structure

```go
package v1

import (
    "testing"

    rtesting "github.com/robomotionio/robomotion-go/testing"
)

func TestGenerateText(t *testing.T) {
    t.Run("basic text generation", func(t *testing.T) {
        // Skip if no credentials
        if !hasCredentials() {
            t.Skip("No API key in environment, skipping")
        }

        // Create node with options set directly
        node := &GenerateText{
            OptModel: "gemini-2.0-flash-lite",
        }

        // Create Quick harness
        q := rtesting.NewQuick(node)

        // Set credential (field name, vault ID, item ID)
        q.SetCredential("OptApiKey", "api_key", "api_key")

        // Set input variables
        q.SetCustom("InText", "Say hello in 3 words")

        // Run OnMessage()
        err := q.Run()
        if err != nil {
            t.Fatalf("GenerateText failed: %v", err)
        }

        // Check outputs
        text := q.GetOutput("text")
        if text == nil || text == "" {
            t.Error("Expected text output to be set")
        }

        t.Logf("Generated: %v", text)
    })
}
```

### 4.2 Quick Harness Methods

| Method | Purpose | Example |
|--------|---------|---------|
| `NewQuick(node)` | Create harness for node | `q := rtesting.NewQuick(&MyNode{})` |
| `SetInput(name, value)` | Set message-scope input | `q.SetInput("fileUri", "files/123")` |
| `SetCustom(field, value)` | Set custom-scope input | `q.SetCustom("InText", "Hello")` |
| `SetCredential(field, vaultID, itemID)` | Set credential reference | `q.SetCredential("OptApiKey", "api_key", "api_key")` |
| `Run()` | Execute `OnCreate()` + `OnMessage()` | `err := q.Run()` |
| `GetOutput(name)` | Get output value | `result := q.GetOutput("text")` |

### 4.3 Setting Node Options

Options with `option` tag are set directly on the node struct:

```go
node := &GenerateImage{
    OptModel:          "gemini-2.5-flash-image",  // enum option
    OptNumberOfImages: "1",                        // string option
    OptAspectRatio:    "16:9",                     // string option
}
q := rtesting.NewQuick(node)
```

### 4.4 Testing Error Cases

```go
t.Run("empty prompt error", func(t *testing.T) {
    if !hasCredentials() {
        t.Skip("No API key in environment, skipping")
    }

    node := &GenerateText{
        OptModel: "gemini-2.0-flash-lite",
    }
    q := rtesting.NewQuick(node)
    q.SetCredential("OptApiKey", "api_key", "api_key")
    q.SetCustom("InText", "")  // Empty prompt

    err := q.Run()
    if err == nil {
        t.Error("Expected error for empty prompt")
    }
})

t.Run("invalid parameter", func(t *testing.T) {
    if !hasCredentials() {
        t.Skip("No API key in environment, skipping")
    }

    node := &SendChatMessage{
        OptModel: "gemini-2.0-flash-lite",
    }
    q := rtesting.NewQuick(node)
    q.SetCredential("OptApiKey", "api_key", "api_key")
    q.SetCustom("InText", "Test")
    q.SetCustom("OptTemperature", "3.0")  // Invalid: > 2.0

    err := q.Run()
    if err == nil {
        t.Error("Expected error for temperature > 2.0")
    }
})
```

---

## 5. Credential Management

### 5.1 The .env File

Create a `.env` file in the `v1/` directory with your test credentials:

```bash
# .env - Add to .gitignore!
GEMINI_API_KEY=your_api_key_here
```

### 5.2 Loading Credentials

The `LoadFromEnv()` function searches for environment variables with these patterns:

```go
credStore.LoadFromEnv("GEMINI", "api_key")
// Searches for: GEMINI_API_KEY, GEMINI_KEY, GEMINI_VALUE
// Stores result under ID: "api_key"
```

### 5.3 Credential Categories & .env Examples

Each credential category has different fields. Here are `.env` examples for each:

---

#### Category 1: Login (Basic Auth)

Used for: HTTP authentication, FTP, browser automation

**Vault Item Structure:**
```json
{
  "username": "user@example.com",
  "password": "secretpassword",
  "server": "https://api.example.com"
}
```

**.env Format:**
```bash
# Category 1 - Login
SERVICE_USERNAME=user@example.com
SERVICE_PASSWORD=secretpassword
SERVICE_SERVER=https://api.example.com
```

**Test Setup:**
```go
// In common_test.go
credStore.AddCredential("login_cred", map[string]interface{}{
    "username": os.Getenv("SERVICE_USERNAME"),
    "password": os.Getenv("SERVICE_PASSWORD"),
    "server":   os.Getenv("SERVICE_SERVER"),
})

// In test
q.SetCredential("OptLogin", "login_cred", "login_cred")
```

---

#### Category 2: Email

Used for: SMTP/IMAP email operations

**Vault Item Structure:**
```json
{
  "inbox": {
    "username": "user@gmail.com",
    "password": "app_password",
    "server": "imap.gmail.com",
    "port": 993,
    "security": "SSL/TLS"
  },
  "smtp": {
    "username": "user@gmail.com",
    "password": "app_password",
    "server": "smtp.gmail.com",
    "port": 587,
    "security": "STARTTLS"
  }
}
```

**.env Format:**
```bash
# Category 2 - Email
EMAIL_INBOX_USERNAME=user@gmail.com
EMAIL_INBOX_PASSWORD=app_password_here
EMAIL_INBOX_SERVER=imap.gmail.com
EMAIL_INBOX_PORT=993

EMAIL_SMTP_USERNAME=user@gmail.com
EMAIL_SMTP_PASSWORD=app_password_here
EMAIL_SMTP_SERVER=smtp.gmail.com
EMAIL_SMTP_PORT=587
```

**Test Setup:**
```go
credStore.AddCredential("email_cred", map[string]interface{}{
    "inbox": map[string]interface{}{
        "username": os.Getenv("EMAIL_INBOX_USERNAME"),
        "password": os.Getenv("EMAIL_INBOX_PASSWORD"),
        "server":   os.Getenv("EMAIL_INBOX_SERVER"),
        "port":     993,
        "security": "SSL/TLS",
    },
    "smtp": map[string]interface{}{
        "username": os.Getenv("EMAIL_SMTP_USERNAME"),
        "password": os.Getenv("EMAIL_SMTP_PASSWORD"),
        "server":   os.Getenv("EMAIL_SMTP_SERVER"),
        "port":     587,
        "security": "STARTTLS",
    },
})
```

---

#### Category 3: Credit Card

Used for: Payment processing (rarely used in tests)

**Vault Item Structure:**
```json
{
  "cardholder": "John Doe",
  "cardnumber": "4111111111111111",
  "cvv": "123",
  "expDate": "12/25"
}
```

**.env Format:**
```bash
# Category 3 - Credit Card (use test card numbers only!)
CARD_HOLDER=Test User
CARD_NUMBER=4111111111111111
CARD_CVV=123
CARD_EXP=12/25
```

---

#### Category 4: API Key/Token

Used for: Most API integrations (Gemini, OpenAI, Slack, etc.)

**Vault Item Structure:**
```json
{
  "value": "sk-abc123..."
}
```

**.env Format:**
```bash
# Category 4 - API Key
GEMINI_API_KEY=AIza...your_key_here
OPENAI_API_KEY=sk-...your_key_here
SLACK_API_KEY=xoxb-...your_token_here
GITHUB_TOKEN=ghp_...your_token_here
```

**Test Setup (Simplest):**
```go
// Automatic loading for API keys
credStore.LoadFromEnv("GEMINI", "api_key")
// This searches for GEMINI_API_KEY, GEMINI_KEY, or GEMINI_VALUE
// and creates {"value": "..."} structure automatically

// In test
q.SetCredential("OptApiKey", "api_key", "api_key")
```

---

#### Category 5: Database

Used for: PostgreSQL, MySQL, MongoDB, SQL Server connections

**Vault Item Structure:**
```json
{
  "server": "localhost",
  "port": 5432,
  "database": "mydb",
  "username": "postgres",
  "password": "secret"
}
```

**.env Format:**
```bash
# Category 5 - Database
DB_SERVER=localhost
DB_PORT=5432
DB_DATABASE=testdb
DB_USERNAME=postgres
DB_PASSWORD=secretpassword

# Or for connection string style
MONGODB_URI=mongodb://user:pass@localhost:27017/mydb
```

**Test Setup:**
```go
credStore.AddCredential("db_cred", map[string]interface{}{
    "server":   os.Getenv("DB_SERVER"),
    "port":     5432,
    "database": os.Getenv("DB_DATABASE"),
    "username": os.Getenv("DB_USERNAME"),
    "password": os.Getenv("DB_PASSWORD"),
})
```

---

#### Category 6: Document (OAuth2, Service Accounts, Certificates)

Used for: Google Service Accounts, OAuth tokens, certificates, complex JSON credentials

**Vault Item Structure:**
```json
{
  "filename": "service-account.json",
  "content": "{\"type\":\"service_account\",\"project_id\":\"...\",\"private_key\":\"...\"}"
}
```

**.env Format:**
```bash
# Category 6 - Document/JSON

# Option A: Path to JSON file
GOOGLE_SERVICE_ACCOUNT_FILE=/path/to/service-account.json

# Option B: Inline JSON (escaped)
GOOGLE_SERVICE_ACCOUNT_JSON={"type":"service_account","project_id":"my-project",...}

# For OAuth2 tokens
OAUTH_CLIENT_ID=123456789.apps.googleusercontent.com
OAUTH_CLIENT_SECRET=GOCSPX-...
OAUTH_REFRESH_TOKEN=1//0g...

# For certificates
CERT_FILE=/path/to/certificate.pem
CERT_KEY_FILE=/path/to/private-key.pem
```

**Test Setup (Service Account):**
```go
import "os"

func loadServiceAccount() map[string]interface{} {
    // Option A: From file
    if path := os.Getenv("GOOGLE_SERVICE_ACCOUNT_FILE"); path != "" {
        content, err := os.ReadFile(path)
        if err == nil {
            return map[string]interface{}{
                "filename": "service-account.json",
                "content":  string(content),
            }
        }
    }

    // Option B: From inline JSON
    if json := os.Getenv("GOOGLE_SERVICE_ACCOUNT_JSON"); json != "" {
        return map[string]interface{}{
            "filename": "service-account.json",
            "content":  json,
        }
    }

    return nil
}

// In TestMain
if sa := loadServiceAccount(); sa != nil {
    credStore.AddCredential("service_account", sa)
}
```

**Test Setup (OAuth2):**
```go
credStore.AddCredential("oauth_cred", map[string]interface{}{
    "filename": "oauth-token.json",
    "content": fmt.Sprintf(`{
        "client_id": "%s",
        "client_secret": "%s",
        "refresh_token": "%s",
        "token_type": "Bearer"
    }`,
        os.Getenv("OAUTH_CLIENT_ID"),
        os.Getenv("OAUTH_CLIENT_SECRET"),
        os.Getenv("OAUTH_REFRESH_TOKEN"),
    ),
})
```

---

#### Category 7: AES Key

Used for: Symmetric encryption/decryption

**Vault Item Structure:**
```json
{
  "value": "base64_encoded_aes_key"
}
```

**.env Format:**
```bash
# Category 7 - AES Key (32 bytes for AES-256, base64 encoded)
AES_KEY=K7gNU3sdo+OL0wNhqoVWhr3g6s1xYv72ol/pe/Unols=
```

**Test Setup:**
```go
credStore.AddCredential("aes_key", map[string]interface{}{
    "value": os.Getenv("AES_KEY"),
})
```

---

#### Category 8: RSA Key Pair

Used for: Asymmetric encryption, JWT signing

**Vault Item Structure:**
```json
{
  "publicKey": "-----BEGIN PUBLIC KEY-----\n...",
  "privateKey": "-----BEGIN PRIVATE KEY-----\n..."
}
```

**.env Format:**
```bash
# Category 8 - RSA Keys (paths to PEM files)
RSA_PUBLIC_KEY_FILE=/path/to/public.pem
RSA_PRIVATE_KEY_FILE=/path/to/private.pem

# Or inline (with \n for newlines)
RSA_PUBLIC_KEY="-----BEGIN PUBLIC KEY-----\nMIIB..."
RSA_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\nMIIE..."
```

**Test Setup:**
```go
func loadRSAKeys() map[string]interface{} {
    result := make(map[string]interface{})

    if pubPath := os.Getenv("RSA_PUBLIC_KEY_FILE"); pubPath != "" {
        if content, err := os.ReadFile(pubPath); err == nil {
            result["publicKey"] = string(content)
        }
    }

    if privPath := os.Getenv("RSA_PRIVATE_KEY_FILE"); privPath != "" {
        if content, err := os.ReadFile(privPath); err == nil {
            result["privateKey"] = string(content)
        }
    }

    return result
}

// In TestMain
if keys := loadRSAKeys(); len(keys) > 0 {
    credStore.AddCredential("rsa_keys", keys)
}
```

---

### 5.4 Multiple Credentials

For packages using multiple credential types:

```go
func TestMain(m *testing.M) {
    credStore = rtesting.NewCredentialStore()
    rtesting.LoadDotEnv(".env")

    // API Key credentials
    credStore.LoadFromEnv("GEMINI", "gemini_key")
    credStore.LoadFromEnv("OPENAI", "openai_key")

    // Database credential
    credStore.AddCredential("postgres", map[string]interface{}{
        "server":   os.Getenv("PG_HOST"),
        "port":     5432,
        "database": os.Getenv("PG_DATABASE"),
        "username": os.Getenv("PG_USER"),
        "password": os.Getenv("PG_PASSWORD"),
    })

    // Service account credential
    if content, err := os.ReadFile(os.Getenv("GOOGLE_SA_FILE")); err == nil {
        credStore.AddCredential("google_sa", map[string]interface{}{
            "filename": "service-account.json",
            "content":  string(content),
        })
    }

    rtesting.InitCredentials(credStore)
    code := m.Run()
    rtesting.ClearCredentials()
    os.Exit(code)
}
```

---

## 6. Testing Helper Functions

Use `MockContext` for testing non-node functions:

```go
func TestGetModuleOrCreate(t *testing.T) {
    t.Run("with connection ID", func(t *testing.T) {
        // Setup: create a client
        connID := addClient("test-key", false)
        defer delClient(connID)

        // Create mock context
        ctx := rtesting.NewMockContext()

        // Test the helper function
        module, err := getModuleOrCreate(ctx, connID, runtime.Credential{}, false)
        if err != nil {
            t.Fatalf("getModuleOrCreate failed: %v", err)
        }

        if module == nil {
            t.Error("Expected module to be returned")
        }

        if module.ApiKey != "test-key" {
            t.Errorf("Expected api key 'test-key', got '%s'", module.ApiKey)
        }
    })
}

func TestCalculateSimilarity(t *testing.T) {
    t.Run("cosine similarity identical vectors", func(t *testing.T) {
        a := []float32{1.0, 0.0, 0.0}
        b := []float32{1.0, 0.0, 0.0}

        sim, _ := calculateSimilarity(a, b, "cosine")

        if sim < 0.99 {
            t.Errorf("Expected ~1.0, got %f", sim)
        }
    })
}
```

---

## 7. Test Categories

### 7.1 What to Test

| Category | Description | Example |
|----------|-------------|---------|
| **Happy Path** | Normal successful operation | Generate text with valid prompt |
| **Error Cases** | Expected failures | Empty input, invalid parameters |
| **Edge Cases** | Boundary conditions | Max length input, special characters |
| **Validation** | Parameter validation | Temperature > 2.0, invalid enum |
| **Connection** | Connect/Disconnect lifecycle | Create and cleanup connections |

### 7.2 Skipping Slow Tests

Mark expensive tests (image generation, video, large files):

```go
t.Run("basic image generation", func(t *testing.T) {
    if !hasCredentials() {
        t.Skip("No API key in environment, skipping")
    }
    if os.Getenv("SKIP_SLOW_TESTS") != "" {
        t.Skip("Skipping slow test")
    }

    // ... expensive test
})
```

Run fast tests only:
```bash
SKIP_SLOW_TESTS=1 go test ./v1/...
```

---

## 8. Running Tests

```bash
# Run all tests
go test ./v1/...

# Run with verbose output
go test -v ./v1/...

# Run specific test
go test -v ./v1/... -run TestGenerateText

# Run specific subtest
go test -v ./v1/... -run TestGenerateText/basic

# Run with coverage
go test -cover ./v1/...

# Skip slow tests
SKIP_SLOW_TESTS=1 go test ./v1/...

# Run with timeout
go test -timeout 5m ./v1/...
```

---

## 9. Best Practices

### 9.1 Always Skip Without Credentials

```go
if !hasCredentials() {
    t.Skip("No API credentials available, skipping")
}
```

### 9.2 Use Subtests for Organization

```go
func TestEmbeddings(t *testing.T) {
    t.Run("basic embedding", func(t *testing.T) { ... })
    t.Run("empty content error", func(t *testing.T) { ... })
    t.Run("with comparison text", func(t *testing.T) { ... })
}
```

### 9.3 Clean Up Resources

```go
t.Run("with connection", func(t *testing.T) {
    connID := addClient("test-key", false)
    defer delClient(connID)  // Always cleanup

    // ... test code
})
```

### 9.4 Log Useful Information

```go
text := q.GetOutput("text")
t.Logf("Generated text: %v", text)

// Log response types for debugging
t.Logf("Response type: %T", response)
```

### 9.5 Test File Naming

| Source File | Test File |
|-------------|-----------|
| `connect.go` | `connect_test.go` |
| `GenerateText.go` | `GenerateText_test.go` |
| `common.go` | `helpers_test.go` (for utility functions) |
| (shared setup) | `common_test.go` |

---

## 10. TESTING.md Template

Every package should include a `TESTING.md` file describing requirements:

```markdown
# Testing Requirements

## Prerequisites

1. Go 1.20+
2. API credentials (see below)

## Credentials Setup

1. Create `.env` file in `v1/` directory
2. Add required credentials:

```bash
# Required
SERVICE_API_KEY=your_api_key_here

# Optional (for specific tests)
SERVICE_SECRET=optional_secret
```

## Getting Credentials

1. Go to [Service Console](https://console.example.com)
2. Create a new project or select existing
3. Navigate to API Keys section
4. Create a new API key
5. Copy the key to `.env`

## Running Tests

```bash
# All tests
go test ./v1/...

# Fast tests only
SKIP_SLOW_TESTS=1 go test ./v1/...

# Specific test
go test -v ./v1/... -run TestGenerateTex
```

## Test Files Required

- None (all tests use API calls)

## Notes

- Tests make real API calls and may incur costs
- Some tests are slow (image/video generation)
- Rate limits may cause occasional failures
```

---

## 11. Complete Example

Here's a complete test file for reference:

```go
// v1/GenerateText_test.go
package v1

import (
    "testing"

    rtesting "github.com/robomotionio/robomotion-go/testing"
)

func TestGenerateText(t *testing.T) {
    t.Run("basic text generation", func(t *testing.T) {
        if !hasCredentials() {
            t.Skip("No GEMINI_API_KEY in environment, skipping")
        }

        node := &GenerateText{
            OptModel: "gemini-2.0-flash-lite",
        }
        q := rtesting.NewQuick(node)
        q.SetCredential("OptApiKey", "api_key", "api_key")
        q.SetCustom("InText", "Say hello in exactly 3 words")

        err := q.Run()
        if err != nil {
            t.Fatalf("GenerateText failed: %v", err)
        }

        text := q.GetOutput("text")
        if text == nil || text == "" {
            t.Error("Expected text output to be set")
        }
        t.Logf("Generated text: %v", text)
    })

    t.Run("with system prompt", func(t *testing.T) {
        if !hasCredentials() {
            t.Skip("No GEMINI_API_KEY in environment, skipping")
        }

        node := &GenerateText{
            OptModel: "gemini-2.0-flash-lite",
        }
        q := rtesting.NewQuick(node)
        q.SetCredential("OptApiKey", "api_key", "api_key")
        q.SetCustom("InSystemPrompt", "You are a pirate. Always respond in pirate speak.")
        q.SetCustom("InText", "Say hello")

        err := q.Run()
        if err != nil {
            t.Fatalf("GenerateText with system prompt failed: %v", err)
        }

        text := q.GetOutput("text")
        if text == nil || text == "" {
            t.Error("Expected text output to be set")
        }
        t.Logf("Generated text (pirate): %v", text)
    })

    t.Run("empty prompt error", func(t *testing.T) {
        if !hasCredentials() {
            t.Skip("No GEMINI_API_KEY in environment, skipping")
        }

        node := &GenerateText{
            OptModel: "gemini-2.0-flash-lite",
        }
        q := rtesting.NewQuick(node)
        q.SetCredential("OptApiKey", "api_key", "api_key")
        q.SetCustom("InText", "")

        err := q.Run()
        if err == nil {
            t.Error("Expected error for empty prompt")
        }
    })

    t.Run("JSON mode", func(t *testing.T) {
        if !hasCredentials() {
            t.Skip("No GEMINI_API_KEY in environment, skipping")
        }

        node := &GenerateText{
            OptModel:    "gemini-2.0-flash-lite",
            OptJSONMode: true,
        }
        q := rtesting.NewQuick(node)
        q.SetCredential("OptApiKey", "api_key", "api_key")
        q.SetCustom("InText", "Return JSON with name=John and age=30")

        err := q.Run()
        if err != nil {
            t.Fatalf("GenerateText JSON mode failed: %v", err)
        }

        text := q.GetOutput("text")
        if text == nil || text == "" {
            t.Error("Expected text output to be set")
        }
        t.Logf("Generated JSON: %v", text)
    })
}
```

---

## 12. Writing Real Integration Tests

**Important**: Write tests that actually test real functionality, not just input validation. Use the scratchpad approach to discover actual API response structures before writing assertions.

### 12.1 The Scratchpad Approach

Before writing tests for a node, create a temporary scratchpad to see the actual output structure:

```go
// v1/scratch_test.go - TEMPORARY FILE, DELETE AFTER USE
package v1

import (
    "encoding/json"
    "fmt"
    "testing"

    rtesting "github.com/robomotionio/robomotion-go/testing"
)

// prettyPrint shows actual output structure
func prettyPrint(name string, v interface{}) {
    fmt.Printf("\n=== %s ===\n", name)
    fmt.Printf("Type: %T\n", v)
    if b, err := json.MarshalIndent(v, "", "  "); err == nil {
        fmt.Printf("Value:\n%s\n", string(b))
    }
}

func TestScratch(t *testing.T) {
    if !hasCredentials() {
        t.Skip("No credentials")
    }

    node := &ListModels{}
    q := rtesting.NewQuick(node)
    q.SetCredential("OptApiKey", "api_key", "api_key")

    err := q.Run()
    if err != nil {
        t.Fatalf("Failed: %v", err)
    }

    // See actual output structure
    prettyPrint("models output", q.GetOutput("models"))
}
```

Run it:
```bash
go test -v ./v1/... -run TestScratch
```

Example output:
```
=== models output ===
Type: map[string]interface {}
Value:
{
  "count": 5,
  "models": [
    {
      "name": "gemini-1.5-pro",
      "displayName": "Gemini 1.5 Pro",
      "supportedGenerationMethods": ["generateContent"]
    }
  ],
  "statistics": {
    "totalModels": 5
  }
}
```

Now write tests based on **actual structure**, not assumptions:

```go
// Real test based on discovered structure
func TestListModels(t *testing.T) {
    t.Run("list available models", func(t *testing.T) {
        // ... setup ...

        err := q.Run()
        if err != nil {
            t.Fatalf("ListModels failed: %v", err)
        }

        models := q.GetOutput("models")
        modelsMap := models.(map[string]interface{})

        // Check actual fields we discovered
        if count, ok := modelsMap["count"].(float64); !ok {
            t.Errorf("Expected count to be a number, got %T", modelsMap["count"])
        } else {
            t.Logf("Total models: %.0f", count)
        }

        if modelList, ok := modelsMap["models"].([]interface{}); ok && len(modelList) > 0 {
            firstModel := modelList[0].(map[string]interface{})
            if _, hasName := firstModel["name"]; !hasName {
                t.Error("Expected model to have name field")
            }
            if _, hasDisplayName := firstModel["displayName"]; !hasDisplayName {
                t.Error("Expected model to have displayName field")
            }
        }
    })
}
```

**Note**: You can either delete `scratch_test.go` after exploring, or keep it for future reference. If keeping it, rename to something like `scratch_explore_test.go` and add a build tag to exclude from normal test runs:

```go
//go:build scratch
// +build scratch

package v1

// Run with: go test -v -tags=scratch ./v1/... -run TestScratch
```

This way the scratchpad won't run during normal `go test` but remains available when you need to explore new node outputs.

### 12.2 Test Files and Testdata

For tests that need files (images, documents, etc.), you have several options:

#### Option A: Create Files Programmatically (Recommended)

Add helper functions in `common_test.go` to generate test files:

```go
import (
    "image"
    "image/color"
    "image/png"
    "os"
)

// createTestImage creates a simple colored PNG image for testing
func createTestImage(width, height int, col color.Color) (*os.File, error) {
    img := image.NewRGBA(image.Rect(0, 0, width, height))
    for y := 0; y < height; y++ {
        for x := 0; x < width; x++ {
            img.Set(x, y, col)
        }
    }

    tmpFile, err := os.CreateTemp("", "test-image-*.png")
    if err != nil {
        return nil, err
    }

    if err := png.Encode(tmpFile, img); err != nil {
        tmpFile.Close()
        os.Remove(tmpFile.Name())
        return nil, err
    }

    tmpFile.Close()
    return tmpFile, nil
}

// createTestMaskImage creates a mask image (white center on black background)
func createTestMaskImage(width, height int) (*os.File, error) {
    img := image.NewRGBA(image.Rect(0, 0, width, height))

    // Fill with black, white center
    for y := 0; y < height; y++ {
        for x := 0; x < width; x++ {
            img.Set(x, y, color.Black)
        }
    }
    centerX, centerY := width/2, height/2
    radius := width / 4
    for y := centerY - radius; y < centerY+radius; y++ {
        for x := centerX - radius; x < centerX+radius; x++ {
            if x >= 0 && x < width && y >= 0 && y < height {
                img.Set(x, y, color.White)
            }
        }
    }

    tmpFile, err := os.CreateTemp("", "test-mask-*.png")
    if err != nil {
        return nil, err
    }

    if err := png.Encode(tmpFile, img); err != nil {
        tmpFile.Close()
        os.Remove(tmpFile.Name())
        return nil, err
    }

    tmpFile.Close()
    return tmpFile, nil
}
```

Usage in tests:
```go
t.Run("edit image", func(t *testing.T) {
    baseImage, err := createTestImage(256, 256, color.RGBA{0, 0, 255, 255})
    if err != nil {
        t.Fatalf("Failed to create test image: %v", err)
    }
    defer os.Remove(baseImage.Name())

    // Use baseImage.Name() as the file path
})
```

#### Option B: Testdata Directory

Create a `testdata/` directory for static test files:

```
v1/
├── testdata/
│   ├── sample.png
│   ├── sample.pdf
│   └── sample.json
├── common_test.go
└── MyNode_test.go
```

Add a helper to get testdata paths:
```go
import "path/filepath"

func getTestdataPath(filename string) string {
    return filepath.Join("testdata", filename)
}
```

**Note**: The `testdata/` directory is a Go convention - it's automatically ignored by the Go build system.

#### Option C: Embed Test Files

For small files, embed them as base64 in code:

```go
import (
    "encoding/base64"
    "os"
)

// Small 1x1 red PNG (embedded)
const testPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg=="

func createEmbeddedTestFile() (*os.File, error) {
    data, err := base64.StdEncoding.DecodeString(testPNGBase64)
    if err != nil {
        return nil, err
    }

    tmpFile, err := os.CreateTemp("", "test-*.png")
    if err != nil {
        return nil, err
    }

    if _, err := tmpFile.Write(data); err != nil {
        tmpFile.Close()
        os.Remove(tmpFile.Name())
        return nil, err
    }

    tmpFile.Close()
    return tmpFile, nil
}
```

#### Option D: Download Test Files

For large files or real-world samples, download during test setup:

```go
import (
    "io"
    "net/http"
    "os"
)

func downloadTestFile(url, localPath string) error {
    // Check if already downloaded
    if _, err := os.Stat(localPath); err == nil {
        return nil
    }

    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    out, err := os.Create(localPath)
    if err != nil {
        return err
    }
    defer out.Close()

    _, err = io.Copy(out, resp.Body)
    return err
}

// In TestMain or test setup
func TestMain(m *testing.M) {
    // Download test files if needed
    os.MkdirAll("testdata", 0755)
    downloadTestFile(
        "https://example.com/sample.pdf",
        "testdata/sample.pdf",
    )
    // ... rest of setup
}
```

### 12.3 Resource Cleanup

When tests create resources (files, connections, etc.), clean them up:

```go
t.Run("upload and cleanup file", func(t *testing.T) {
    if !hasCredentials() {
        t.Skip("No credentials")
    }

    // Create temporary local file
    tmpFile, err := os.CreateTemp("", "test-upload-*.txt")
    if err != nil {
        t.Fatalf("Failed to create temp file: %v", err)
    }
    defer os.Remove(tmpFile.Name())  // Cleanup local file

    tmpFile.WriteString("Test content for upload")
    tmpFile.Close()

    // Upload file
    uploadNode := &FileUpload{}
    q := rtesting.NewQuick(uploadNode)
    q.SetCredential("OptApiKey", "api_key", "api_key")
    q.SetCustom("InFilePath", tmpFile.Name())

    err = q.Run()
    if err != nil {
        t.Fatalf("Upload failed: %v", err)
    }

    // Get file name for cleanup
    fileInfo := q.GetOutput("file").(map[string]interface{})
    fileName := fileInfo["name"].(string)  // e.g., "files/abc123"

    // Cleanup uploaded file via API
    defer func() {
        deleteNode := &FileDelete{}
        dq := rtesting.NewQuick(deleteNode)
        dq.SetCredential("OptApiKey", "api_key", "api_key")
        dq.SetCustom("InFileName", fileName)
        if err := dq.Run(); err != nil {
            t.Logf("Warning: cleanup failed: %v", err)
        }
    }()

    // ... perform assertions ...
})
```

### 12.4 Connection Cleanup

Always cleanup connections in tests:

```go
t.Run("connection lifecycle", func(t *testing.T) {
    if !hasCredentials() {
        t.Skip("No credentials")
    }

    // Connect
    connectNode := &Connect{}
    cq := rtesting.NewQuick(connectNode)
    cq.SetCredential("OptApiKey", "api_key", "api_key")

    err := cq.Run()
    if err != nil {
        t.Fatalf("Connect failed: %v", err)
    }

    connID := cq.GetOutput("connection_id").(string)

    // Always disconnect
    defer func() {
        disconnectNode := &Disconnect{}
        dq := rtesting.NewQuick(disconnectNode)
        dq.SetInput("connection_id", connID)
        dq.Run()
    }()

    // ... test operations using connection ...
})
```

---

## 13. Docker for External Dependencies

Some packages require external services (databases, message queues, etc.). Use Docker Compose for test environments.

### 13.1 Docker Compose Setup

Create `v1/docker-compose.test.yml`:

```yaml
# v1/docker-compose.test.yml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: testuser
      POSTGRES_PASSWORD: testpass
      POSTGRES_DB: testdb
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U testuser -d testdb"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

  mongodb:
    image: mongo:6
    ports:
      - "27017:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: testuser
      MONGO_INITDB_ROOT_PASSWORD: testpass
```

### 13.2 Test Setup with Docker

In `common_test.go`:

```go
package v1

import (
    "os"
    "os/exec"
    "testing"
    "time"

    rtesting "github.com/robomotionio/robomotion-go/testing"
)

var credStore *rtesting.CredentialStore

func TestMain(m *testing.M) {
    // Start Docker services if needed
    if os.Getenv("USE_DOCKER") == "1" {
        if err := startDockerServices(); err != nil {
            panic("Failed to start Docker services: " + err.Error())
        }
        defer stopDockerServices()
    }

    // Initialize credentials
    credStore = rtesting.NewCredentialStore()
    rtesting.LoadDotEnv(".env")

    // Database credentials (from Docker or env)
    credStore.AddCredential("postgres", map[string]interface{}{
        "server":   getEnvOrDefault("PG_HOST", "localhost"),
        "port":     5432,
        "database": getEnvOrDefault("PG_DATABASE", "testdb"),
        "username": getEnvOrDefault("PG_USER", "testuser"),
        "password": getEnvOrDefault("PG_PASSWORD", "testpass"),
    })

    rtesting.InitCredentials(credStore)
    code := m.Run()
    rtesting.ClearCredentials()
    os.Exit(code)
}

func startDockerServices() error {
    cmd := exec.Command("docker-compose", "-f", "docker-compose.test.yml", "up", "-d", "--wait")
    cmd.Dir = "."
    return cmd.Run()
}

func stopDockerServices() {
    cmd := exec.Command("docker-compose", "-f", "docker-compose.test.yml", "down", "-v")
    cmd.Dir = "."
    cmd.Run()
}

func getEnvOrDefault(key, defaultValue string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return defaultValue
}

func hasDockerServices() bool {
    // Check if Postgres is reachable
    cmd := exec.Command("docker-compose", "-f", "docker-compose.test.yml", "ps", "-q", "postgres")
    output, err := cmd.Output()
    return err == nil && len(output) > 0
}
```

### 13.3 Running Tests with Docker

```bash
# Start services and run tests
USE_DOCKER=1 go test -v ./v1/...

# Or manually manage Docker
docker-compose -f v1/docker-compose.test.yml up -d --wait
go test -v ./v1/...
docker-compose -f v1/docker-compose.test.yml down -v
```

### 13.4 Skip Tests Without Services

```go
func TestPostgresQuery(t *testing.T) {
    t.Run("execute query", func(t *testing.T) {
        if !hasDockerServices() && os.Getenv("PG_HOST") == "" {
            t.Skip("No PostgreSQL available, skipping")
        }

        // ... test code ...
    })
}
```

### 13.5 TESTING.md for Docker-Based Tests

```markdown
# Testing Requirements

## Prerequisites

1. Go 1.20+
2. Docker and Docker Compose (for integration tests)

## Running Tests

### Option 1: Docker (Recommended)
```bash
# Start services and run tests
USE_DOCKER=1 go test -v ./v1/...
```

### Option 2: External Database
Create `.env`:
```bash
PG_HOST=your-postgres-host
PG_PORT=5432
PG_DATABASE=testdb
PG_USER=youruser
PG_PASSWORD=yourpassword
```

Then:
```bash
go test -v ./v1/...
```
```

---

## 14. Troubleshooting

| Issue | Cause | Solution |
|-------|-------|----------|
| `No Token Value` | SetCredential with empty itemID | Use `q.SetCredential("OptApiKey", "api_key", "api_key")` — both IDs required |
| `nil interface conversion` | OptVariable with empty name in spec | Update to robomotion-go v1.9.2+ |
| Tests skipped | No credentials in .env | Create `.env` with required API keys |
| API errors | Invalid credentials | Verify API key is valid and has required permissions |
| Rate limit errors | Too many API calls | Add delays between tests or use `SKIP_SLOW_TESTS=1` |
