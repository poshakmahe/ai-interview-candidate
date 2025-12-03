#!/bin/bash

# SecureVault Health Check Script
# This script verifies that all services are running correctly

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}SecureVault Health Check${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""

# Check if Docker is running
echo -n "Checking Docker... "
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}✗ FAIL${NC}"
    echo -e "${RED}Docker is not running. Please start Docker Desktop.${NC}"
    exit 1
fi
echo -e "${GREEN}✓ OK${NC}"

# Check if containers are running
echo -n "Checking containers... "
RUNNING_CONTAINERS=$(docker-compose ps --services --filter "status=running" 2>/dev/null | wc -l)
if [ "$RUNNING_CONTAINERS" -lt 3 ]; then
    echo -e "${RED}✗ FAIL${NC}"
    echo -e "${YELLOW}Expected 3 containers (postgres, backend, frontend), found $RUNNING_CONTAINERS${NC}"
    echo -e "${YELLOW}Run: docker-compose up -d${NC}"
    exit 1
fi
echo -e "${GREEN}✓ OK${NC} ($RUNNING_CONTAINERS/3 running)"

# Check PostgreSQL
echo -n "Checking PostgreSQL... "
if docker-compose exec -T postgres pg_isready -U postgres > /dev/null 2>&1; then
    echo -e "${GREEN}✓ OK${NC}"
else
    echo -e "${RED}✗ FAIL${NC}"
    echo -e "${YELLOW}PostgreSQL is not ready. Wait a few seconds and try again.${NC}"
    exit 1
fi

# Check Backend API
echo -n "Checking Backend API (http://localhost:8080)... "
BACKEND_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health 2>/dev/null || echo "000")
if [ "$BACKEND_RESPONSE" -eq 200 ]; then
    echo -e "${GREEN}✓ OK${NC}"
else
    echo -e "${RED}✗ FAIL${NC} (HTTP $BACKEND_RESPONSE)"
    echo -e "${YELLOW}Backend is not responding. Check logs: docker-compose logs backend${NC}"
    exit 1
fi

# Check Frontend
echo -n "Checking Frontend (http://localhost:3000)... "
FRONTEND_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3000 2>/dev/null || echo "000")
if [ "$FRONTEND_RESPONSE" -eq 200 ]; then
    echo -e "${GREEN}✓ OK${NC}"
else
    echo -e "${RED}✗ FAIL${NC} (HTTP $FRONTEND_RESPONSE)"
    echo -e "${YELLOW}Frontend is not responding. Check logs: docker-compose logs frontend${NC}"
    exit 1
fi

# Test Backend API functionality
echo -n "Testing Backend API endpoints... "
# Test health endpoint returns JSON
HEALTH_JSON=$(curl -s http://localhost:8080/health 2>/dev/null)
if echo "$HEALTH_JSON" | grep -q "status"; then
    echo -e "${GREEN}✓ OK${NC}"
else
    echo -e "${YELLOW}⚠ WARNING${NC}"
    echo -e "${YELLOW}Health endpoint returned unexpected response${NC}"
fi

echo ""
echo -e "${BLUE}======================================${NC}"
echo -e "${GREEN}All systems operational!${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""
echo "You can now access:"
echo "  • Frontend:  http://localhost:3000"
echo "  • Backend:   http://localhost:8080"
echo "  • Postgres:  localhost:5432"
echo ""
echo "Useful commands:"
echo "  • View logs:     docker-compose logs -f"
echo "  • Restart:       docker-compose restart"
echo "  • Stop:          docker-compose down"
echo "  • Full reset:    docker-compose down -v && docker-compose up -d"
echo ""
