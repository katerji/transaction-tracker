# CLAUDE.md

## Quick start
```bash
cp .env.example .env   # fill in OPENAI_API_KEY
go run *.go             # starts on :8080
```

## Build & deploy
- `docker compose up -d` for local Docker
- `fly deploy` for Fly.io (config in fly.toml)
- Pushes to `main` auto-deploy to Fly.io via GitHub Actions (`.github/workflows/deploy.yml`)
- CGO is required (go-sqlite3) — the Dockerfile handles this

## Architecture
- Go stdlib HTTP server (no router library)
- SQLite via go-sqlite3 (no ORM, raw SQL)
- OpenAI gpt-4o-mini for parsing SMS text into transactions
- Dashboard is a single HTML file embedded via `//go:embed`
- Frontend is vanilla JS + CSS, no frameworks

## Conventions
- All monetary amounts are in AED — currency conversion happens in the OpenAI prompt
- Billing cycle runs from the 23rd to the 22nd of each month
- Income/Transfer category is excluded from spending totals
- Transactions are deduplicated by (description, amount, transaction_date)
- Manual entries always have confidence=100
