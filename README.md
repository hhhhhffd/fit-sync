<div align="center">

<h1>FitSync</h1>
<hr>

**Fitness challenges in one place: Telegram bot, Telegram WebApp, and a Go REST API on SQLite**

![Go](https://img.shields.io/badge/GO-1.24-00ADD8?style=flat-square&logo=go&logoColor=white)
![Python](https://img.shields.io/badge/PYTHON-3.10+-3776AB?style=flat-square&logo=python&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLITE-file-003B57?style=flat-square&logo=sqlite&logoColor=white)
![License](https://img.shields.io/badge/LICENSE-MIT-green?style=flat-square)

[Stack](#technical-stack) · [Quick start](#quick-start) · [License](#license)

</div>

> [!NOTE]
> Archived — this project is no longer maintained.

> [!WARNING]
> Known authentication weaknesses — not intended for production use.

## What it is

FitSync is a social platform for fitness competitions. A user creates a challenge, invites rivals by invite code or login, and logs progress. The system declares the winner automatically once the goal is reached and updates wins, streaks and achievements.

Two clients share one state: a Telegram bot (aiogram) and a browser SPA that also runs as a Telegram WebApp. Both talk to the same Go REST API backed by SQLite.

## Highlights

- 19 REST endpoints (3 public, 16 JWT-protected) in a Go service built on `gorilla/mux`.
- Challenge types `accumulative` and `consistency`, with an auto-win check on every progress update.
- 10 seeded achievements based on wins and streaks, plus a leaderboard.
- Bot push notifications to rivals and a daily reminder scheduler.
- `make run-tunnel` starts a Cloudflare Tunnel and writes the public URL into `.env`, as the Telegram WebApp requires HTTPS.

## Technical stack

| Layer | Technology |
|-------|------------|
| API | Go 1.24, `gorilla/mux`, `golang-jwt/jwt` v5 |
| Data | SQLite (`mattn/go-sqlite3`, file `fitness.db`), SQL migration |
| Bot | Python 3.10+, aiogram 3.3, aiohttp |
| Frontend | HTML, CSS, vanilla JS, Telegram WebApp SDK |
| Infra | Makefile, Cloudflare Tunnel (`cloudflared`) |

## Quick start

Requirements: Go, Python 3.10+, `cloudflared`, and a bot token from [@BotFather](https://t.me/BotFather).

```bash
cp .env.example .env   # set TELEGRAM_BOT_TOKEN and JWT_SECRET
make install
make run-backend       # terminal 1: API on :8080, the DB is created on start
make run-tunnel        # terminal 2: public HTTPS URL, updates WEBAPP_URL in .env
make run-bot           # terminal 3: Telegram bot
```

`make dev` starts all three in the background; `make stop` stops them.

| Variable | Default | Purpose |
|----------|---------|---------|
| `PORT` | `8080` | API port |
| `DB_PATH` | `fitness.db` | SQLite file |
| `API_URL` | `http://localhost:8080/api` | API address used by the bot |
| `TELEGRAM_BOT_TOKEN` | placeholder | Bot token |
| `JWT_SECRET` | placeholder | JWT signing secret |
| `WEBAPP_URL` | set by `make run-tunnel` | WebApp address opened from the bot |

## Project structure

```text
backend/       Go API (cmd/server, internal/{api,services,database}, migrations)
telegram-bot/  aiogram bot: handlers, notifications, message texts
frontend/      SPA pages and the Telegram WebApp entry (webapp.html)
Makefile       run, tunnel and status targets
```

## License

MIT, see [LICENSE](LICENSE).
