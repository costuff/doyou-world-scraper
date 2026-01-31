#!/bin/bash
set -e

echo "=================================================="
echo "  DoYou World Radio Scraper - Raspberry Pi Setup"
echo "=================================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if running on ARM
ARCH=$(uname -m)
if [[ "$ARCH" != "aarch64" && "$ARCH" != "armv7l" ]]; then
    echo -e "${YELLOW}Warning: Not running on ARM architecture (detected: $ARCH)${NC}"
    echo "This script is designed for Raspberry Pi. Continue anyway? (y/n)"
    read -r response
    if [[ ! "$response" =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    echo "Install Docker first: curl -fsSL https://get.docker.com | sh"
    exit 1
fi

# Check if Docker Compose is installed
if ! docker compose version &> /dev/null; then
    echo -e "${RED}Error: Docker Compose is not installed${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Docker and Docker Compose are installed${NC}"
echo ""

# Prompt for configuration
echo "Configuration:"
echo "-------------"

read -p "Enter your GitHub username (default: costuff): " GITHUB_USER
GITHUB_USER=${GITHUB_USER:-costuff}

read -p "Enter API port (default: 3000): " API_PORT
API_PORT=${API_PORT:-3000}

read -p "Enter Frontend port (default: 8080): " FRONTEND_PORT
FRONTEND_PORT=${FRONTEND_PORT:-8080}

read -p "Enter NAS mount path (default: /mnt/music): " NAS_PATH
NAS_PATH=${NAS_PATH:-/mnt/music}

read -p "Enter timezone (default: Europe/Athens): " TZ
TZ=${TZ:-Europe/Athens}

# Create .env file
echo ""
echo "Creating .env file..."
cat > .env << EOF
# GitHub Container Registry
GITHUB_REPOSITORY_OWNER=$GITHUB_USER

# Port configuration
API_PORT=$API_PORT
FRONTEND_PORT=$FRONTEND_PORT

# Storage paths
NAS_PATH=$NAS_PATH

# Timezone
TZ=$TZ
EOF

echo -e "${GREEN}✓ .env file created${NC}"
echo ""

# Create directories
echo "Creating directories..."
mkdir -p csv downloads
echo -e "${GREEN}✓ Directories created${NC}"
echo ""

# Login to GitHub Container Registry
echo "Logging in to GitHub Container Registry..."
echo "You'll need a GitHub Personal Access Token with 'read:packages' permission"
echo "Create one at: https://github.com/settings/tokens/new"
echo ""
read -p "Enter your GitHub username: " GH_USERNAME
read -sp "Enter your GitHub Personal Access Token: " GH_TOKEN
echo ""

echo "$GH_TOKEN" | docker login ghcr.io -u "$GH_USERNAME" --password-stdin

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Successfully logged in to GitHub Container Registry${NC}"
else
    echo -e "${RED}✗ Failed to login. Please check your credentials.${NC}"
    exit 1
fi
echo ""

# Pull images
echo "Pulling Docker images..."
docker compose -f docker-compose.prod.yml pull

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Images pulled successfully${NC}"
else
    echo -e "${RED}✗ Failed to pull images${NC}"
    exit 1
fi
echo ""

# Start containers
echo "Starting containers..."
docker compose -f docker-compose.prod.yml up -d

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Containers started successfully${NC}"
else
    echo -e "${RED}✗ Failed to start containers${NC}"
    exit 1
fi
echo ""

# Check container status
echo "Container status:"
docker compose -f docker-compose.prod.yml ps
echo ""

# Setup Watchtower for auto-updates
echo "Setting up Watchtower for automatic updates..."
docker run -d \
  --name watchtower \
  --restart unless-stopped \
  -v /var/run/docker.sock:/var/run/docker.sock \
  containrrr/watchtower \
  --interval 300 \
  --cleanup \
  doyou-api doyou-frontend

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Watchtower configured (checks every 5 minutes)${NC}"
else
    echo -e "${YELLOW}⚠ Failed to setup Watchtower (non-critical)${NC}"
fi
echo ""

# Get local IP
LOCAL_IP=$(hostname -I | awk '{print $1}')

echo "=================================================="
echo -e "${GREEN}Deployment Complete!${NC}"
echo "=================================================="
echo ""
echo "Access your application at:"
echo "  Frontend: http://$LOCAL_IP:$FRONTEND_PORT"
echo "  API:      http://$LOCAL_IP:$API_PORT/api/v1/health"
echo ""
echo "Local access:"
echo "  Frontend: http://localhost:$FRONTEND_PORT"
echo "  API:      http://localhost:$API_PORT"
echo ""
echo "Useful commands:"
echo "  View logs:    docker compose -f docker-compose.prod.yml logs -f"
echo "  Stop:         docker compose -f docker-compose.prod.yml down"
echo "  Restart:      docker compose -f docker-compose.prod.yml restart"
echo "  Update:       docker compose -f docker-compose.prod.yml pull && docker compose -f docker-compose.prod.yml up -d"
echo ""
echo "Watchtower will automatically check for updates every 5 minutes."
echo "=================================================="
