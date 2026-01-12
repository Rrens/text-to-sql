# Text-to-SQL API Platform

Production-ready, multi-tenant Text-to-SQL API platform built with Go.

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- 🏢 **Multi-tenant**: Workspace-based isolation for multiple users
- 🗄️ **Multi-database**: PostgreSQL, ClickHouse, MySQL support
- 🤖 **Multi-LLM**: OpenAI, Anthropic, Gemini, Ollama (local), DeepSeek
- 🔒 **Secure**: Encrypted credentials, JWT auth, SQL validation
- 🚀 **On-premises**: No cloud dependencies, self-hosted

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- (Optional) Ollama for local LLM

### 1. Clone & Setup

```bash
git clone https://github.com/Rrens/text-to-sql.git
cd text-to-sql
make setup
```

Or manually:

```bash
# Copy configs
cp configs/config.yaml.example configs/config.local.yaml
cp .env.example .env

# Start dependencies
make docker-up

# Build & run
make run
```

### 2. Pull Ollama Model (Optional)

```bash
docker exec -it mcp-ollama-1 ollama pull llama3
```

### 3. Test the API

```bash
# Health check
curl http://localhost:8080/api/v1/health

# Or run full API test
make test-api
```

## API Usage

### Authentication

```bash
# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "securepass123"}'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "securepass123"}'
```

### Create Workspace

```bash
curl -X POST http://localhost:8080/api/v1/workspaces \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "My Workspace"}'
```

### Add Database Connection

```bash
curl -X POST http://localhost:8080/api/v1/workspaces/<workspace_id>/connections \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production DB",
    "database_type": "postgres",
    "host": "db.example.com",
    "port": 5432,
    "database": "myapp",
    "username": "readonly",
    "password": "secret",
    "read_only": true
  }'
```

### Execute Text-to-SQL Query

```bash
curl -X POST http://localhost:8080/api/v1/workspaces/<workspace_id>/query \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "connection_id": "<connection_id>",
    "question": "Show me top 10 users by order count",
    "llm_provider": "ollama",
    "execute": true
  }'
```

## API Endpoints

| Method | Endpoint                                   | Description          |
| ------ | ------------------------------------------ | -------------------- |
| POST   | `/auth/register`                           | Register new user    |
| POST   | `/auth/login`                              | Login and get tokens |
| POST   | `/auth/refresh`                            | Refresh access token |
| GET    | `/workspaces`                              | List workspaces      |
| POST   | `/workspaces`                              | Create workspace     |
| GET    | `/workspaces/{id}`                         | Get workspace        |
| PATCH  | `/workspaces/{id}`                         | Update workspace     |
| DELETE | `/workspaces/{id}`                         | Delete workspace     |
| GET    | `/workspaces/{id}/connections`             | List connections     |
| POST   | `/workspaces/{id}/connections`             | Create connection    |
| GET    | `/workspaces/{id}/connections/{id}`        | Get connection       |
| DELETE | `/workspaces/{id}/connections/{id}`        | Delete connection    |
| GET    | `/workspaces/{id}/connections/{id}/schema` | Get DB schema        |
| POST   | `/workspaces/{id}/query`                   | Execute text-to-SQL  |
| POST   | `/workspaces/{id}/generate`                | Generate SQL only    |
| GET    | `/llm-providers`                           | List LLM providers   |
| GET    | `/health`                                  | Health check         |
| GET    | `/ready`                                   | Readiness check      |

See [docs/openapi.yaml](docs/openapi.yaml) for full API specification.
A Postman collection is also available at [docs/postman_collection.json](docs/postman_collection.json) - import this file directly into Postman.

## Configuration

### Environment Variables

| Variable            | Description                 | Required |
| ------------------- | --------------------------- | -------- |
| `JWT_SECRET`        | JWT signing key (32+ chars) | Yes      |
| `POSTGRES_PASSWORD` | Platform database password  | Yes      |
| `REDIS_PASSWORD`    | Redis password              | No       |
| `OPENAI_API_KEY`    | OpenAI API key              | No       |
| `ANTHROPIC_API_KEY` | Anthropic API key           | No       |
| `GEMINI_API_KEY`    | Google Gemini API key       | No       |
| `DEEPSEEK_API_KEY`  | DeepSeek API key            | No       |
| `OLLAMA_HOST`       | Ollama server URL           | No       |

### LLM Providers

| Provider   | Local | API Key | Best For             |
| ---------- | ----- | ------- | -------------------- |
| **Ollama** | ✅    | No      | On-premises, privacy |
| OpenAI     | ❌    | Yes     | Best quality         |
| Anthropic  | ❌    | Yes     | Complex queries      |
| Gemini     | ❌    | Yes     | Fast & multimodal    |
| DeepSeek   | ❌    | Yes     | Code-focused         |

### Supported Databases

| Database   | Type         | Features            |
| ---------- | ------------ | ------------------- |
| PostgreSQL | `postgres`   | Full support, RLS   |
| ClickHouse | `clickhouse` | Analytics, columnar |
| MySQL      | `mysql`      | Standard SQL        |

## Deployment

### Docker Compose (Recommended)

```bash
docker-compose -f deployments/docker/docker-compose.yaml up -d
```

### Kubernetes

```bash
kubectl apply -f deployments/kubernetes/deployment.yaml
```

### Systemd (Bare Metal)

```bash
sudo cp deployments/systemd/text-to-sql.service /etc/systemd/system/
sudo systemctl enable text-to-sql
sudo systemctl start text-to-sql
```

## Project Structure

```
.
├── cmd/server/           # Application entrypoint
├── internal/
│   ├── api/              # HTTP handlers & middleware
│   ├── config/           # Configuration management
│   ├── domain/           # Domain models
│   ├── llm/              # LLM provider adapters
│   ├── mcp/              # Database adapters
│   ├── repository/       # Data access layer
│   ├── security/         # Auth, encryption
│   └── service/          # Business logic
├── migrations/           # Database migrations
├── configs/              # Configuration files
├── deployments/          # Docker, K8s, systemd
├── docs/                 # API documentation
└── scripts/              # Utility scripts
```

## Security

- **Credentials**: Encrypted with AES-256-GCM
- **Authentication**: JWT with access/refresh tokens
- **SQL Validation**: Read-only enforcement, blocked patterns
- **Rate Limiting**: Per-user request limits
- **Workspace Isolation**: Multi-tenant architecture

## Development

```bash
# Run tests
make test

# Run with coverage
make test-coverage

# Lint code
make lint

# Format code
make fmt

# Build for all platforms
make build-all
```

## License

MIT License - see [LICENSE](LICENSE) for details.
