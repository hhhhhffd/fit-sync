.PHONY: help install run-backend run-bot run-tunnel run stop kill-all clean status

# Colors for output
GREEN  := \033[0;32m
YELLOW := \033[0;33m
RED    := \033[0;31m
BLUE   := \033[0;34m
NC     := \033[0m # No Color

help:
	@echo "$(GREEN)🏋️  Fitness App - Commands$(NC)"
	@echo ""
	@echo "$(BLUE)🚀 Quick Start:$(NC)"
	@echo "  make run-backend   - Start backend server"
	@echo "  make run-bot       - Start Telegram bot"
	@echo "  make run-tunnel    - Start Cloudflare tunnel (auto-updates .env!)"
	@echo "  make dev           - Start everything in one command"
	@echo "  make status        - Check what's running"
	@echo "  make stop          - Stop all services"
	@echo ""
	@echo "$(BLUE)📦 Setup:$(NC)"
	@echo "  make install       - Install all dependencies"
	@echo "  make init-db       - Reset database"
	@echo ""
	@echo "$(BLUE)🛠️  Development:$(NC)"
	@echo "  make build         - Build backend binary"
	@echo "  make test          - Run tests"
	@echo "  make clean         - Clean everything"
	@echo "  make kill-all      - Force kill all processes"
	@echo ""
	@echo "$(BLUE)💡 Tip:$(NC) Open 3 terminals and run:"
	@echo "  Terminal 1: make run-backend"
	@echo "  Terminal 2: make run-tunnel"
	@echo "  Terminal 3: make run-bot"

install:
	@echo "$(GREEN)📦 Installing dependencies...$(NC)"
	@echo "$(YELLOW)Installing Go dependencies...$(NC)"
	@cd backend && go mod download && go mod tidy
	@echo "$(YELLOW)Installing Python dependencies...$(NC)"
	@cd telegram-bot && pip3 install -r requirements.txt
	@echo "$(GREEN)✅ Done! Check your .env file for TELEGRAM_BOT_TOKEN$(NC)"

run-backend:
	@echo "$(GREEN)🚀 Starting backend server on port 8080...$(NC)"
	@cd backend && go run cmd/server/main.go

run-bot:
	@echo "$(GREEN)🤖 Starting Telegram bot...$(NC)"
	@if [ -z "$$(grep TELEGRAM_BOT_TOKEN .env | cut -d '=' -f2)" ]; then \
		echo "$(RED)❌ Error: TELEGRAM_BOT_TOKEN not set in .env$(NC)"; \
		exit 1; \
	fi
	@cd telegram-bot && python3 bot.py

run-tunnel:
	@echo "$(GREEN)🌐 Starting Cloudflare tunnel...$(NC)"
	@if ! command -v cloudflared >/dev/null 2>&1; then \
		echo "$(RED)❌ cloudflared not installed!$(NC)"; \
		echo "$(YELLOW)Install: brew install cloudflared$(NC)"; \
		exit 1; \
	fi
	@if ! lsof -ti:8080 >/dev/null 2>&1; then \
		echo "$(RED)❌ Backend not running! Start it first: make run-backend$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)⏳ Starting Cloudflare tunnel...$(NC)"
	@cloudflared tunnel --url http://localhost:8080 > cloudflared.log 2>&1 & \
	CF_PID=$$!; \
	echo "   Waiting for tunnel to create..."; \
	CF_URL=""; \
	for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do \
		sleep 1; \
		CF_URL=$$(grep -o 'https://[a-z0-9-]*\.trycloudflare\.com' cloudflared.log 2>/dev/null | head -1); \
		if [ -n "$$CF_URL" ]; then \
			break; \
		fi; \
		if [ $$i -eq 15 ]; then \
			echo "$(RED)❌ Cloudflare failed to create tunnel in 15 seconds$(NC)"; \
			kill $$CF_PID 2>/dev/null; \
			exit 1; \
		fi; \
	done; \
	if [ -n "$$CF_URL" ]; then \
		echo "$(GREEN)✅ Cloudflare started: $$CF_URL$(NC)"; \
		sed -i.bak "s|WEBAPP_URL=.*|WEBAPP_URL=$$CF_URL/webapp.html|g" .env && rm -f .env.bak; \
		echo "$(GREEN)✅ .env updated automatically!$(NC)"; \
		echo ""; \
		echo "$(BLUE)📱 Tunnel URL: $$CF_URL$(NC)"; \
		echo ""; \
		echo "$(YELLOW)Press Ctrl+C to stop tunnel (PID: $$CF_PID)$(NC)"; \
		wait $$CF_PID; \
	else \
		echo "$(RED)❌ Failed to get Cloudflare URL$(NC)"; \
		kill $$CF_PID 2>/dev/null; \
		exit 1; \
	fi

status:
	@echo "$(BLUE)📊 Service Status:$(NC)"
	@echo ""
	@echo "$(YELLOW)Backend (port 8080):$(NC)"
	@if lsof -ti:8080 >/dev/null 2>&1; then \
		echo "  $(GREEN)✅ Running$(NC) (PID: $$(lsof -ti:8080))"; \
	else \
		echo "  $(RED)❌ Not running$(NC)"; \
	fi
	@echo ""
	@echo "$(YELLOW)Cloudflare Tunnel:$(NC)"
	@if pgrep -f "cloudflared tunnel" >/dev/null 2>&1; then \
		echo "  $(GREEN)✅ Running$(NC)"; \
		grep -o 'https://[a-z0-9-]*\.trycloudflare\.com' cloudflared.log 2>/dev/null | head -1 | sed 's/^/  URL: /' || echo "  $(YELLOW)⚠️  URL not available yet$(NC)"; \
	else \
		echo "  $(RED)❌ Not running$(NC)"; \
	fi
	@echo ""
	@echo "$(YELLOW)Telegram Bot:$(NC)"
	@if pgrep -f "python3 bot.py" >/dev/null 2>&1; then \
		echo "  $(GREEN)✅ Running$(NC) (PID: $$(pgrep -f 'python3 bot.py'))"; \
	else \
		echo "  $(RED)❌ Not running$(NC)"; \
	fi
	@echo ""

stop:
	@echo "$(YELLOW)🛑 Stopping all services...$(NC)"
	@pkill -f "go run cmd/server/main.go" 2>/dev/null || true
	@pkill -f "python3 bot.py" 2>/dev/null || true
	@pkill -f "cloudflared tunnel" 2>/dev/null || true
	@sleep 1
	@echo "$(GREEN)✅ All services stopped$(NC)"

kill-all:
	@echo "$(RED)💀 Force killing all processes...$(NC)"
	@pkill -9 -f "cloudflared" 2>/dev/null || true
	@pkill -9 -f "python3 bot.py" 2>/dev/null || true
	@pkill -9 -f "go run" 2>/dev/null || true
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	@sleep 1
	@if lsof -ti:8080 >/dev/null 2>&1; then \
		echo "$(RED)❌ Port 8080 still in use$(NC)"; \
	else \
		echo "$(GREEN)✅ Port 8080 free$(NC)"; \
	fi

build:
	@echo "$(GREEN)🔨 Building backend...$(NC)"
	@mkdir -p bin
	@cd backend && go build -o ../bin/fitness-server cmd/server/main.go
	@echo "$(GREEN)✅ Binary: bin/fitness-server$(NC)"

test:
	@echo "$(GREEN)🧪 Running tests...$(NC)"
	@cd backend && go test ./... -v

clean:
	@echo "$(YELLOW)🧹 Cleaning...$(NC)"
	@rm -rf bin/
	@rm -f backend/fitness.db
	@rm -f cloudflared.log
	@rm -rf telegram-bot/__pycache__
	@rm -rf telegram-bot/handlers/__pycache__
	@echo "$(GREEN)✅ Clean complete!$(NC)"

init-db:
	@echo "$(YELLOW)🗄️  Resetting database...$(NC)"
	@rm -f backend/fitness.db
	@echo "$(GREEN)✅ Database will be recreated on next start$(NC)"

# Quick dev setup
dev-setup: install init-db
	@echo "$(GREEN)✅ Development environment ready!$(NC)"
	@echo ""
	@echo "$(BLUE)Next steps:$(NC)"
	@echo "1. Update .env with your TELEGRAM_BOT_TOKEN"
	@echo "2. Run: make run-backend (in terminal 1)"
	@echo "3. Run: make run-ngrok (in terminal 2)"
	@echo "4. Update .env with ngrok URL"
	@echo "5. Run: make run-bot (in terminal 3)"

# Show logs
logs-backend:
	@echo "$(BLUE)📋 Backend logs (live):$(NC)"
	@tail -f backend/*.log 2>/dev/null || echo "$(YELLOW)No log files found$(NC)"

logs-tunnel:
	@echo "$(BLUE)📋 Cloudflare logs:$(NC)"
	@tail -f cloudflared.log 2>/dev/null || echo "$(YELLOW)No cloudflared.log found$(NC)"

# Test registration
test-register:
	@echo "$(BLUE)🧪 Testing registration...$(NC)"
	@curl -X POST http://localhost:8080/api/register \
		-H "Content-Type: application/json" \
		-d '{"email":"test@example.com","login":"testuser","password":"password123","name":"Test User"}' \
		| python3 -m json.tool || echo "$(RED)❌ Failed$(NC)"

# Check tunnel URL
tunnel-url:
	@grep -o 'https://[a-z0-9-]*\.trycloudflare\.com' cloudflared.log 2>/dev/null | head -1 || echo "$(RED)Tunnel not running$(NC)"

# Dev mode - start everything in background
dev:
	@echo "$(GREEN)🚀 Starting development environment...$(NC)"
	@echo ""
	@echo "$(YELLOW)1️⃣  Starting backend...$(NC)"
	@cd backend && go run cmd/server/main.go > /dev/null 2>&1 & \
	echo $$! > /tmp/fitness-backend.pid; \
	sleep 2
	@if ! lsof -ti:8080 >/dev/null 2>&1; then \
		echo "$(RED)❌ Backend failed to start$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)   ✅ Backend running on port 8080$(NC)"
	@echo ""
	@echo "$(YELLOW)2️⃣  Starting Cloudflare tunnel...$(NC)"
	@cloudflared tunnel --url http://localhost:8080 > cloudflared.log 2>&1 & \
	echo $$! > /tmp/fitness-tunnel.pid; \
	CF_URL=""; \
	for i in 1 2 3 4 5 6 7 8 9 10; do \
		sleep 1; \
		CF_URL=$$(grep -o 'https://[a-z0-9-]*\.trycloudflare\.com' cloudflared.log 2>/dev/null | head -1); \
		if [ -n "$$CF_URL" ]; then \
			break; \
		fi; \
	done; \
	if [ -n "$$CF_URL" ]; then \
		echo "$(GREEN)   ✅ Tunnel running: $$CF_URL$(NC)"; \
		sed -i.bak "s|WEBAPP_URL=.*|WEBAPP_URL=$$CF_URL/webapp.html|g" .env && rm -f .env.bak; \
		echo "$(GREEN)   ✅ .env updated$(NC)"; \
	else \
		echo "$(RED)   ❌ Tunnel failed$(NC)"; \
	fi
	@echo ""
	@echo "$(YELLOW)3️⃣  Starting Telegram bot...$(NC)"
	@cd telegram-bot && python3 bot.py > /dev/null 2>&1 & \
	echo $$! > /tmp/fitness-bot.pid; \
	sleep 2
	@if pgrep -f "python3 bot.py" >/dev/null 2>&1; then \
		echo "$(GREEN)   ✅ Bot running$(NC)"; \
	else \
		echo "$(RED)   ❌ Bot failed$(NC)"; \
	fi
	@echo ""
	@echo "$(GREEN)✨ All services started!$(NC)"
	@echo ""
	@make status
	@echo ""
	@echo "$(YELLOW)💡 Tip: Use 'make stop' to stop all services$(NC)"

# Stop dev mode
stop-dev:
	@echo "$(YELLOW)🛑 Stopping development environment...$(NC)"
	@if [ -f /tmp/fitness-backend.pid ]; then kill $$(cat /tmp/fitness-backend.pid) 2>/dev/null && rm /tmp/fitness-backend.pid; fi
	@if [ -f /tmp/fitness-tunnel.pid ]; then kill $$(cat /tmp/fitness-tunnel.pid) 2>/dev/null && rm /tmp/fitness-tunnel.pid; fi
	@if [ -f /tmp/fitness-bot.pid ]; then kill $$(cat /tmp/fitness-bot.pid) 2>/dev/null && rm /tmp/fitness-bot.pid; fi
	@pkill -f "go run cmd/server/main.go" 2>/dev/null || true
	@pkill -f "python3 bot.py" 2>/dev/null || true
	@pkill -f "cloudflared tunnel" 2>/dev/null || true
	@sleep 1
	@echo "$(GREEN)✅ All services stopped$(NC)"
