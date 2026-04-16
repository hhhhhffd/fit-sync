#!/bin/bash

# Colors
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Starting Fitness App with Cloudflare Tunnel...${NC}\n"

# Check if cloudflared is installed
if ! command -v cloudflared &> /dev/null; then
    echo -e "${RED}❌ cloudflared is not installed!${NC}"
    echo -e "${YELLOW}Install it: brew install cloudflared (macOS)${NC}"
    echo -e "${YELLOW}Or download from: https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation${NC}"
    exit 1
fi

# Check if backend is running
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null ; then
    echo -e "${YELLOW}⚠️  Port 8080 is already in use. Backend might be running.${NC}"
else
    echo -e "${YELLOW}⚠️  Backend is not running on port 8080${NC}"
    echo -e "${YELLOW}Please start backend first: make run-backend${NC}"
    exit 1
fi

# Start cloudflared
echo -e "${GREEN}Starting Cloudflare tunnel...${NC}"
cloudflared tunnel --url http://localhost:8080 > cloudflared.log 2>&1 &
CF_PID=$!

# Wait for cloudflared to start and get URL
echo -e "${YELLOW}Waiting for tunnel to be ready...${NC}"
sleep 5

# Get cloudflared URL from log
CF_URL=""
for i in {1..10}; do
    CF_URL=$(grep -o 'https://[a-z0-9-]*\.trycloudflare\.com' cloudflared.log | head -1)
    if [ -n "$CF_URL" ]; then
        break
    fi
    sleep 2
done

if [ -z "$CF_URL" ]; then
    echo -e "${RED}❌ Failed to get Cloudflare URL${NC}"
    echo -e "${YELLOW}Check cloudflared.log for details${NC}"
    cat cloudflared.log
    kill $CF_PID 2>/dev/null
    exit 1
fi

echo -e "${GREEN}✅ Cloudflare tunnel created!${NC}"
echo -e "${GREEN}URL: ${CF_URL}${NC}\n"

# Update .env file
WEBAPP_URL="${CF_URL}/webapp.html"
echo -e "${GREEN}Updating .env file...${NC}"

if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    sed -i '' "s|WEBAPP_URL=.*|WEBAPP_URL=${WEBAPP_URL}|g" .env
else
    # Linux
    sed -i "s|WEBAPP_URL=.*|WEBAPP_URL=${WEBAPP_URL}|g" .env
fi

echo -e "${GREEN}✅ .env updated with: ${WEBAPP_URL}${NC}\n"

echo -e "${YELLOW}📱 Now you need to:${NC}"
echo -e "${YELLOW}1. Restart the Telegram bot: make run-bot${NC}"
echo -e "${YELLOW}2. Open your bot in Telegram${NC}"
echo -e "${YELLOW}3. Send /webapp to get the Web App button${NC}\n"

echo -e "${GREEN}🌐 Web App URL: ${WEBAPP_URL}${NC}\n"

echo -e "${YELLOW}Press Ctrl+C to stop tunnel${NC}"

# Wait for user to stop
trap "echo -e '\n${RED}Stopping Cloudflare tunnel...${NC}'; kill $CF_PID 2>/dev/null; exit" INT TERM

wait $CF_PID
