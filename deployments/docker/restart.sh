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
FE_ONLY=false
BE_ONLY=false
for arg in "$@"; do
    case $arg in
        --no-cache) NO_CACHE=true ;;
        --clean)    CLEAN=true ;;
        --fe)       FE_ONLY=true ;;
        --be)       BE_ONLY=true ;;
        --help)
            echo "Usage: ./restart.sh [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --no-cache  Build without Docker cache"
            echo "  --clean     Prune build cache before building"
            echo "  --fe        Rebuild frontend only"
            echo "  --be        Rebuild backend only"
            echo "  --help      Show this help"
            exit 0
            ;;
    esac
done

# Step 1: Stop containers
echo ""
if [ "$FE_ONLY" = true ]; then
    echo "[1/3] Stopping frontend container..."
    docker compose -f "$COMPOSE_FILE" stop frontend nginx
elif [ "$BE_ONLY" = true ]; then
    echo "[1/3] Stopping backend container..."
    docker compose -f "$COMPOSE_FILE" stop app
else
    echo "[1/3] Stopping all containers..."
    docker compose -f "$COMPOSE_FILE" down
fi

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
if [ "$FE_ONLY" = true ]; then
    if [ "$NO_CACHE" = true ]; then
        echo "[3/3] Rebuilding frontend (no cache) and starting..."
        docker compose -f "$COMPOSE_FILE" build --no-cache frontend
    else
        echo "[3/3] Rebuilding frontend and starting..."
        docker compose -f "$COMPOSE_FILE" build frontend
    fi
    docker compose -f "$COMPOSE_FILE" up -d frontend nginx
elif [ "$BE_ONLY" = true ]; then
    if [ "$NO_CACHE" = true ]; then
        echo "[3/3] Rebuilding backend (no cache) and starting..."
        docker compose -f "$COMPOSE_FILE" build --no-cache app
    else
        echo "[3/3] Rebuilding backend and starting..."
        docker compose -f "$COMPOSE_FILE" build app
    fi
    docker compose -f "$COMPOSE_FILE" up -d app
else
    if [ "$NO_CACHE" = true ]; then
        echo "[3/3] Building all (no cache) and starting..."
        docker compose -f "$COMPOSE_FILE" build --no-cache
        docker compose -f "$COMPOSE_FILE" up -d
    else
        echo "[3/3] Building all and starting..."
        docker compose -f "$COMPOSE_FILE" up -d --build
    fi
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
