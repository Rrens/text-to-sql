#!/bin/bash

# Text-to-SQL Platform Launcher

echo "🚀 Initializing Text-to-SQL Platform..."

# Check requirements
if ! command -v docker &> /dev/null; then
    echo "❌ Error: Docker is not installed. Please install Docker Desktop first."
    exit 1
fi

echo "📦 Building and starting services..."
cd deployments/docker || exit 1

# Start Docker Compose
docker-compose up --build -d

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Application started successfully!"
    echo "---------------------------------------------------"
    echo "🌐 Frontend URL : http://localhost"
    echo "🔌 Backend API  : http://localhost/api/v1/health"
    echo "---------------------------------------------------"
    echo "Run 'docker-compose logs -f' in deployments/docker to see logs."
else
    echo "❌ Failed to start application."
    exit 1
fi
