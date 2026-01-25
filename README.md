# Transaction Tracker - Go Edition

A lightweight, self-hosted transaction tracking system built with Go (stdlib only), OpenAI, and Notion. No n8n, no external dependencies - just a single binary.

## Features

- 💰 **Automatic transaction parsing** - Send SMS text, AI extracts details
- 💱 **Currency conversion** - Converts all amounts to AED
- 📊 **Smart categorization** - AI assigns categories with confidence scores
- 📅 **Billing cycle tracking** - 23rd to 22nd of each month
- 📱 **Phone-friendly** - Works with HTTP Shortcuts app
- 🚀 **Single binary** - No runtime dependencies
- ☁️ **Free cloud hosting** - Deploy to Fly.io
- 🐳 **Docker support** - Containerized deployment

## Quick Start

### Prerequisites

- Go 1.22+ (for development)
- OpenAI API key
- Notion account with integration set up
- (Optional) Docker for containerized deployment

### 1. Set Up Notion

Follow the Notion setup guide to:
1. Create a Notion database with required properties
2. Create a Notion integration
3. Share the database with your integration
4. Get your API key and database ID

See `NOTION_SETUP_GUIDE.md` for detailed instructions.

### 2. Clone and Configure

```bash
cd transaction-tracker

# Copy environment template
cp .env.example .env

# Edit .env with your credentials
nano .env
```

Required environment variables:
```bash
OPENAI_API_KEY=sk-proj-your-key-here
NOTION_API_KEY=secret_your-key-here
NOTION_DATABASE_ID=your-database-id-here
PORT=8080  # optional, defaults to 8080
```

### 3. Run Locally

**Option A: Run directly**
```bash
go run *.go
```

**Option B: Build and run**
```bash
go build -o transaction-tracker
./transaction-tracker
```

**Option C: Docker**
```bash
docker build -t transaction-tracker .
docker run -p 8080:8080 --env-file .env transaction-tracker
```

The server will start on `http://localhost:8080`

### 4. Test the API

**Log a transaction:**
```bash
curl -X POST http://localhost:8080/transaction \
  -H "Content-Type: application/json" \
  -d '{"text": "AED 50 at Starbucks Dubai Mall"}'
```

**Get stats:**
```bash
curl http://localhost:8080/stats
```

**Health check:**
```bash
curl http://localhost:8080/health
```

## API Reference

### POST /transaction

Log one or more transactions from SMS text.

**Request:**
```json
{
  "text": "Your account was debited AED 50 at Starbucks"
}
```

**Response:**
```json
{
  "success": true,
  "message": "✅ Added 1 transaction!\n\n1. Starbucks\n   💰 Amount: 50.00 AED\n   📁 Category: 🍔 Food & Dining (95% confidence)\n   📅 Cycle: Jan 2026\n\n━━━━━━━━━━━━━━━\n💵 Total: 50.00 AED",
  "count": 1,
  "total": 50.00,
  "transactions": [...]
}
```

### GET /stats

Get spending statistics for the current billing cycle.

**Response:**
```json
{
  "success": true,
  "message": "📊 Billing Cycle: Jan 2026 (23rd - 22nd)\n...",
  "cycle": "Jan 2026",
  "total": 150.50,
  "count": 5,
  "categories": [...],
  "lastTransaction": {...}
}
```

### GET /health

Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "time": "2026-01-25T10:30:00Z"
}
```

## Cloud Deployment

### Deploy to Fly.io (Free Tier)

1. **Install Fly CLI:**
```bash
# macOS
brew install flyctl

# Linux/WSL
curl -L https://fly.io/install.sh | sh

# Windows
powershell -Command "iwr https://fly.io/install.ps1 -useb | iex"
```

2. **Login and create app:**
```bash
fly auth login
fly launch
```

3. **Set secrets:**
```bash
fly secrets set OPENAI_API_KEY=sk-proj-your-key-here
fly secrets set NOTION_API_KEY=secret_your-key-here
fly secrets set NOTION_DATABASE_ID=your-database-id-here
```

4. **Deploy:**
```bash
fly deploy
```

Your app will be live at `https://your-app-name.fly.dev`

**Fly.io free tier includes:**
- 3 shared-cpu-1x VMs (256MB RAM each)
- 160GB outbound data transfer
- Perfect for this use case!

## Phone Setup

Use the HTTP Shortcuts app (Android) or Shortcuts app (iOS) to log transactions from your phone.

**For local deployment:**
- Use your computer's local IP: `http://192.168.1.100:8080/transaction`

**For cloud deployment:**
- Use your app URL: `https://your-app.fly.dev/transaction`

See `HTTP_SHORTCUTS_SETUP_GUIDE.md` for detailed phone setup instructions.

## Project Structure

```
transaction-tracker/
├── main.go           # HTTP server, routing, main logic
├── openai.go         # OpenAI API client
├── notion.go         # Notion API client
├── go.mod            # Go module definition
├── Dockerfile        # Docker container definition
├── .env.example      # Environment variable template
└── README.md         # This file
```

