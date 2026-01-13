#!/bin/bash
# scripts/test-api.sh - Test the API endpoints

set -e

BASE_URL="${API_URL:-http://localhost:4081}"
API="$BASE_URL/api/v1"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=== Text-to-SQL API Test ==="
echo "Base URL: $BASE_URL"

# Health check
echo -e "\n${YELLOW}1. Health Check${NC}"
HEALTH=$(curl -s "$API/health")
echo "$HEALTH" | jq .
if echo "$HEALTH" | jq -e '.success == true' > /dev/null; then
    echo -e "${GREEN}✓ Health check passed${NC}"
else
    echo -e "${RED}✗ Health check failed${NC}"
    exit 1
fi

# Ready check
echo -e "\n${YELLOW}2. Ready Check${NC}"
READY=$(curl -s "$API/ready")
echo "$READY" | jq .
if echo "$READY" | jq -e '.success == true' > /dev/null; then
    echo -e "${GREEN}✓ Ready check passed${NC}"
else
    echo -e "${RED}✗ Ready check failed${NC}"
    exit 1
fi

# Register user
EMAIL="test-$(date +%s)@example.com"
PASSWORD="testpassword123"

echo -e "\n${YELLOW}3. Register User${NC}"
REGISTER=$(curl -s -X POST "$API/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"$EMAIL\", \"password\": \"$PASSWORD\"}")
echo "$REGISTER" | jq .
if echo "$REGISTER" | jq -e '.success == true' > /dev/null; then
    echo -e "${GREEN}✓ Registration successful${NC}"
else
    echo -e "${RED}✗ Registration failed${NC}"
    exit 1
fi

# Login
echo -e "\n${YELLOW}4. Login${NC}"
LOGIN=$(curl -s -X POST "$API/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"$EMAIL\", \"password\": \"$PASSWORD\"}")
echo "$LOGIN" | jq .
ACCESS_TOKEN=$(echo "$LOGIN" | jq -r '.data.access_token')
REFRESH_TOKEN=$(echo "$LOGIN" | jq -r '.data.refresh_token')
if [ "$ACCESS_TOKEN" != "null" ] && [ -n "$ACCESS_TOKEN" ]; then
    echo -e "${GREEN}✓ Login successful${NC}"
else
    echo -e "${RED}✗ Login failed${NC}"
    exit 1
fi

# Create workspace
echo -e "\n${YELLOW}5. Create Workspace${NC}"
WORKSPACE=$(curl -s -X POST "$API/workspaces" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"name": "Test Workspace"}')
echo "$WORKSPACE" | jq .
WORKSPACE_ID=$(echo "$WORKSPACE" | jq -r '.data.id')
if [ "$WORKSPACE_ID" != "null" ] && [ -n "$WORKSPACE_ID" ]; then
    echo -e "${GREEN}✓ Workspace created: $WORKSPACE_ID${NC}"
else
    echo -e "${RED}✗ Workspace creation failed${NC}"
    exit 1
fi

# List workspaces
echo -e "\n${YELLOW}6. List Workspaces${NC}"
WORKSPACES=$(curl -s "$API/workspaces" \
    -H "Authorization: Bearer $ACCESS_TOKEN")
echo "$WORKSPACES" | jq .
echo -e "${GREEN}✓ Listed workspaces${NC}"

# Get LLM providers
echo -e "\n${YELLOW}7. List LLM Providers${NC}"
PROVIDERS=$(curl -s "$API/llm-providers" \
    -H "Authorization: Bearer $ACCESS_TOKEN")
echo "$PROVIDERS" | jq .
echo -e "${GREEN}✓ Listed LLM providers${NC}"

# Refresh token
echo -e "\n${YELLOW}8. Refresh Token${NC}"
REFRESH=$(curl -s -X POST "$API/auth/refresh" \
    -H "Content-Type: application/json" \
    -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}")
echo "$REFRESH" | jq .
NEW_TOKEN=$(echo "$REFRESH" | jq -r '.data.access_token')
if [ "$NEW_TOKEN" != "null" ] && [ -n "$NEW_TOKEN" ]; then
    echo -e "${GREEN}✓ Token refreshed${NC}"
else
    echo -e "${RED}✗ Token refresh failed${NC}"
fi

# Delete workspace
echo -e "\n${YELLOW}9. Delete Workspace${NC}"
DELETE=$(curl -s -X DELETE "$API/workspaces/$WORKSPACE_ID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -w "%{http_code}")
if [ "$DELETE" = "204" ]; then
    echo -e "${GREEN}✓ Workspace deleted${NC}"
else
    echo "Response: $DELETE"
    echo -e "${RED}✗ Workspace deletion failed${NC}"
fi

echo -e "\n${GREEN}=== All API Tests Passed ===${NC}"
