#!/bin/bash

# gorrson - Expert RSS Engine (Installation Script)
# Optimized for Linux 

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}===> Gorrson: Starting Installation...${NC}"

# 1. Check for Go
if ! command -v go &> /dev/null
then
    echo -e "${RED}[ERROR] Go is not installed. Please install Go (1.20+) and try again.${NC}"
    exit 1
fi

# 2. Compile the binary
echo -e "${BLUE}===> Building frontend assets...${NC}"
(cd internal/api/web && npm install && npm run build)
if [ $? -ne 0 ]; then
    echo -e "${RED}[ERROR] Frontend build failed.${NC}"
    exit 1
fi

echo -e "${BLUE}===> Building binary...${NC}"
go build -v -o rss-cli main.go

if [ $? -eq 0 ]; then
    echo -e "${GREEN}[SUCCESS] Binary 'rss-cli' built successfully.${NC}"
else
    echo -e "${RED}[ERROR] Build failed.${NC}"
    exit 1
fi

# 3. Create data directories
echo -e "${BLUE}===> Setting up directories...${NC}"
mkdir -p data/cache/images
echo -e "${GREEN}[SUCCESS] Data and cache directories created.${NC}"

# 4. Final instructions
echo -e "\n${GREEN}Gorrson is now installed!${NC}"
echo -e "----------------------------------------"
echo -e "To start the server, run:"
echo -e "  ${BLUE}./rss-cli serve${NC}"
echo -e "\nTo register your first user:"
echo -e "  ${BLUE}./rss-cli register --user <name> --pass <password>${NC}"
echo -e "\nTo add your first feed:"
echo -e "  ${BLUE}./rss-cli add --url <feed-url> --user <name>${NC}"
echo -e "----------------------------------------"
