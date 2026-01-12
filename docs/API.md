# Text-to-SQL API Documentation

Base URL: `http://localhost:8081/api/v1`

## Authentication

### Register

**POST** `/auth/register`

Register a new user account.

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "strongpassword123"
}
```

**Response (201 Created):**

```json
{
  "success": true,
  "data": {
    "id": "e6a7f571-fe3b-4c11-9042-a363a82f9789",
    "email": "user@example.com"
  }
}
```

### Login

**POST** `/auth/login`

Authenticate and receive access/refresh tokens.

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "strongpassword123"
}
```

**Response (200 OK):**

```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUz...",
    "refresh_token": "eyJhbGciOiJIUz...",
    "expires_in": 900
  }
}
```

### Refresh Token

**POST** `/auth/refresh`

Get a new access token using a refresh token.

**Request Body:**

```json
{
  "refresh_token": "eyJhbGciOiJIUz..."
}
```

---

## Workspaces

### List Workspaces

**GET** `/workspaces`

List all workspaces you are a member of.

**Response (200 OK):**

```json
{
  "success": true,
  "data": [
    {
      "id": "3e26775a-fdbc-41ca-acde-ee18ea971155",
      "name": "Production Workspace",
      "created_at": "2026-01-12T10:00:00Z"
    }
  ]
}
```

### Create Workspace

**POST** `/workspaces`

**Request Body:**

```json
{
  "name": "My New Workspace"
}
```

### Get Workspace

**GET** `/workspaces/{workspace_id}`

### Update Workspace

**PATCH** `/workspaces/{workspace_id}`

**Request Body:**

```json
{
  "name": "Updated Name"
}
```

### Delete Workspace

**DELETE** `/workspaces/{workspace_id}`

---

## Connections

Manage database connections within a workspace.

### List Connections

**GET** `/workspaces/{workspace_id}/connections`

### Create Connection

**POST** `/workspaces/{workspace_id}/connections`

**Request Body:**

```json
{
  "name": "Prod DB",
  "database_type": "postgres", // postgres, mysql, clickhouse
  "host": "db.example.com",
  "port": 5432,
  "database": "analytics",
  "username": "reader",
  "password": "secure_password",
  "ssl_mode": "require", // disable, require, verify-ca, verify-full
  "read_only": true
}
```

### Get Schema

**GET** `/workspaces/{workspace_id}/connections/{connection_id}/schema`

Returns the cached schema (tables, columns) for the connection.

---

## Query Generation & Execution

### Execute Query

**POST** `/workspaces/{workspace_id}/query`

Generate SQL from natural language and execute it against the database.

**Request Body:**

```json
{
  "connection_id": "uuid-here",
  "question": "Show me top 10 users by total spend in 2024",
  "llm_provider": "gemini", // openai, anthropic, gemini, ollama, deepseek
  "llm_model": "gemini-1.5-pro", // optional, defaults to provider default
  "execute": true
}
```

**Response (200 OK):**

```json
{
  "success": true,
  "data": {
    "request_id": "req-123",
    "question": "Show me top 10 users by total spend in 2024",
    "sql": "SELECT u.name, SUM(o.total) as spend FROM users u ...",
    "result": {
      "columns": ["name", "spend"],
      "rows": [
        ["Alice", 5000],
        ["Bob", 4500]
      ],
      "row_count": 2,
      "truncated": false
    },
    "metadata": {
      "execution_time_ms": 150,
      "llm_latency_ms": 800,
      "tokens_used": 350
    }
  }
}
```

### Generate SQL Only

**POST** `/workspaces/{workspace_id}/generate`

Same as `/query` but `execute` is forced to `false`. Returns the generated SQL without running it.

---

## System

### Health Check

**GET** `/health`

### Readiness Check

**GET** `/ready`

### List LLM Providers

**GET** `/llm-providers`

Returns successfully configured LLM providers and their available models.

**Response:**

```json
{
  "success": true,
  "data": {
    "default_provider": "gemini",
    "providers": [
      {
        "name": "gemini",
        "models": ["gemini-1.5-flash", "gemini-1.5-pro"],
        "default": true
      },
      {
        "name": "ollama",
        "models": ["llama3", "mistral"],
        "default": false
      }
    ]
  }
}
```

### Flush Cache

**POST** `/cache/flush`

Flush all schema cache from Redis.

**Headers:** `Authorization: Bearer <token>`

**Response (200 OK):**

```json
{
  "success": true,
  "data": {
    "message": "cache flushed successfully",
    "keys_deleted": 5
  }
}
```
