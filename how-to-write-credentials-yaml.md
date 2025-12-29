# How to Write credentials.yaml

This guide explains how to create properly formatted `credentials.yaml` files for Robomotion packages.

## Quick Start

Every credential needs **two required fields**:
- `name`: snake_case identifier (used in code)
- `title`: Human-readable display name (shown as "Create [title]..." in UI)

```yaml
credentials:
  - name: api_key
    title: "My Service API Key"
    category: 4
    fields:
      - name: value
        title: "API Key"
        type: password
        required: true
```

---

## File Structure

Every `credentials.yaml` must start with the `credentials:` root element:

```yaml
credentials:
  - name: first_credential
    # ... credential definition

  - name: second_credential
    # ... credential definition
```

---

## Credential Properties

### Required Properties

| Property | Type | Description |
|----------|------|-------------|
| `name` | string | **snake_case** identifier (e.g., `api_key`, `database_login`) |
| `title` | string | Human-readable name shown in UI (e.g., `"OpenAI API Key"`) |
| `category` | integer | Credential category (see table below) |
| `fields` | array | List of input fields |

### Optional Properties

| Property | Type | Description |
|----------|------|-------------|
| `description` | string | Explains what this credential is for |
| `default` | boolean | If `true`, this is the default credential for the category |
| `applicableTo` | array | List of node IDs that can use this credential |
| `compositeField` | string | Field name for composite credentials (e.g., `content` for OAuth2) |
| `help` | array | Step-by-step instructions for obtaining the credential |
| `resources` | array | Links to external documentation |

---

## Categories

| Category | Type | Use Case |
|----------|------|----------|
| 1 | Login | Username + Password (basic auth, Windows login) |
| 2 | Email | Email credentials with SMTP settings |
| 4 | API Key | Single API key/token |
| 5 | Database | Connection string with host, port, database, credentials |
| 6 | Document | JSON blob (OAuth2, Service Accounts) |
| 7 | AES Key | Encryption keys |
| 8 | RSA Key | RSA key pairs |

---

## Field Properties

### Required Field Properties

| Property | Type | Description |
|----------|------|-------------|
| `name` | string | Field identifier (snake_case) |
| `title` | string | Field label shown in UI |
| `type` | string | Input type (see valid types below) |

### Optional Field Properties

| Property | Type | Description |
|----------|------|-------------|
| `required` | boolean | Whether the field is required |
| `description` | string | Help text for the field |
| `placeholder` | string | Placeholder text in input |
| `default` | any | Default value |
| `value` | any | Fixed value (for readonly/hidden fields) |

### Valid Field Types

| Type | Description |
|------|-------------|
| `text` | Single-line text input |
| `password` | Masked password input |
| `textarea` | Multi-line text input |
| `number` | Numeric input |
| `select` | Dropdown selection |
| `hidden` | Hidden field (not shown in UI) |
| `readonly` | Read-only display field |

> **Important:** `string` is NOT a valid type. Use `text` or `password` instead.

---

## Examples

### API Key (Category 4)

```yaml
credentials:
  - name: api_key
    title: "OpenAI API Key"
    category: 4
    description: "API key for OpenAI authentication"
    applicableTo:
      - Robomotion.OpenAI.Connect
      - Robomotion.OpenAI.GenerateText
    fields:
      - name: value
        title: "API Key"
        type: password
        required: true
        description: "Your OpenAI API key"
    help:
      - "1. Go to platform.openai.com"
      - "2. Navigate to API Keys section"
      - "3. Create a new secret key"
    resources:
      - label: "OpenAI Platform"
        url: "https://platform.openai.com/api-keys"
```

### Login Credentials (Category 1)

```yaml
credentials:
  - name: database_login
    title: "Database Credentials"
    category: 1
    description: "Username and password for database authentication"
    applicableTo:
      - Robomotion.MySQL.Connect
    fields:
      - name: username
        title: "Username"
        type: text
        required: true
        description: "Database username"
      - name: password
        title: "Password"
        type: password
        required: true
        description: "Database password"
```

### Database Connection (Category 5)

