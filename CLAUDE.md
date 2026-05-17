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
- Categories engine — DB-backed (`categories` table), user-manageable CRUD, seeded with 11 UAE defaults on first boot; `exclude_from_totals` flag drives spending exclusions
- Merchant rules engine overrides OpenAI categories for known merchants — DB-backed (`merchant_rules` table), priority-ordered, case-insensitive substring matching
- OpenAI prompt is built dynamically from the `categories` table on every parse request
- Dashboard served from `static/` (embedded via `//go:embed`)
- Frontend is vanilla JS + Alpine.js + Tailwind CSS

## Conventions
- All monetary amounts are in AED — currency conversion happens in the OpenAI prompt
- Billing cycle runs from the 23rd to the 22nd of each month
- Categories with `exclude_from_totals = 1` are excluded from spending totals (default: `Income/Transfer`)
- Transactions are deduplicated by (description, amount, transaction_date)
- Manual entries always have confidence=100
- Transaction `source` tracks categorization origin: `"openai"` (AI-assigned), `"rule"` (matched a merchant rule), `"manual"` (user edited) — retroactive rule application skips `"manual"` transactions

## Categories
Categories are user-managed and stored in the `categories` DB table. 11 defaults are seeded on first boot:

| Category | Emoji | Excluded from totals |
|---|---|---|
| Groceries | 🛒 | No |
| Dining Out | 🍔 | No |
| Transport | 🚗 | No |
| Shopping | 🛍️ | No |
| Subscriptions | 📱 | No |
| Bills & Utilities | 💳 | No |
| Health | 💊 | No |
| Travel | ✈️ | No |
| Entertainment | 🎬 | No |
| Cash Withdrawal | 💵 | No |
| Income/Transfer | 💰 | Yes |

Add, rename, or delete categories from the Categories tab (Manage mode). Renaming cascades to all transactions and merchant rules. Deleting is blocked if any transactions reference the category.
