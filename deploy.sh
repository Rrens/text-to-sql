#!/bin/bash

# Configuration
COMPOSE_FILE="deployments/docker/docker-compose.prod.yaml"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Starting deployment...${NC}"

# Check if docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Docker is not installed. Please install Docker first.${NC}"
    exit 1
fi

# Check for .env file
if [ ! -f .env ]; then
    echo -e "${YELLOW}No .env file found. Creating from .env.example if exists...${NC}"
    if [ -f .env.example ]; then
        cp .env.example .env
        echo -e "${GREEN}Created .env from .env.example. Please edit it with your production secrets!${NC}"
    else
        echo -e "${RED}Error: .env file missing and no .env.example found.${NC}"
        exit 1
    fi
fi

# Pull latest images (if using pre-built images) or build locally
echo -e "${YELLOW}Building and starting services...${NC}"

# Build and start containers
# We use --build to ensure we have the latest code
docker compose -f $COMPOSE_FILE up -d --build --remove-orphans

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Deployment successful!${NC}"
    echo -e "${GREEN}Services are running at:${NC}"
    echo -e "  Frontend: http://localhost (or your server IP)"
    echo -e "  Backend API: http://localhost/api/v1 (internal via nginx)"
else
    echo -e "${RED}Deployment failed.${NC}"
    exit 1
fi

# Show status
docker compose -f $COMPOSE_FILE ps
