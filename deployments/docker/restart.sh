#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.prod.yaml"

echo "========================================="
echo "  Text-to-SQL - Restart Production"
echo "========================================="

# Parse flags
NO_CACHE=false
CLEAN=false
for arg in "$@"; do
    case $arg in
        --no-cache) NO_CACHE=true ;;
        --clean)    CLEAN=true ;;
        --help)
            echo "Usage: ./restart.sh [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --no-cache  Build without Docker cache"
            echo "  --clean     Prune build cache before building"
            echo "  --help      Show this help"
            exit 0
            ;;
    esac
done

# Step 1: Stop containers
echo ""
echo "[1/3] Stopping containers..."
docker compose -f "$COMPOSE_FILE" down

# Step 2: Clean if requested
if [ "$CLEAN" = true ]; then
    echo ""
    echo "[2/3] Cleaning Docker build cache..."
    docker builder prune -f
else
    echo ""
    echo "[2/3] Skipping clean (use --clean to prune build cache)"
fi

# Step 3: Build & Start
echo ""
if [ "$NO_CACHE" = true ]; then
    echo "[3/3] Building (no cache) and starting..."
    docker compose -f "$COMPOSE_FILE" build --no-cache
    docker compose -f "$COMPOSE_FILE" up -d
else
    echo "[3/3] Building and starting..."
    docker compose -f "$COMPOSE_FILE" up -d --build
fi

# Show status
echo ""
echo "========================================="
echo "  Containers Status"
echo "========================================="
docker compose -f "$COMPOSE_FILE" ps

echo ""
echo "Done! Access at http://localhost:${NGINX_PORT:-9999}"
echo "Logs: docker compose -f $COMPOSE_FILE logs -f"
