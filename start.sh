#!/bin/bash

# Text-to-SQL Platform Launcher
# This script starts the entire Docker environment

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🚀 Text-to-SQL Platform Launcher"
echo "================================"

# Check Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Error: Docker is not installed. Please install Docker first."
    exit 1
fi

if ! docker info &> /dev/null; then
    echo "❌ Error: Docker is not running. Please start Docker first."
    exit 1
fi

# Forward to docker-start.sh
exec "$SCRIPT_DIR/deployments/docker/docker-start.sh" "${@:-up}"