## How It Works

```
1. Receive SMS text via HTTP POST
     ↓
2. Call OpenAI to parse transaction details
   - Extracts date, amount, merchant
   - Categorizes transaction
   - Converts currency to AED
     ↓
3. Calculate billing cycle (23rd cutoff)
     ↓
4. Save to Notion database via API
     ↓
5. Return confirmation to user
```

## Configuration

All configuration is done via environment variables:

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `OPENAI_API_KEY` | OpenAI API key for parsing | Yes | - |
| `NOTION_API_KEY` | Notion integration secret | Yes | - |
| `NOTION_DATABASE_ID` | Notion database ID | Yes | - |
| `PORT` | Server port | No | `8080` |

## Development

**Run tests:**
```bash
go test ./...
```

**Build for production:**
```bash
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o transaction-tracker .
```

**Cross-compile for different platforms:**
```bash
# macOS
GOOS=darwin GOARCH=amd64 go build -o transaction-tracker-mac

# Linux
GOOS=linux GOARCH=amd64 go build -o transaction-tracker-linux

# Windows
GOOS=windows GOARCH=amd64 go build -o transaction-tracker.exe
```

## Supported Transaction Categories

- 🍔 Food & Dining
- 🚗 Transport
- 🛍️ Shopping
- 💳 Bills & Utilities
- 🎬 Entertainment
- 💪 Health & Fitness
- ✈️ Travel
- 💵 Cash Withdrawal
- 💰 Income/Transfer
- ❓ Unknown (low confidence)

## Currency Conversion

The system automatically converts the following currencies to AED:

| Currency | Rate (approx) |
|----------|---------------|
| USD | × 3.67 |
| EUR | × 4.00 |
| GBP | × 4.70 |
| SAR | × 0.98 |

Other currencies are converted using approximate rates by OpenAI.

## Troubleshooting

**Issue: "OPENAI_API_KEY environment variable is required"**
- Solution: Create `.env` file with your API keys (see `.env.example`)

**Issue: Notion API returns 401 Unauthorized**
- Solution: Verify your Notion API key is correct
- Ensure database is shared with your integration

**Issue: Notion API returns 404 Not Found**
- Solution: Check your `NOTION_DATABASE_ID` is correct
- Verify database exists and is accessible

**Issue: OpenAI parsing fails**
- Solution: Check your OpenAI API key
- Verify you have credits in your OpenAI account
- Check API key has appropriate permissions

**Issue: Cloud deployment sleeps/goes inactive**
- Fly.io free tier apps may sleep after inactivity
- First request after sleep may be slower
- Consider upgrading to paid tier for always-on

## Cost Breakdown

**Free Cloud Hosting:**
- Fly.io: Free tier (3 VMs, 160GB transfer)

**API Costs:**
- OpenAI (GPT-4o-mini): ~$0.0001 per transaction
- Notion API: Free (unlimited)

**Estimated monthly cost:**
- 100 transactions/month: ~$0.01
- Essentially free!

## Security

**Best Practices:**
- ✅ Never commit `.env` file to Git
- ✅ Use secrets management in cloud platforms
- ✅ Rotate API keys periodically
- ✅ Monitor API usage for unusual activity
- ✅ Use HTTPS in production (automatic on cloud platforms)

**What data is sent where:**
- OpenAI: SMS text for parsing (via HTTPS)
- Notion: Parsed transaction data (via HTTPS API)
- No data is logged or stored by the application

## Advantages Over n8n

✅ **Simpler** - Single Go binary, no workflow editor needed
✅ **Lighter** - ~10MB binary vs full n8n stack
✅ **Faster** - Direct API calls, no workflow engine overhead
✅ **Free cloud hosting** - Easily deploy to Fly.io
✅ **Version control friendly** - Just code, no JSON workflows
✅ **Developer-friendly** - Easy to modify and extend
✅ **No dependencies** - Uses only Go stdlib
✅ **Better for CI/CD** - Standard Go project structure

## Extending the Application

**Add new features:**

1. **Email notifications:**
```go
// Add email client in main.go
func sendEmailNotification(tx Transaction) error {
    // Implement email sending
}
```

2. **Budget alerts:**
```go
// Add budget checking in notion.go
func (c *NotionClient) CheckBudget(category string) (bool, error) {
    // Query totals and compare to budget
}
```

3. **Export to CSV:**
```go
// Add export endpoint
func exportHandler(w http.ResponseWriter, r *http.Request) {
    // Query Notion and generate CSV
}
```

## License

MIT License - feel free to use and modify!

## Support

**Having issues?**
1. Check the logs for error messages
2. Verify environment variables are set correctly
3. Test API endpoints with curl
4. Check Notion database is properly configured

**Resources:**
- OpenAI API Docs: https://platform.openai.com/docs
- Notion API Docs: https://developers.notion.com
- Go Documentation: https://go.dev/doc/
- Fly.io Docs: https://fly.io/docs/

---

**Built with ❤️ using Go stdlib, OpenAI, and Notion**