```yaml
credentials:
  - name: mysql_database
    title: "MySQL Database"
    category: 5
    description: "MySQL database connection settings"
    applicableTo:
      - Robomotion.MySQL.Connect
      - Robomotion.MySQL.Query
    fields:
      - name: type
        title: "Database Type"
        type: hidden
        default: mysql
      - name: server
        title: "Server"
        type: text
        required: true
        placeholder: "localhost"
        description: "Database server hostname"
      - name: port
        title: "Port"
        type: text
        placeholder: "3306"
        description: "Database port"
      - name: database
        title: "Database"
        type: text
        required: true
        description: "Database name"
      - name: username
        title: "Username"
        type: text
        required: true
      - name: password
        title: "Password"
        type: password
        required: true
```

### OAuth2 / Service Account (Category 6)

```yaml
credentials:
  - name: oauth2
    title: "OAuth2 Credentials"
    category: 6
    default: true
    compositeField: content
    description: "OAuth2 authorization for API access"
    applicableTo:
      - Robomotion.Google.Drive.Connect
    fields:
      - name: _redirect_url
        title: "Redirect URL"
        type: readonly
        value: "http://localhost:9876/oauth2/callback"
        description: "Add this to your OAuth app's Redirect URIs"
      - name: client_id
        title: "Client ID"
        type: text
        required: true
        description: "OAuth 2.0 Client ID"
      - name: client_secret
        title: "Client Secret"
        type: password
        required: true
        description: "OAuth 2.0 Client Secret"
    help:
      - "1. Go to Google Cloud Console"
      - "2. Create OAuth 2.0 credentials"
      - "3. Add the redirect URL above"
      - "4. Copy Client ID and Secret"
    resources:
      - label: "Google Cloud Console"
        url: "https://console.cloud.google.com/"
```

### Multiple Credentials Per Package

```yaml
credentials:
  - name: api_key
    title: "API Key"
    category: 4
    default: true
    fields:
      - name: value
        title: "API Key"
        type: password
        required: true
    applicableTo:
      - Robomotion.Service.Connect

  - name: secret_key
    title: "Secret Key"
    category: 4
    fields:
      - name: value
        title: "Secret Key"
        type: password
        required: true
    applicableTo:
      - Robomotion.Service.Connect
```

### AES Encryption Key (Category 7)

```yaml
credentials:
  - name: aes_encryption_key
    title: "AES Encryption Key"
    category: 7
    description: "AES key for encryption/decryption operations"
    applicableTo:
      - Robomotion.Crypto.Encrypt
      - Robomotion.Crypto.Decrypt
    fields:
      - name: value
        title: "AES Key"
        type: password
        required: true
        description: "Hex-encoded AES key (32 or 64 characters)"
```

---

## Common Mistakes to Avoid

### 1. Using `type:` Instead of `name:`

```yaml
# WRONG - deprecated
credentials:
  - type: API Key
    category: 4
    fields: ...

# CORRECT
credentials:
  - name: api_key
    title: "API Key"
    category: 4
    fields: ...
```

### 2. Using Invalid Field Type `string`

```yaml
# WRONG
fields:
  - name: username
    type: string

# CORRECT
fields:
  - name: username
    type: text
```

### 3. Non-Snake_Case Names

```yaml
# WRONG
credentials:
  - name: "My API Key"
  - name: "APIKey"
  - name: "api-key"

# CORRECT
credentials:
  - name: api_key
  - name: my_api_key
  - name: service_token
```

### 4. Missing `credentials:` Wrapper

```yaml
# WRONG - no root element
name: api_key
title: "API Key"
category: 4
fields: ...

# CORRECT
credentials:
  - name: api_key
    title: "API Key"
    category: 4
    fields: ...
```

### 5. Using Deprecated `secret: true`

```yaml
# WRONG - deprecated
fields:
  - name: value
    type: string
    secret: true

# CORRECT - use type: password instead
fields:
  - name: value
    type: password
```

---

## Checklist

Before committing your `credentials.yaml`:

- [ ] File starts with `credentials:`
- [ ] Every credential has `name:` (snake_case)
- [ ] Every credential has `title:` (human-readable)
- [ ] Every credential has `category:` (1-8)
- [ ] Every credential has `fields:` array
- [ ] All field types are valid (`text`, `password`, `textarea`, `number`, `select`, `hidden`, `readonly`)
- [ ] No `type:` at credential root level (use `name:` instead)
- [ ] No `secret: true` in fields (use `type: password` instead)
- [ ] Sensitive fields use `type: password`
- [ ] `applicableTo` lists all applicable node IDs

---

## Reference

For the complete specification, see:
- `robomotion-new-designer/public/credentials/CREDENTIALS.md`
