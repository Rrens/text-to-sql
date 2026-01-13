#!/bin/bash
echo "🚀 Starting application..."
docker-compose up -d

echo ""
echo "✅ Done! Application is running at http://localhost"

echo "API: http://localhost:4081"
echo "DB:  localhost:55432"
