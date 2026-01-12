#!/bin/bash
# scripts/setup.sh - Initial setup script for development

set -e

echo "=== Text-to-SQL Development Setup ==="

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check prerequisites
echo -e "\n${YELLOW}Checking prerequisites...${NC}"

command -v go >/dev/null 2>&1 || { echo -e "${RED}Go is required but not installed.${NC}"; exit 1; }
command -v docker >/dev/null 2>&1 || { echo -e "${RED}Docker is required but not installed.${NC}"; exit 1; }
command -v docker-compose >/dev/null 2>&1 || command -v docker compose >/dev/null 2>&1 || { echo -e "${RED}Docker Compose is required but not installed.${NC}"; exit 1; }

echo -e "${GREEN}✓ Go $(go version | awk '{print $3}')${NC}"
echo -e "${GREEN}✓ Docker $(docker --version | awk '{print $3}')${NC}"

# Create config files
echo -e "\n${YELLOW}Setting up configuration...${NC}"

if [ ! -f configs/config.local.yaml ]; then
    cp configs/config.yaml.example configs/config.local.yaml
    echo -e "${GREEN}✓ Created configs/config.local.yaml${NC}"
else
    echo -e "${YELLOW}⚠ configs/config.local.yaml already exists${NC}"
fi

if [ ! -f .env ]; then
    cp .env.example .env
    # Generate random JWT secret (macOS compatible)
    JWT_SECRET=$(openssl rand -base64 32 | tr -d '\n' | tr '/' '_')
    # macOS compatible sed
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s|JWT_SECRET=.*|JWT_SECRET=${JWT_SECRET}|" .env
    else
        sed -i "s|JWT_SECRET=.*|JWT_SECRET=${JWT_SECRET}|" .env
    fi
    echo -e "${GREEN}✓ Created .env with generated JWT_SECRET${NC}"
else
    echo -e "${YELLOW}⚠ .env already exists${NC}"
fi

# Download dependencies
echo -e "\n${YELLOW}Downloading Go dependencies...${NC}"
go mod download
go mod tidy
echo -e "${GREEN}✓ Dependencies downloaded${NC}"

# Check if containers already exist on standard ports
REDIS_PORT=6379
POSTGRES_PORT=5432
USE_EXISTING=false

if lsof -i :6379 >/dev/null 2>&1 && lsof -i :5432 >/dev/null 2>&1; then
    echo -e "\n${YELLOW}Existing Redis and PostgreSQL detected on standard ports.${NC}"
    echo -e "${YELLOW}Using existing services...${NC}"
    USE_EXISTING=true
fi

if [ "$USE_EXISTING" = false ]; then
    # Start services
    echo -e "\n${YELLOW}Starting Docker services...${NC}"
    if command -v docker-compose >/dev/null 2>&1; then
        docker-compose -f deployments/docker/docker-compose.yaml up -d postgres redis
    else
        docker compose -f deployments/docker/docker-compose.yaml up -d postgres redis
    fi
    echo -e "${GREEN}✓ PostgreSQL and Redis started${NC}"
    
    # Wait for PostgreSQL
    echo -e "\n${YELLOW}Waiting for PostgreSQL to be ready...${NC}"
    CONTAINER_NAME=$(docker ps --filter "ancestor=postgres:15-alpine" --format "{{.Names}}" | head -1)
    if [ -z "$CONTAINER_NAME" ]; then
        CONTAINER_NAME="docker-postgres-1"
    fi
    for i in {1..30}; do
        if docker exec "$CONTAINER_NAME" pg_isready -U texttosql >/dev/null 2>&1; then
            echo -e "${GREEN}✓ PostgreSQL is ready${NC}"
            break
        fi
        echo -n "."
        sleep 1
    done
else
    echo -e "${GREEN}✓ Using existing PostgreSQL and Redis${NC}"
fi

# Build the application
echo -e "\n${YELLOW}Building application...${NC}"
mkdir -p bin
go build -o bin/server ./cmd/server
echo -e "${GREEN}✓ Built bin/server${NC}"

# Run tests
echo -e "\n${YELLOW}Running tests...${NC}"
go test ./... -count=1 -short
echo -e "${GREEN}✓ All tests passed${NC}"

echo -e "\n${GREEN}=== Setup Complete ===${NC}"
echo -e "\nTo start the server:"
echo -e "  ${YELLOW}./bin/server${NC}"
echo -e "  or"
echo -e "  ${YELLOW}make run${NC}"
echo -e "\nTo start Ollama (optional):"
echo -e "  ${YELLOW}docker compose -f deployments/docker/docker-compose.yaml up -d ollama${NC}"
echo -e "  ${YELLOW}docker exec -it docker-ollama-1 ollama pull llama3${NC}"
echo -e "\nAPI will be available at: ${GREEN}http://localhost:8080${NC}"
