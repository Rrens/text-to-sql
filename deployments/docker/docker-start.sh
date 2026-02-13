#!/bin/bash

# ============================================
# Text-to-SQL Docker Environment Starter
# ============================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DOCKER_DIR="$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Text-to-SQL Docker Environment${NC}"
echo "============================================"

# Check Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed. Please install Docker first.${NC}"
    exit 1
fi

if ! docker info &> /dev/null; then
    echo -e "${RED}❌ Docker is not running. Please start Docker first.${NC}"
    exit 1
fi

# Check for .env.docker
ENV_FILE="$PROJECT_ROOT/.env.docker"
if [ ! -f "$ENV_FILE" ]; then
    echo -e "${YELLOW}⚠️  No .env.docker found. Creating from template...${NC}"
    if [ -f "$PROJECT_ROOT/.env.docker.example" ]; then
        cp "$PROJECT_ROOT/.env.docker.example" "$ENV_FILE"
    fi
fi

cd "$DOCKER_DIR"

# Parse arguments
ACTION="${1:-up}"

case "$ACTION" in
    up|start)
        echo -e "${GREEN}📦 Building and starting services...${NC}"
        docker compose --env-file "$ENV_FILE" up --build -d
        
        echo ""
        echo -e "${GREEN}✅ Services started successfully!${NC}"
        echo "============================================"
        echo -e "🌐 Application    : ${BLUE}http://localhost${NC}"
        echo -e "🔌 Backend API    : ${BLUE}http://localhost/api/v1/health${NC}"
        echo -e "🗄️  PostgreSQL     : localhost:55432"
        echo -e "📦 Redis          : localhost:56379"
        echo "============================================"
        echo ""
        echo "Useful commands:"
        echo "  View logs:     docker compose logs -f"
        echo "  Stop:          $0 down"
        echo "  Restart:       $0 restart"
        ;;
    
    down|stop)
        echo -e "${YELLOW}🛑 Stopping services...${NC}"
        docker compose --env-file "$ENV_FILE" down
        echo -e "${GREEN}✅ Services stopped.${NC}"
        ;;
    
    restart)
        echo -e "${YELLOW}🔄 Restarting services...${NC}"
        docker compose --env-file "$ENV_FILE" down
        docker compose --env-file "$ENV_FILE" up --build -d
        echo -e "${GREEN}✅ Services restarted.${NC}"
        ;;
    
    logs)
        docker compose --env-file "$ENV_FILE" logs -f "${@:2}"
        ;;
    
    ps|status)
        docker compose --env-file "$ENV_FILE" ps
        ;;
    
    clean)
        echo -e "${RED}🗑️  Cleaning up all resources (including volumes)...${NC}"
        docker compose --env-file "$ENV_FILE" down -v --rmi local
        echo -e "${GREEN}✅ Cleanup complete.${NC}"
        ;;
    
    build)
        echo -e "${BLUE}🔨 Building images...${NC}"
        docker compose --env-file "$ENV_FILE" build --no-cache
        echo -e "${GREEN}✅ Build complete.${NC}"
        ;;
    
    *)
        echo "Usage: $0 {up|down|restart|logs|ps|clean|build}"
        echo ""
        echo "Commands:"
        echo "  up|start    Build and start all services"
        echo "  down|stop   Stop all services"
        echo "  restart     Restart all services"
        echo "  logs        View logs (add service name to filter)"
        echo "  ps|status   Show service status"
        echo "  clean       Remove all containers, volumes, and images"
        echo "  build       Build images without cache"
        exit 1
        ;;
esac
