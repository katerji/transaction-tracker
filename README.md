# Transaction Tracker

A personal finance tracker built with Go, SQLite, and OpenAI. Paste an SMS bank notification and it extracts the merchant, amount, date, and category automatically. All transactions are displayed on a mobile-friendly dashboard.

## Setup

```bash
cp .env.example .env
```

Edit `.env` with your values:

```
OPENAI_API_KEY=your-key-here
DATABASE_PATH=./transactions.db
PORT=8080
```

## Run

```bash
go run *.go
```

Dashboard at `http://localhost:8080`.

## API

| Endpoint | Method | Description |
|---|---|---|
| `/` | GET | Dashboard |
| `/transaction` | POST | Parse SMS text via OpenAI and save |
| `/transaction/manual` | POST | Add transaction manually |
| `/transaction/:id` | PUT | Update a transaction |
| `/transaction/:id` | DELETE | Delete a transaction |
| `/stats` | GET | Current billing cycle stats |
| `/health` | GET | Health check |

### Example

```bash
curl -X POST http://localhost:8080/transaction \
  -H "Content-Type: application/json" \
  -d '{"text": "Your account was debited AED 50 at Starbucks on 24/01/2026"}'
```

## Deploy

### Docker

```bash
docker compose up -d
```

### Fly.io

```bash
fly secrets set OPENAI_API_KEY=your-key-here
fly deploy
```

Pushes to `main` automatically deploy to Fly.io via GitHub Actions.

## How it works

1. SMS text is sent via POST to `/transaction`
2. OpenAI (gpt-4o-mini) parses the text into structured data
3. Amounts are converted to AED if in another currency
4. Transaction is saved to SQLite with a billing cycle (23rd–22nd)
5. Dashboard shows spending by category for the current cycle

Categories: Food & Dining, Transport, Shopping, Bills & Utilities, Entertainment, Health & Fitness, Travel, Cash Withdrawal, Income/Transfer, Unknown.
