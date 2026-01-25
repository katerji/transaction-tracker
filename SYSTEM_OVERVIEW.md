# Transaction Tracker - Complete System Overview

## Table of Contents
1. [System Purpose & Overview](#system-purpose--overview)
2. [Architecture Overview](#architecture-overview)
3. [Technology Stack](#technology-stack)
4. [Project Structure](#project-structure)
5. [Data Flow](#data-flow)
6. [Database Schema](#database-schema)
7. [API Endp/cosoints](#api-endpoints)
8. [Frontend Architecture](#frontend-architecture)
9. [Key Features](#key-features)
10. [Configuration](#configuration)
11. [Deployment](#deployment)
12. [Code Patterns](#code-patterns)
13. [Important Functions](#important-functions)
14. [Recent Changes](#recent-changes)
15. [Development Workflow](#development-workflow)

---

## System Purpose & Overview

### What It Does
Transaction Tracker is a lightweight, self-hosted expense tracking system that automatically parses transaction SMS messages using AI and stores them in a SQLite database. It provides a real-time dashboard for visualizing spending patterns organized by billing cycles (23rd of each month).

### Why It Exists
- **Problem**: Manual expense tracking is tedious and error-prone
- **Solution**: Copy SMS → AI parses details → Automatically categorized & stored
- **Value**: Instant insights into spending patterns with zero manual data entry

### Core Capabilities
1. AI-powered transaction parsing from natural language text
2. Automatic currency conversion to AED
3. Smart categorization with confidence scoring
4. Billing cycle-based analytics (23rd to 22nd of each month)
5. Real-time web dashboard with edit/delete capabilities
6. Zero external dependencies (no database server required)

---

## Architecture Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLIENT LAYER                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │ Web Browser  │  │ HTTP Request │  │Phone Shortcut│         │
│  │  (Dashboard) │  │   (cURL)     │  │   (Mobile)   │         │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘         │
│         │                  │                  │                  │
│         └──────────────────┼──────────────────┘                  │
│                            │                                     │
└────────────────────────────┼─────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                       HTTP SERVER LAYER                         │
│                    (Go net/http stdlib)                         │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐    │
│  │  Route Handlers (main.go)                              │    │
│  │  • GET  /              → Dashboard UI                  │    │
│  │  • POST /transaction    → Log new transaction          │    │
│  │  • PUT  /transaction/:id → Update transaction          │    │
│  │  • DELETE /transaction/:id → Delete transaction        │    │
│  │  • GET  /stats          → Get billing cycle stats      │    │
│  │  • GET  /health         → Health check                 │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                  │
└────────────────────────────────────────────────────────────────┘
                             │
                    ┌────────┴────────┐
                    │                 │
                    ▼                 ▼
┌──────────────────────────┐  ┌─────────────────────────┐
│   OPENAI CLIENT LAYER    │  │  DATABASE CLIENT LAYER  │
│     (openai.go)          │  │     (database.go)       │
│                          │  │                         │
│ • ParseTransactions()    │  │ • SaveTransaction()     │
│   - Sends to GPT-4o-mini │  │ • GetStats()            │
│   - Extracts structured  │  │ • UpdateTransaction()   │
│     data from text       │  │ • DeleteTransaction()   │
│   - Returns Transaction  │  │ • Migration management  │
│     array                │  │                         │
└──────────┬───────────────┘  └────────┬────────────────┘
           │                           │
           ▼                           ▼
┌──────────────────────┐    ┌────────────────────────┐
│  OpenAI API          │    │  SQLite Database       │
│  (External Service)  │    │  (transactions.db)     │
│                      │    │                        │
│  • GPT-4o-mini model │    │  • Single file DB      │
│  • JSON responses    │    │  • No server required  │
│  • HTTPS API         │    │  • ACID compliance     │
└──────────────────────┘    └────────────────────────┘
```

### Component Interaction Flow

```
User Input (SMS Text)
    │
    ├──→ HTTP POST /transaction
    │       │
    │       ├──→ Validate request
    │       │
    │       ├──→ OpenAIClient.ParseTransactions()
    │       │       │
    │       │       ├──→ Build system prompt
    │       │       ├──→ Call OpenAI API
    │       │       └──→ Return []Transaction
    │       │
    │       ├──→ For each transaction:
    │       │       ├──→ enrichTransaction()
    │       │       │       ├──→ Add timestamp
    │       │       │       └──→ Calculate billing cycle
    │       │       │
    │       │       └──→ DatabaseClient.SaveTransaction()
    │       │               └──→ INSERT INTO transactions
    │       │
    │       └──→ Return success response
    │
    └──→ Dashboard refreshes automatically
            │
            └──→ GET /stats
                    │
                    └──→ DatabaseClient.GetStats()
                            ├──→ Query current cycle totals
                            ├──→ Group by category
                            ├──→ Fetch transactions per category
                            └──→ Return StatsResponse
```

---

## Technology Stack

### Backend
- **Language**: Go 1.22+
- **Standard Library**:
  - `net/http` - HTTP server and routing
  - `encoding/json` - JSON marshaling/unmarshaling
  - `database/sql` - Database interface
  - `log` - Structured logging
  - `time` - Date/time calculations
  - `fmt` - String formatting

### Database
- **SQLite 3** - Embedded SQL database
  - Driver: `github.com/mattn/go-sqlite3` (only external dependency)
  - Single-file storage
  - No server process required
  - CGO-enabled for SQLite C bindings

### AI/ML
- **OpenAI GPT-4o-mini**
  - Transaction parsing and categorization
  - Natural language understanding
  - Currency conversion logic
  - Confidence scoring

### Frontend
- **Pure HTML/CSS/JavaScript** - No frameworks
  - Embedded in Go binary as string literal
  - Vanilla JavaScript (ES6+)
  - CSS Grid for responsive layout
  - Fetch API for AJAX calls

### Deployment
- **Docker** - Containerization
  - Multi-stage builds (builder + runtime)
  - Alpine Linux base (minimal size)
  - Volume mounting for database persistence

- **Fly.io** - Cloud hosting
  - Free tier compatible
  - Auto-scaling with machine sleep
  - Persistent volume storage
  - HTTPS automatic

### Development Tools
- **Go modules** - Dependency management
- **Docker Compose** - Local development environment
- **Environment variables** - Configuration management

---

## Project Structure

```
transaction-tracker/
│
├── main.go              # HTTP server, routing, handlers, main application logic
│   ├── Config struct
│   ├── Request/Response types
│   ├── loadConfig()
│   ├── main() - entry point
│   ├── healthHandler()
│   ├── transactionHandler()
│   ├── statsHandler()
│   ├── dashboardHandler()
│   ├── transactionDetailHandler()
│   ├── updateTransactionHandler()
│   ├── deleteTransactionHandler()
│   ├── enrichTransaction()
│   ├── calculateBillingCycle()
│   ├── getCategoryEmoji()
│   └── pluralize()
│
├── database.go          # SQLite database client and operations
│   ├── DatabaseClient struct
│   ├── NewDatabaseClient()
│   ├── runMigrations()
│   ├── SaveTransaction()
│   ├── GetStats()
│   ├── UpdateTransaction()
│   ├── DeleteTransaction()
│   └── Close()
│
├── openai.go            # OpenAI API client for transaction parsing
│   ├── OpenAIClient struct
│   ├── Transaction struct (shared data model)
│   ├── openAIRequest/Response types
│   ├── systemPrompt constant
│   ├── NewOpenAIClient()
│   └── ParseTransactions()
│
├── dashboard.go         # Embedded HTML/CSS/JS for web UI
│   └── dashboardHTML constant (1087 lines of embedded frontend)
│
├── go.mod               # Go module definition
│   └── Dependency: github.com/mattn/go-sqlite3
│
├── go.sum               # Dependency checksums
│
├── Dockerfile           # Multi-stage Docker build configuration
│   ├── Builder stage (Go 1.22 + build deps)
│   └── Runtime stage (Alpine + SQLite libs)
│
├── docker-compose.yml   # Docker Compose for local dev
│   ├── Service: app
│   └── Volume: sqlite_data (persistence)
│
├── fly.toml             # Fly.io deployment configuration
│   ├── App name, region settings
│   ├── HTTP service config
│   ├── Volume mounts
│   └── VM specifications
│
├── .env.example         # Environment variable template
│   ├── OPENAI_API_KEY
│   ├── DATABASE_PATH
│   └── PORT
│
└── README.md            # User documentation
```

### File Breakdown

#### main.go (390 lines)
**Purpose**: Core application - HTTP server, routing, business logic

**Key Components**:
- **Config management**: Environment variable loading with defaults
- **HTTP handlers**: 6 endpoints handling all API operations
- **Business logic**: Transaction enrichment, billing cycle calculation
- **Helper functions**: Category emoji mapping, string formatting
- **Logging**: Comprehensive request/response logging with microsecond precision

**Dependencies**: database.go, openai.go, dashboard.go

---

#### database.go (308 lines)
**Purpose**: SQLite database abstraction layer

**Key Components**:
- **Connection management**: Pool configuration optimized for SQLite (1 writer)
- **Migrations**: Automatic schema creation with indexes
- **CRUD operations**: Save, Update, Delete transactions
- **Analytics**: Complex stats queries with grouping and aggregation
- **Error handling**: Proper error wrapping with context

**Special Features**:
- UNIQUE constraint on (description, amount, transaction_date) prevents duplicates
- Indexes on billing_cycle, transaction_date, category for query performance
- Proper transaction lifecycle management

---

#### openai.go (164 lines)
**Purpose**: OpenAI API integration for AI-powered parsing

**Key Components**:
- **HTTP client**: Standard library net/http for API calls
- **System prompt**: Comprehensive 93-line prompt with parsing rules
- **Request/Response mapping**: JSON serialization for OpenAI API
- **Transaction extraction**: Parses AI response into structured data

**AI Capabilities**:
- Multi-transaction parsing from single text input
- Currency conversion (USD, EUR, GBP, SAR → AED)
- Category classification (10 categories)
- Confidence scoring (0-100)
- Date inference (current year if missing)

---

#### dashboard.go (1087 lines)
**Purpose**: Embedded single-page web application

**Key Components**:
- **Responsive CSS**: Mobile-first design with breakpoints
- **Vanilla JavaScript**: No framework dependencies
- **Real-time updates**: Auto-refresh every 30 seconds
- **Modal UI**: Edit and delete confirmations
- **DOM manipulation**: Direct updates without full refresh
- **Toast notifications**: Non-intrusive success messages

**UI Features**:
- Gradient header with total spending
- Category breakdown with expandable transactions
- Edit modal with form validation
- Delete confirmation modal
- Loading states and error handling
- Last update timestamp with manual refresh

---

#### Dockerfile (40 lines)
**Purpose**: Multi-stage build for optimized container

**Build Strategy**:
1. **Builder stage**: Go 1.22 + gcc + SQLite dev libs
2. **Runtime stage**: Alpine + SQLite runtime libs only
3. **Result**: Small image (~30MB) with all dependencies

**Key Features**:
- CGO enabled for SQLite C bindings
- Static binary compilation
- Separate /data directory for database
- Port 8080 exposed

---

#### docker-compose.yml (22 lines)
**Purpose**: Local development environment

**Configuration**:
- Single service (app)
- Port mapping 8080:8080
- Environment variables from .env file
- Named volume for database persistence
- Auto-restart policy

---

#### fly.toml (22 lines)
**Purpose**: Fly.io deployment configuration

**Settings**:
- App name: transaction-tracker-damp-bush-5483
- Region: Singapore (sin)
- Auto-scaling: sleep when idle, start on request
- Persistent volume: /data mount point
- VM: 1GB RAM, 1 shared CPU
- HTTPS forced

---

## Data Flow

### 1. New Transaction Flow (POST /transaction)

```
┌────────────────────────────────────────────────────────────────┐
│ STEP 1: User Input                                             │
└────────────────────────────────────────────────────────────────┘
    User sends SMS text via HTTP POST
    Example: {"text": "AED 50 at Starbucks Dubai Mall"}
                    │
                    ▼
┌────────────────────────────────────────────────────────────────┐
│ STEP 2: HTTP Request Validation (main.go)                     │
└────────────────────────────────────────────────────────────────┘
    transactionHandler()
    • Check HTTP method is POST
    • Decode JSON request body
    • Validate text field is not empty
    • Log request details with remote address
                    │
                    ▼
┌────────────────────────────────────────────────────────────────┐
│ STEP 3: AI Parsing (openai.go)                                │
└────────────────────────────────────────────────────────────────┘
    OpenAIClient.ParseTransactions(text)

    3a. Build OpenAI Request:
        • Model: "gpt-4o-mini"
        • System prompt (parsing rules, categories, currencies)
        • User message: original SMS text
        • Temperature: 0.3 (deterministic)
        • Max tokens: 1500

    3b. Call OpenAI API:
        POST https://api.openai.com/v1/chat/completions
        Headers:
            Authorization: Bearer {OPENAI_API_KEY}
            Content-Type: application/json

    3c. Parse Response:
        • Extract message content from response
        • Unmarshal JSON array into []Transaction
        • Each transaction has:
            - date: "2026-01-25"
            - description: "Starbucks Dubai Mall"
            - amount: 50.0 (in AED)
            - category: "Food & Dining"
            - confidence: 95
                    │
                    ▼
┌────────────────────────────────────────────────────────────────┐
│ STEP 4: Transaction Enrichment (main.go)                      │
└────────────────────────────────────────────────────────────────┘
    For each parsed transaction:

    enrichTransaction(tx)
    • Add timestamp: RFC3339 format
        Example: "2026-01-25T14:30:00Z"

    • Calculate billing cycle:
        calculateBillingCycle(tx.Date)
        Logic:
            - Parse transaction date
            - If day < 23: cycle = previous month
            - If day >= 23: cycle = current month
            - Format: "Jan 2026"
        Example: Jan 15 → "Dec 2025"
                 Jan 25 → "Jan 2026"
                    │
                    ▼
┌────────────────────────────────────────────────────────────────┐
│ STEP 5: Database Storage (database.go)                        │
└────────────────────────────────────────────────────────────────┘
    DatabaseClient.SaveTransaction(enrichedTx)

    SQL Query:
        INSERT INTO transactions
        (description, amount, transaction_date, category,
         confidence, billing_cycle, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)

    Constraint Check:
        UNIQUE(description, amount, transaction_date)
        → Prevents duplicate transactions

    Indexed Fields:
        • billing_cycle (for stats queries)
        • transaction_date (for sorting)
        • category (for grouping)
                    │
                    ▼
┌────────────────────────────────────────────────────────────────┐
│ STEP 6: Response Generation (main.go)                         │
└────────────────────────────────────────────────────────────────┘
    Build TransactionResponse:
    {
        "success": true,
        "message": "✅ Added 1 transaction!\n\n
                    1. Starbucks Dubai Mall
                       💰 Amount: 50.00 AED
                       📁 Category: 🍔 Food & Dining (95% confidence)
                       📅 Cycle: Jan 2026

                    ━━━━━━━━━━━━━━━
                    💵 Total: 50.00 AED",
        "count": 1,
        "total": 50.00,
        "transactions": [{ enriched transaction object }]
    }
                    │
                    ▼
┌────────────────────────────────────────────────────────────────┐
│ STEP 7: Client Response                                       │
└────────────────────────────────────────────────────────────────┘
    HTTP 200 OK
    Content-Type: application/json

    Client receives confirmation and can display to user
```

### 2. Dashboard Stats Flow (GET /stats)

```
┌────────────────────────────────────────────────────────────────┐
│ STEP 1: HTTP Request                                           │
└────────────────────────────────────────────────────────────────┘
    Browser loads dashboard or JavaScript calls loadStats()
    GET /stats
                    │
                    ▼
┌────────────────────────────────────────────────────────────────┐
│ STEP 2: Calculate Current Cycle (database.go)                 │
└────────────────────────────────────────────────────────────────┘
    currentCycle = calculateBillingCycle(today)
    Example: Today is Jan 25, 2026 → "Jan 2026"
                    │
                    ▼
┌────────────────────────────────────────────────────────────────┐
│ STEP 3: Query Total & Count                                   │
└────────────────────────────────────────────────────────────────┘
    SQL:
        SELECT COALESCE(SUM(amount), 0), COUNT(*)
        FROM transactions
        WHERE billing_cycle = 'Jan 2026'
          AND category != 'Income/Transfer'

    Result: total = 450.50, count = 12
                    │
                    ▼
┌────────────────────────────────────────────────────────────────┐
│ STEP 4: Query Category Breakdown                              │
└────────────────────────────────────────────────────────────────┘
    SQL:
        SELECT category, SUM(amount) as total, COUNT(*) as count
        FROM transactions
        WHERE billing_cycle = 'Jan 2026'
        GROUP BY category
        ORDER BY total DESC

    Results:
        Food & Dining    → 200.50 AED (5 transactions)
        Transport        → 150.00 AED (4 transactions)
        Shopping         → 100.00 AED (3 transactions)
                    │
                    ▼
┌────────────────────────────────────────────────────────────────┐
│ STEP 5: Fetch Transactions Per Category                       │
└────────────────────────────────────────────────────────────────┘
    For each category:
        SQL:
            SELECT id, description, amount, transaction_date,
                   category, confidence, billing_cycle, created_at
            FROM transactions
            WHERE billing_cycle = 'Jan 2026'
              AND category = 'Food & Dining'
            ORDER BY transaction_date DESC, created_at DESC

    Build CategoryStats with transactions array
                    │
                    ▼
┌────────────────────────────────────────────────────────────────┐
│ STEP 6: Sort Categories & Format Response                     │
└────────────────────────────────────────────────────────────────┘
    • Sort categories by total (descending)
    • Add emoji for each category
    • Query last transaction
    • Build formatted message string

    Return StatsResponse with:
        - cycle: "Jan 2026"
        - total: 450.50
        - count: 12
        - categories: [CategoryStats array]
        - lastTransaction: {details}
                    │
                    ▼
┌────────────────────────────────────────────────────────────────┐
│ STEP 7: Client Renders Dashboard                              │
└────────────────────────────────────────────────────────────────┘
    JavaScript:
        • renderStats(data)
        • Build HTML for total section
        • Build category grid with transactions
        • Attach event listeners for expand/collapse
        • Update "Last updated" timestamp
```

### 3. Update Transaction Flow (PUT /transaction/:id)

```
User clicks Edit → openEditModal(tx)
    → Fill form with current values
    → User modifies fields
    → Click Save
        │
        ▼
saveTransaction()
    • Build updated transaction object
    • PUT /transaction/{id}
    • Backend: updateTransactionHandler()
        - Recalculate billing cycle from new date
        - UPDATE transactions SET ... WHERE id = ?
        - Check rowsAffected > 0
        │
        ▼
updateTransactionInDOM(id, oldTx, newTx)
    • Find transaction in categoriesData
    • If category changed:
        - Remove from old category
        - Add to new category (create if needed)
        - Update totals for both
        - Remove old category if empty
    • If category same:
        - Update fields in place
        - Recalculate category total
    • Update global statsData.total
    • updateTotalsInDOM() - refresh header
    • updateCategoriesInDOM() - rebuild categories grid
    • showToast("Transaction updated successfully")
    │
    ▼
DOM updated without full page refresh
Category automatically expands if previously expanded
```

### 4. Delete Transaction Flow (DELETE /transaction/:id)

```
User clicks Delete → openDeleteModal(id, desc, amount)
    → Show confirmation modal
    → User clicks Confirm
        │
        ▼
confirmDelete()
    • DELETE /transaction/{id}
    • Backend: deleteTransactionHandler()
        - DELETE FROM transactions WHERE id = ?
        - Check rowsAffected > 0
        │
        ▼
deleteTransactionFromDOM(id)
    • Find and remove from categoriesData
    • Decrease category total and count
    • Decrease statsData.total and count
    • Remove category if count = 0
    • updateTotalsInDOM()
    • updateCategoriesInDOM()
    • showToast("Transaction deleted successfully")
    │
    ▼
DOM updated instantly
Deleted item removed from view
Totals recalculated and displayed
```

---

## Database Schema

### Table: transactions

```sql
CREATE TABLE IF NOT EXISTS transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    description TEXT NOT NULL,
    amount REAL NOT NULL,
    transaction_date TEXT NOT NULL,
    category TEXT NOT NULL,
    confidence INTEGER,
    billing_cycle TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(description, amount, transaction_date)
);
```

### Indexes

```sql
CREATE INDEX IF NOT EXISTS idx_billing_cycle ON transactions(billing_cycle);
CREATE INDEX IF NOT EXISTS idx_transaction_date ON transactions(transaction_date);
CREATE INDEX IF NOT EXISTS idx_category ON transactions(category);
```

### Field Descriptions

| Field | Type | Nullable | Description | Example |
|-------|------|----------|-------------|---------|
| `id` | INTEGER | NO | Auto-incrementing primary key | 1 |
| `description` | TEXT | NO | Merchant or transaction description | "Starbucks Dubai Mall" |
| `amount` | REAL | NO | Amount in AED (positive for expenses, negative for income) | 50.00 |
| `transaction_date` | TEXT | NO | Date in YYYY-MM-DD format | "2026-01-25" |
| `category` | TEXT | NO | Transaction category (one of 10 predefined) | "Food & Dining" |
| `confidence` | INTEGER | YES | AI confidence score (0-100) | 95 |
| `billing_cycle` | TEXT | NO | Billing cycle in "Mon YYYY" format | "Jan 2026" |
| `created_at` | TEXT | NO | Timestamp when record was created (RFC3339) | "2026-01-25T14:30:00Z" |

### Categories

Fixed set of 10 categories:

| Category | Emoji | Use Cases |
|----------|-------|-----------|
| Food & Dining | 🍔 | Restaurants, cafes, food delivery |
| Transport | 🚗 | Uber, Careem, gas, parking, car maintenance |
| Shopping | 🛍️ | Retail, online shopping, clothing, electronics |
| Bills & Utilities | 💳 | Electricity, water, internet, subscriptions |
| Entertainment | 🎬 | Movies, concerts, streaming, gaming |
| Health & Fitness | 💪 | Gym, medical, pharmacy, supplements |
| Travel | ✈️ | Flights, hotels, travel bookings |
| Cash Withdrawal | 💵 | ATM withdrawals |
| Income/Transfer | 💰 | Salary, refunds, transfers (excluded from stats) |
| Unknown | ❓ | Low confidence transactions (< 70%) |

### Unique Constraint Logic

```sql
UNIQUE(description, amount, transaction_date)
```

**Purpose**: Prevent duplicate transactions

**Example**:
- First insert: "Starbucks", 50.00, "2026-01-25" → Success
- Second insert: "Starbucks", 50.00, "2026-01-25" → SQLite error (duplicate)
- Application: Logs error, continues with next transaction

**Note**: Different descriptions OR amounts OR dates will create separate records

### Query Performance

**Optimized Queries**:

1. **Stats by billing cycle**:
```sql
SELECT COALESCE(SUM(amount), 0), COUNT(*)
FROM transactions
WHERE billing_cycle = ?
  AND category != 'Income/Transfer'
```
Uses: `idx_billing_cycle` index → O(log n) lookup

2. **Category breakdown**:
```sql
SELECT category, SUM(amount), COUNT(*)
FROM transactions
WHERE billing_cycle = ?
GROUP BY category
ORDER BY total DESC
```
Uses: `idx_billing_cycle` + `idx_category` → Efficient grouping

3. **Transactions by category**:
```sql
SELECT *
FROM transactions
WHERE billing_cycle = ? AND category = ?
ORDER BY transaction_date DESC, created_at DESC
```
Uses: Composite scan on both indexes → Fast sorting

---

## API Endpoints

### 1. POST /transaction

**Description**: Log new transaction(s) from SMS text

**Request**:
```http
POST /transaction HTTP/1.1
Content-Type: application/json

{
    "text": "AED 50 at Starbucks. AED 30 at Careem"
}
```

**Request Body Schema**:
```go
type TransactionRequest struct {
    Text string `json:"text"`  // Required: SMS text or transaction description
}
```

**Success Response** (200 OK):
```json
{
    "success": true,
    "message": "✅ Added 2 transactions!\n\n1. Starbucks\n   💰 Amount: 50.00 AED\n   📁 Category: 🍔 Food & Dining (95% confidence)\n   📅 Cycle: Jan 2026\n\n2. Careem\n   💰 Amount: 30.00 AED\n   📁 Category: 🚗 Transport (98% confidence)\n   📅 Cycle: Jan 2026\n\n━━━━━━━━━━━━━━━\n💵 Total: 80.00 AED",
    "count": 2,
    "total": 80.00,
    "transactions": [
        {
            "id": 45,
            "date": "2026-01-25",
            "description": "Starbucks",
            "amount": 50.00,
            "category": "Food & Dining",
            "confidence": 95,
            "timestamp": "2026-01-25T14:30:00Z",
            "billingCycle": "Jan 2026"
        },
        {
            "id": 46,
            "date": "2026-01-25",
            "description": "Careem",
            "amount": 30.00,
            "category": "Transport",
            "confidence": 98,
            "timestamp": "2026-01-25T14:30:05Z",
            "billingCycle": "Jan 2026"
        }
    ]
}
```

**No Transactions Response** (200 OK):
```json
{
    "success": false,
    "message": "No transactions found in the provided text",
    "count": 0
}
```

**Error Responses**:
- **405 Method Not Allowed**: Wrong HTTP method
- **400 Bad Request**: Invalid JSON or empty text field
- **500 Internal Server Error**: OpenAI API failure or database error

**Behavior**:
- Accepts single or multiple transactions in one text input
- Each transaction is parsed independently by AI
- Failed saves are logged but don't fail the entire request
- Duplicate transactions (same description, amount, date) are skipped
- Currency conversion happens automatically in AI layer

---

### 2. GET /stats

**Description**: Get spending statistics for current billing cycle

**Request**:
```http
GET /stats HTTP/1.1
```

**Success Response** (200 OK):
```json
{
    "success": true,
    "message": "📊 Billing Cycle: Jan 2026 (23rd - 22nd)\n━━━━━━━━━━━━━━━\n💰 Total Spent: 450.50 AED\n\nBy Category:\n🍔 Food & Dining: 200.50 AED (5 transactions)\n🚗 Transport: 150.00 AED (4 transactions)\n🛍️ Shopping: 100.00 AED (3 transactions)\n\n━━━━━━━━━━━━━━━\n🕐 Last transaction:\n   Starbucks Dubai Mall - 50.00 AED (today)",
    "cycle": "Jan 2026",
    "total": 450.50,
    "count": 12,
    "categories": [
        {
            "category": "Food & Dining",
            "emoji": "🍔",
            "total": 200.50,
            "count": 5,
            "transactions": [
                {
                    "id": 45,
                    "date": "2026-01-25",
                    "description": "Starbucks Dubai Mall",
                    "amount": 50.00,
                    "category": "Food & Dining",
                    "confidence": 95,
                    "timestamp": "2026-01-25T14:30:00Z",
                    "billingCycle": "Jan 2026"
                }
                // ... more transactions
            ]
        }
        // ... more categories
    ],
    "lastTransaction": {
        "description": "Starbucks Dubai Mall",
        "amount": 50.00,
        "date": "2026-01-25"
    }
}
```

**Empty State Response** (200 OK):
```json
{
    "success": true,
    "message": "📊 Billing Cycle: Jan 2026\n\nNo transactions found for this cycle yet.\n\nStart logging your expenses!",
    "cycle": "Jan 2026",
    "total": 0,
    "count": 0,
    "categories": []
}
```

**Error Responses**:
- **405 Method Not Allowed**: Wrong HTTP method
- **500 Internal Server Error**: Database query failure

**Behavior**:
- Automatically calculates current billing cycle based on today's date
- Excludes "Income/Transfer" category from totals
- Categories sorted by total amount (descending)
- Transactions within categories sorted by date (newest first)
- Last transaction shows relative date ("today" vs "Jan 25")

---

### 3. PUT /transaction/:id

**Description**: Update an existing transaction

**Request**:
```http
PUT /transaction/45 HTTP/1.1
Content-Type: application/json

{
    "description": "Starbucks Mall of Emirates",
    "amount": 55.00,
    "date": "2026-01-24",
    "category": "Food & Dining"
}
```

**Request Body Schema**:
```go
type Transaction struct {
    Description string  `json:"description"`
    Amount      float64 `json:"amount"`
    Date        string  `json:"date"`        // YYYY-MM-DD
    Category    string  `json:"category"`
    // Note: billing_cycle is auto-recalculated from date
}
```

**Success Response** (200 OK):
```json
{
    "success": true,
    "message": "Transaction updated successfully"
}
```

**Error Responses**:
- **400 Bad Request**: Invalid transaction ID or malformed JSON
- **404 Not Found**: Transaction ID doesn't exist
- **405 Method Not Allowed**: Wrong HTTP method
- **500 Internal Server Error**: Database update failure

**Behavior**:
- Billing cycle is automatically recalculated from the new date
- Updates only the specified fields
- Confidence score is not updated (from original AI parsing)
- Created_at timestamp remains unchanged

---

### 4. DELETE /transaction/:id

**Description**: Delete a transaction

**Request**:
```http
DELETE /transaction/45 HTTP/1.1
```

**Success Response** (200 OK):
```json
{
    "success": true,
    "message": "Transaction deleted successfully"
}
```

**Error Responses**:
- **400 Bad Request**: Invalid transaction ID format
- **404 Not Found**: Transaction ID doesn't exist
- **405 Method Not Allowed**: Wrong HTTP method
- **500 Internal Server Error**: Database delete failure

**Behavior**:
- Permanently removes transaction from database
- Cannot be undone
- Frontend shows confirmation modal before deletion
- ID is extracted from URL path after `/transaction/`

---

### 5. GET /

**Description**: Serve dashboard HTML

**Request**:
```http
GET / HTTP/1.1
```

**Success Response** (200 OK):
```http
Content-Type: text/html; charset=utf-8

<!DOCTYPE html>
<html lang="en">
...embedded dashboard HTML...
</html>
```

**Error Responses**:
- **404 Not Found**: Any path other than exactly `/`
- **405 Method Not Allowed**: Wrong HTTP method

**Behavior**:
- Serves embedded HTML from dashboardHTML constant
- Single-page application with embedded CSS and JavaScript
- Auto-loads stats on page load
- Auto-refreshes every 30 seconds

---

### 6. GET /health

**Description**: Health check endpoint

**Request**:
```http
GET /health HTTP/1.1
```

**Success Response** (200 OK):
```json
{
    "status": "healthy",
    "time": "2026-01-25T14:30:00Z"
}
```

**Behavior**:
- Always returns 200 OK if server is running
- Used for monitoring and load balancer health checks
- Returns current server time in RFC3339 format
- Logs each health check with remote address

---

## Frontend Architecture

### Technology Choices

**No Framework Approach**:
- **Why**: Zero dependencies, maximum portability
- **Benefits**: Fast load times, no build step, embedded in binary
- **Trade-offs**: More manual DOM manipulation

### Single HTML File Structure

The entire frontend is embedded as a string constant in `dashboard.go`:

```go
const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <!-- CSS embedded in <style> tags -->
</head>
<body>
    <!-- HTML structure -->
    <!-- JavaScript embedded in <script> tags -->
</body>
</html>`
```

**Size**: 1087 lines total
- CSS: ~500 lines
- HTML: ~100 lines
- JavaScript: ~500 lines

### CSS Architecture

**Design System**:
```css
/* Color Palette */
--primary-gradient: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
--background: white;
--text-primary: #333;
--text-secondary: #666;
--border: #f0f0f0;
--success: #4caf50;
--error: #f44;
```

**Responsive Breakpoints**:
- Mobile: < 480px
- Tablet: 480px - 768px
- Desktop: 768px - 1200px
- Large: > 1200px

**Grid System**:
```css
.categories-grid {
    display: grid;
    grid-template-columns: 1fr;              /* Mobile */
}

@media (min-width: 768px) {
    grid-template-columns: repeat(2, 1fr);   /* Tablet */
}

@media (min-width: 1200px) {
    grid-template-columns: repeat(3, 1fr);   /* Desktop */
}
```

**Component Styles**:
1. **Total Section**: Gradient card with large numbers
2. **Category Cards**: Expandable with hover effects
3. **Transaction Items**: Left-bordered cards with actions
4. **Modals**: Centered overlays with backdrop blur
5. **Toast Notifications**: Slide-in from right with auto-dismiss

### JavaScript Architecture

**State Management**:

```javascript
// Global state (simple variables, no framework)
let statsData = null;          // Current stats response
let categoriesData = [];       // Array of category objects
let currentTransactionId = null; // For edit/delete operations
let currentTransaction = null;  // Original transaction for edit
```

**Why Global State**:
- Simple data flow
- Easy to update from any function
- No need for complex state management library

**Key Functions**:

#### 1. Data Loading
```javascript
async function loadStats() {
    // Fetch /stats endpoint
    // Handle loading state
    // Handle errors
    // Call renderStats() on success
}
```

**Flow**:
```
loadStats()
  ├─> Show loading spinner
  ├─> fetch('/stats')
  ├─> Check response.ok
  ├─> Parse JSON
  ├─> Store in statsData & categoriesData
  ├─> renderStats(data)
  └─> Update lastUpdate timestamp
```

#### 2. Rendering
```javascript
function renderStats(data) {
    // Build HTML string
    // Update content div innerHTML
    // Handles empty state
    // Builds total section, categories, transactions
}
```

**Rendering Strategy**: String concatenation
```javascript
let html = '';
html += '<div class="total-section">';
html += '<div class="total-amount">' + formatCurrency(data.total) + '</div>';
html += '</div>';

content.innerHTML = html;
```

**Why Not Template Literals**: Embedded in Go string, avoiding escape complexity

#### 3. Category Toggle
```javascript
function toggleCategory(index) {
    const list = document.getElementById('transactions-' + index);
    const icon = document.getElementById('icon-' + index);

    if (list.classList.contains('expanded')) {
        list.classList.remove('expanded');
        icon.textContent = '▶';
    } else {
        list.classList.add('expanded');
        icon.textContent = '▼';
    }
}
```

**Animation**: CSS transition on max-height
```css
.transactions-list {
    max-height: 0;
    transition: max-height 0.3s ease;
}

.transactions-list.expanded {
    max-height: 1000px;
}
```

#### 4. Edit Transaction
```javascript
function openEditModal(tx) {
    // Store current transaction
    // Populate form fields
    // Show modal
}

async function saveTransaction() {
    // Build updated transaction object
    // PUT /transaction/:id
    // Close modal on success
    // Update DOM without refresh
}

function updateTransactionInDOM(txId, oldTx, newTx) {
    // Complex logic for category changes
    // Update categoriesData in memory
    // Recalculate totals
    // Rebuild affected DOM sections
    // Show success toast
}
```

**DOM Update Strategy**: Surgical updates

Instead of full page refresh:
1. Update in-memory data (categoriesData, statsData)
2. Recalculate affected totals
3. Rebuild only changed sections (categories grid)
4. Preserve UI state (expanded categories)

**Category Change Logic**:
```javascript
if (categoryChanged) {
    // Remove from old category
    oldCategory.transactions.splice(index, 1);
    oldCategory.total -= oldTx.amount;
    oldCategory.count--;

    // Add to new category (create if needed)
    let newCategory = categoriesData.find(c => c.category === newCategory);
    if (!newCategory) {
        newCategory = createNewCategory();
        categoriesData.push(newCategory);
    }
    newCategory.transactions.push(updatedTx);
    newCategory.total += newTx.amount;
    newCategory.count++;

    // Remove empty categories
    if (oldCategory.count === 0) {
        categoriesData.splice(categoryIndex, 1);
    }
}
```

#### 5. Delete Transaction
```javascript
function openDeleteModal(id, description, amount) {
    // Show confirmation with preview
}

async function confirmDelete() {
    // DELETE /transaction/:id
    // Close modal on success
    // Remove from DOM
}

function deleteTransactionFromDOM(txId) {
    // Find in categoriesData
    // Remove transaction
    // Update category totals
    // Update global totals
    // Remove empty categories
    // Rebuild DOM
    // Show success toast
}
```

#### 6. Toast Notifications
```javascript
function showToast(message) {
    // Create toast element
    // Append to body
    // Trigger slide-in animation
    // Auto-hide after 3 seconds
    // Remove from DOM
}
```

**Animation Sequence**:
```
Create element (opacity: 0, translateX: 400px)
    ↓ 10ms delay
Add 'show' class (opacity: 1, translateX: 0)
    ↓ 3 seconds
Add 'hide' class (opacity: 0, translateX: 400px)
    ↓ 300ms
Remove from DOM
```

### Event Handling

**Auto-load on Page Load**:
```javascript
window.addEventListener('DOMContentLoaded', () => {
    loadStats();
});
```

**Auto-refresh**:
```javascript
setInterval(() => {
    loadStats();
}, 30000); // Every 30 seconds
```

**Manual Refresh**:
```javascript
<button class="refresh-btn" onclick="loadStats()">Refresh</button>
```

**Modal Close on Overlay Click**:
```javascript
document.addEventListener('click', (e) => {
    if (e.target.classList.contains('modal-overlay')) {
        closeEditModal();
        closeDeleteModal();
    }
});
```

**Inline Event Handlers** (for dynamic content):
```javascript
// Category click
onclick="toggleCategory(0)"

// Edit button
onclick='openEditModal({"id":1,"description":"Test",...})'

// Delete button
onclick='openDeleteModal(1, "Test", 50.00)'
```

**Why Inline Handlers**: HTML is dynamically generated as string

### Helper Functions

```javascript
function formatCurrency(amount) {
    return new Intl.NumberFormat('en-AE', {
        style: 'currency',
        currency: 'AED',
        minimumFractionDigits: 2
    }).format(amount);
}
// Output: "AED 50.00"

function formatDate(date) {
    return date.toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric'
    });
}
// Output: "Jan 25"

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
// Prevents XSS attacks
```

### Security Considerations

**XSS Prevention**:
```javascript
// Always escape user content
html += '<div>' + escapeHtml(tx.description) + '</div>';

// JSON in onclick is safe (browser parses as data)
onclick='openEditModal(${JSON.stringify(tx)})'
```

**HTTPS Only** (in production):
- Fly.io forces HTTPS
- Protects API keys in transit

**No Stored Credentials**:
- No authentication in frontend
- API keys only in backend environment

---

## Key Features

### 1. AI-Powered Transaction Parsing

**Capability**: Extract structured data from natural language

**Input Examples**:
```
"AED 50 at Starbucks"
"Spent $20 at uber"
"Your account was debited AED 150.50 at Carrefour Dubai Mall on 25 Jan"
"Paid 30 USD for Netflix subscription"
```

**Output**:
```go
Transaction{
    Date:        "2026-01-25",
    Description: "Starbucks",
    Amount:      50.00,  // Converted to AED
    Category:    "Food & Dining",
    Confidence:  95
}
```

**AI Model**: GPT-4o-mini
- **Cost**: ~$0.0001 per transaction
- **Speed**: ~1-2 seconds response time
- **Accuracy**: 95%+ with well-formatted SMS

**System Prompt Features**:
1. Multi-transaction parsing
2. Currency conversion (5 currencies)
3. Category classification (10 categories)
4. Confidence scoring
5. Date inference (uses current year if missing)
6. Conservative categorization (Unknown if < 70% confidence)

---

### 2. Multi-Currency Conversion

**Supported Currencies**:
```go
AED → 1.00 (base)
USD → × 3.67
EUR → × 4.00
GBP → × 4.70
SAR → × 0.98
Others → AI determines approximate rate
```

**Conversion Logic**: In AI layer (systemPrompt)

**Example**:
```
Input:  "Paid $100 for shopping"
AI:     amount = 100 * 3.67 = 367.00
Output: amount: 367.00, category: "Shopping"
```

**Why AI-based**:
- Handles unusual currencies
- Can interpret context ("hundred dollars" vs "$100")
- Updates automatically with market changes (prompt update)

---

### 3. Billing Cycle Tracking

**Cycle Definition**: 23rd to 22nd of each month

**Logic** (in calculateBillingCycle function):
```go
func calculateBillingCycle(dateStr string) string {
    txDate, _ := time.Parse("2006-01-02", dateStr)

    cycleStart := txDate
    if txDate.Day() < 23 {
        // Before 23rd: use previous month
        cycleStart = txDate.AddDate(0, -1, 0)
    }
    // Set to 23rd of cycle month
    cycleStart = time.Date(
        cycleStart.Year(),
        cycleStart.Month(),
        23, 0, 0, 0, 0,
        time.UTC
    )

    return cycleStart.Format("Jan 2006")
}
```

**Examples**:
```
Jan 15, 2026 → "Dec 2025" (before cutoff)
Jan 23, 2026 → "Jan 2026" (on cutoff)
Jan 25, 2026 → "Jan 2026" (after cutoff)
Feb 22, 2026 → "Jan 2026" (last day of cycle)
Feb 23, 2026 → "Feb 2026" (new cycle)
```

**Why 23rd**: Credit card billing cycle alignment

---

### 4. Smart Categorization

**Categories with Confidence**:

```json
{
    "category": "Food & Dining",
    "confidence": 95
}
```

**AI Decision Tree**:
1. Exact merchant match → High confidence (90-100%)
   - "Starbucks" → Food & Dining (98%)
   - "Careem" → Transport (99%)

2. Keyword match → Medium confidence (70-89%)
   - "restaurant" → Food & Dining (85%)
   - "taxi" → Transport (80%)

3. Ambiguous → Low confidence (< 70%)
   - "Amazon" → Shopping (65%) or Unknown
   - "Payment" → Unknown (50%)

**Handling Unknown**:
- Prompt rule: "Only use Unknown if confidence < 70"
- UI: Shows with ❓ emoji
- User can manually edit via dashboard

---

### 5. Real-Time Dashboard

**Features**:
- **Auto-load**: Fetches stats on page load
- **Auto-refresh**: Updates every 30 seconds
- **Manual refresh**: Button for immediate update
- **Last update timestamp**: Shows when data was last fetched

**Responsive Design**:
- Mobile: Single column, stacked cards
- Tablet: 2-column grid
- Desktop: 3-column grid
- All layouts support portrait/landscape

**Visual Hierarchy**:
```
┌─────────────────────────────────┐
│  💰 Transaction Tracker         │
│  Billing Cycle Dashboard        │
├─────────────────────────────────┤
│  Total Spent                    │
│  AED 450.50                     │
│  12 transactions in Jan 2026    │
├─────────────────────────────────┤
│  🕐 Last Transaction            │
│  Starbucks - AED 50.00 (today)  │
├─────────────────────────────────┤
│  By Category                    │
│  ┌───────────┬───────────┬──────┐
│  │🍔 Food    │🚗 Transport│🛍️ Shop│
│  │200.50 AED │150.00 AED │100 AED│
│  └───────────┴───────────┴──────┘
└─────────────────────────────────┘
```

---

### 6. Edit & Delete Transactions

**Recent Addition**: Full CRUD operations with DOM updates

**Edit Flow**:
1. Click "✏️ Edit" on transaction
2. Modal appears with pre-filled form
3. Modify description, amount, date, or category
4. Click "Save Changes"
5. PUT request to /transaction/:id
6. DOM updates instantly:
   - Transaction moves to new category if changed
   - Totals recalculate
   - Category removed if empty
   - Toast notification shows
7. No page refresh needed

**Delete Flow**:
1. Click "🗑️ Delete" on transaction
2. Confirmation modal with preview
3. Click "Delete" to confirm
4. DELETE request to /transaction/:id
5. DOM updates instantly:
   - Transaction removed from view
   - Category total decreases
   - Global total decreases
   - Category removed if empty
   - Toast notification shows

**Smart DOM Updates**:
- Tracks expanded categories
- Preserves expansion state after update
- Smooth animations (CSS transitions)
- No flash or reload

---

### 7. Duplicate Prevention

**Mechanism**: Database UNIQUE constraint

```sql
UNIQUE(description, amount, transaction_date)
```

**Behavior**:
```
First:  "Starbucks", 50.00, "2026-01-25" → Saved
Second: "Starbucks", 50.00, "2026-01-25" → SQLite error
App:    Logs error, continues with next transaction
```

**Why Useful**:
- User accidentally submits same SMS twice
- Multiple devices logging same transaction
- Prevents inflated totals

**Edge Cases**:
- Same merchant, different amount → Separate records
- Same merchant, same amount, different date → Separate records
- Different description, same amount/date → Separate records

---

### 8. Comprehensive Logging

**Log Format**:
```
[Timestamp with microseconds] [Component] Message
```

**Example**:
```
2026-01-25 14:30:00.123456 [Server] Starting Transaction Tracker...
2026-01-25 14:30:00.234567 [Database] Connecting to database at: ./transactions.db
2026-01-25 14:30:00.345678 [Database] Running migrations...
2026-01-25 14:30:00.456789 [Server] Server ready at http://localhost:8080
2026-01-25 14:30:15.567890 [API] POST /transaction - New transaction request from 127.0.0.1:54321
2026-01-25 14:30:15.678901 [API] Processing transaction text: AED 50 at Starbucks
2026-01-25 14:30:17.789012 [API] OpenAI parsed 1 transaction(s)
2026-01-25 14:30:17.890123 [Database] Saving transaction: Starbucks (50.00 AED)
2026-01-25 14:30:17.901234 [Database] Transaction saved successfully
2026-01-25 14:30:17.912345 [API] Successfully saved 1/1 transaction(s), total: 50.00 AED
```

**Log Components**:
- `[Server]`: Application lifecycle
- `[Database]`: DB operations
- `[API]`: HTTP requests/responses
- Includes remote addresses for security auditing

**Configuration**:
```go
log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
```

---

### 9. Zero-Dependency Backend

**Only External Dependency**: SQLite driver

```go
// go.mod
require github.com/mattn/go-sqlite3 v1.14.33
```

**All Other Packages**: Go stdlib
- `net/http` - HTTP server
- `encoding/json` - JSON parsing
- `database/sql` - DB interface
- `log` - Logging
- `time` - Date calculations
- `fmt` - String formatting

**Benefits**:
- Minimal attack surface
- Fast compilation
- Easy to audit
- No dependency conflicts
- Long-term stability

---

### 10. Containerized Deployment

**Multi-Stage Build**:

```dockerfile
# Stage 1: Builder
FROM golang:1.22-alpine
RUN apk add gcc musl-dev sqlite-dev
COPY . .
RUN CGO_ENABLED=1 go build -o transaction-tracker

# Stage 2: Runtime
FROM alpine:latest
RUN apk add ca-certificates sqlite-libs
COPY --from=builder /app/transaction-tracker .
CMD ["./transaction-tracker"]
```

**Result**: ~30MB image with all dependencies

**Docker Compose Features**:
- Named volume for data persistence
- Environment file support
- Port mapping
- Auto-restart on failure

**Fly.io Features**:
- Persistent volume mount at /data
- Auto-scaling (sleep when idle)
- HTTPS automatic
- Health checks
- Zero-downtime deploys

---

## Configuration

### Environment Variables

| Variable | Required | Default | Description | Example |
|----------|----------|---------|-------------|---------|
| `OPENAI_API_KEY` | ✅ Yes | - | OpenAI API key for GPT-4o-mini | `sk-proj-abc123...` |
| `DATABASE_PATH` | ❌ No | `./transactions.db` | Path to SQLite database file | `/data/transactions.db` |
| `PORT` | ❌ No | `8080` | HTTP server port | `3000` |

### Configuration Loading

```go
func loadConfig() (*Config, error) {
    config := &Config{
        OpenAIKey:    os.Getenv("OPENAI_API_KEY"),
        DatabasePath: os.Getenv("DATABASE_PATH"),
        Port:         os.Getenv("PORT"),
    }

    // Apply defaults
    if config.Port == "" {
        config.Port = "8080"
    }
    if config.DatabasePath == "" {
        config.DatabasePath = "./transactions.db"
    }

    // Validate required
    if config.OpenAIKey == "" {
        return nil, fmt.Errorf("OPENAI_API_KEY is required")
    }

    return config, nil
}
```

### Environment File (.env)

```bash
# .env (not committed to Git)
OPENAI_API_KEY=sk-proj-your-actual-key-here
DATABASE_PATH=./transactions.db
PORT=8080
```

**Loading in Different Environments**:

**Local Development**:
```bash
# Load from .env file
export $(cat .env | xargs)
go run *.go
```

**Docker**:
```bash
# docker-compose.yml
env_file:
  - .env
```

**Fly.io**:
```bash
# Set as secrets (encrypted)
fly secrets set OPENAI_API_KEY=sk-proj-...
fly secrets set DATABASE_PATH=/data/transactions.db
```

### Database Path Configuration

**Local Development**:
```bash
DATABASE_PATH=./transactions.db
```
→ Creates file in current directory

**Docker**:
```bash
DATABASE_PATH=/data/transactions.db
```
→ Uses mounted volume (persists across container restarts)

**Fly.io**:
```bash
DATABASE_PATH=/data/transactions.db
```
→ Uses persistent volume (specified in fly.toml)

### Port Configuration

**Default**: 8080

**Custom Port**:
```bash
PORT=3000
go run *.go
# Server starts on :3000
```

**Fly.io Override**:
```toml
# fly.toml
[http_service]
  internal_port = 8080  # Must match PORT env var
```

---

## Deployment

### Local Development

**Method 1: Direct Run**
```bash
# Install dependencies
go mod download

# Run
go run *.go
```

**Method 2: Build & Run**
```bash
# Build binary
go build -o transaction-tracker

# Run
./transaction-tracker
```

**Method 3: Docker Compose**
```bash
# Build and start
docker-compose up --build

# Run in background
docker-compose up -d

# View logs
docker-compose logs -f

# Stop
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

### Docker Deployment

**Build**:
```bash
docker build -t transaction-tracker .
```

**Run**:
```bash
docker run -p 8080:8080 \
  -e OPENAI_API_KEY=sk-proj-... \
  -e DATABASE_PATH=/data/transactions.db \
  -v transactions-data:/data \
  transaction-tracker
```

**Volume Persistence**:
```bash
# Data survives container restart
docker stop <container-id>
docker start <container-id>
# Database still intact at /data/transactions.db
```

### Fly.io Cloud Deployment

**Prerequisites**:
```bash
# Install Fly CLI
brew install flyctl  # macOS
# or
curl -L https://fly.io/install.sh | sh  # Linux/WSL
```

**Initial Setup**:
```bash
# Login
fly auth login

# Launch (creates app)
fly launch
# Follow prompts:
#   - App name: transaction-tracker-xxx
#   - Region: Singapore (sin) or closest
#   - Create Postgres? No (using SQLite)
#   - Deploy now? No (set secrets first)
```

**Configure Secrets**:
```bash
fly secrets set OPENAI_API_KEY=sk-proj-your-key-here
fly secrets set DATABASE_PATH=/data/transactions.db
```

**Create Persistent Volume**:
```bash
fly volumes create transactions_data --size 1
```

**Deploy**:
```bash
fly deploy
```

**Monitor**:
```bash
# View logs
fly logs

# SSH into machine
fly ssh console

# Check status
fly status

# Open in browser
fly open
```

**Update**:
```bash
# After code changes
fly deploy

# Zero-downtime deployment
# Fly.io automatically handles rolling update
```

**Scaling**:
```bash
# Auto-scaling configured in fly.toml
[http_service]
  auto_stop_machines = 'stop'
  auto_start_machines = true
  min_machines_running = 0

# Behavior:
# - Sleeps after 5 minutes of inactivity
# - Wakes on first request (~1-2s delay)
# - Free tier: 3 VMs max
```

**Cost**:
- **Free Tier**: 3 shared VMs, 160GB transfer
- **Volume**: $0.15/GB/month (1GB = $0.15/month)
- **Total**: ~$0.15/month for this app

### Production Checklist

**Before Deploying**:
- [ ] Set strong OpenAI API key
- [ ] Configure DATABASE_PATH for persistent storage
- [ ] Set appropriate PORT (or use default 8080)
- [ ] Test locally with production-like data
- [ ] Verify volume mounts work
- [ ] Check database migrations run successfully

**Security**:
- [ ] Never commit .env file
- [ ] Use secrets management (fly secrets)
- [ ] Enable HTTPS (automatic on Fly.io)
- [ ] Monitor API usage for unusual activity
- [ ] Rotate API keys periodically

**Monitoring**:
- [ ] Set up log aggregation
- [ ] Monitor /health endpoint
- [ ] Track OpenAI API costs
- [ ] Watch database file size
- [ ] Set up alerts for errors

---

## Code Patterns

### 1. Error Handling

**Pattern**: Explicit error checking with context

```go
// Check every error
result, err := someOperation()
if err != nil {
    log.Printf("[Component] Failed to do X: %v", err)
    return fmt.Errorf("failed to do X: %w", err)
}

// HTTP error responses
if err != nil {
    log.Printf("[API] Error: %v", err)
    http.Error(w, "User-friendly message", http.StatusInternalServerError)
    return
}
```

**Error Wrapping**:
```go
// Add context while preserving original error
return fmt.Errorf("failed to save transaction: %w", err)

// Unwrap later if needed
errors.Is(err, sql.ErrNoRows)
```

**Example from database.go**:
```go
func (c *DatabaseClient) SaveTransaction(tx Transaction) error {
    _, err := c.db.Exec(query, ...)
    if err != nil {
        log.Printf("[Database] Failed to save transaction: %v", err)
        return fmt.Errorf("failed to save transaction: %w", err)
    }
    return nil
}
```

### 2. Logging Pattern

**Structured Logging**:
```go
// Component prefix for filtering
log.Printf("[Server] Message")
log.Printf("[Database] Message")
log.Printf("[API] Message")

// Include context
log.Printf("[API] POST /transaction - Request from %s", r.RemoteAddr)
log.Printf("[Database] Saving transaction: %s (%.2f AED)", desc, amount)

// Log both start and completion
log.Printf("[Database] Fetching stats for billing cycle: %s", cycle)
// ... operation ...
log.Printf("[Database] Found %d transactions, total: %.2f AED", count, total)
```

**Log Levels** (implicit):
- Normal operation: `log.Printf(...)`
- Startup/config: `log.Printf(...)`
- Fatal errors: `log.Fatalf(...)` (exits program)

**Configuration**:
```go
log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
// Output: 2026/01/25 14:30:00.123456 [API] Message
```

### 3. HTTP Handler Pattern

**Closure for Dependency Injection**:
```go
func handlerName(dep1 *Dep1, dep2 *Dep2) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Handler logic with access to dependencies
        dep1.Method()
        dep2.Method()
    }
}

// Usage in main()
http.HandleFunc("/endpoint", handlerName(dep1, dep2))
```

**Example**:
```go
func transactionHandler(openAI *OpenAIClient, db *DatabaseClient) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Has access to openAI and db
        transactions, err := openAI.ParseTransactions(text)
        db.SaveTransaction(tx)
    }
}
```

**Why**: Avoid global variables, testable handlers

### 4. JSON Handling

**Request Parsing**:
```go
var req RequestType
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    http.Error(w, "Invalid request body", http.StatusBadRequest)
    return
}
```

**Response Writing**:
```go
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(responseData)
```

**Why `Encoder/Decoder`**: Streams data, memory efficient

### 5. Database Patterns

**Connection Management**:
```go
// Initialize once in main
dbClient, err := NewDatabaseClient(path)
defer dbClient.Close()

// Pass to handlers
http.HandleFunc("/endpoint", handler(dbClient))
```

**Query Pattern**:
```go
// Use placeholders (prevents SQL injection)
db.QueryRow("SELECT * FROM table WHERE id = ?", id)

// Scan results
var field1, field2 string
err := row.Scan(&field1, &field2)
```

**Transaction Safety** (SQLite):
```go
// Set connection pool for SQLite
db.SetMaxOpenConns(1)  // Only 1 writer at a time
db.SetMaxIdleConns(1)
db.SetConnMaxLifetime(0)
```

**Migration Pattern**:
```go
func runMigrations() error {
    migrations := []string{
        `CREATE TABLE IF NOT EXISTS ...`,
        `CREATE INDEX IF NOT EXISTS ...`,
    }
    for _, migration := range migrations {
        if _, err := db.Exec(migration); err != nil {
            return err
        }
    }
    return nil
}
```

### 6. OpenAI API Pattern

**Request Structure**:
```go
type openAIRequest struct {
    Model       string
    Messages    []openAIMessage
    Temperature float64
    MaxTokens   int
}

reqBody := openAIRequest{
    Model: "gpt-4o-mini",
    Messages: []openAIMessage{
        {Role: "system", Content: systemPrompt},
        {Role: "user", Content: userText},
    },
    Temperature: 0.3,  // Low = deterministic
    MaxTokens:   1500,
}
```

**HTTP Client**:
```go
client := &http.Client{}  // Reuse across requests

req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
req.Header.Set("Authorization", "Bearer " + apiKey)
req.Header.Set("Content-Type", "application/json")

resp, err := client.Do(req)
defer resp.Body.Close()
```

**Response Parsing**:
```go
var openAIResp openAIResponse
json.Unmarshal(body, &openAIResp)

content := openAIResp.Choices[0].Message.Content

var transactions []Transaction
json.Unmarshal([]byte(content), &transactions)
```

### 7. Time Handling

**Consistent Format**: RFC3339 for timestamps
```go
timestamp := time.Now().Format(time.RFC3339)
// "2026-01-25T14:30:00Z"
```

**Date Format**: YYYY-MM-DD for dates
```go
date := time.Now().Format("2006-01-02")
// "2026-01-25"
```

**Billing Cycle Format**: "Mon YYYY"
```go
cycle := cycleStart.Format("Jan 2006")
// "Jan 2026"
```

**Parsing**:
```go
txDate, err := time.Parse("2006-01-02", dateStr)
if err != nil {
    // Handle invalid date
}
```

---

## Important Functions

### 1. calculateBillingCycle()

**Location**: main.go

**Purpose**: Determine which billing cycle a date belongs to

**Signature**:
```go
func calculateBillingCycle(dateStr string) string
```

**Logic**:
```go
func calculateBillingCycle(dateStr string) string {
    // Parse date
    txDate, err := time.Parse("2006-01-02", dateStr)
    if err != nil {
        txDate = time.Now()  // Fallback to today
    }

    // Start with transaction month
    cycleStart := txDate

    // If before 23rd, belongs to previous month's cycle
    if txDate.Day() < 23 {
        cycleStart = txDate.AddDate(0, -1, 0)
    }

    // Set to 23rd of cycle month
    cycleStart = time.Date(
        cycleStart.Year(),
        cycleStart.Month(),
        23, 0, 0, 0, 0,
        time.UTC,
    )

    // Format as "Jan 2026"
    return cycleStart.Format("Jan 2006")
}
```

**Examples**:
```
Input: "2026-01-15" → Output: "Dec 2025"
Input: "2026-01-23" → Output: "Jan 2026"
Input: "2026-01-31" → Output: "Jan 2026"
Input: "2026-02-22" → Output: "Jan 2026"
Input: "2026-02-23" → Output: "Feb 2026"
```

**Why Important**: Core business logic for cycle tracking

---

### 2. ParseTransactions()

**Location**: openai.go

**Purpose**: Convert natural language to structured transactions

**Signature**:
```go
func (c *OpenAIClient) ParseTransactions(text string) ([]Transaction, error)
```

**Flow**:
```go
1. Build OpenAI request with system prompt + user text
2. Marshal to JSON
3. POST to https://api.openai.com/v1/chat/completions
4. Check status code
5. Parse response JSON
6. Extract message content
7. Unmarshal content into []Transaction
8. Return transactions
```

**Error Handling**:
```go
// HTTP errors
if resp.StatusCode != 200 {
    return nil, fmt.Errorf("OpenAI API error (status %d): %s", ...)
}

// No response
if len(openAIResp.Choices) == 0 {
    return nil, fmt.Errorf("no response from OpenAI")
}

// Parse errors
if err := json.Unmarshal([]byte(content), &transactions); err != nil {
    return nil, fmt.Errorf("failed to parse transactions: %w", err)
}
```

**Why Important**: Heart of AI parsing logic

---

### 3. GetStats()

**Location**: database.go

**Purpose**: Generate comprehensive statistics for current cycle

**Signature**:
```go
func (c *DatabaseClient) GetStats() (*StatsResponse, error)
```

**Complex Query Flow**:

```go
// 1. Calculate current cycle
currentCycle := calculateBillingCycle(time.Now().Format("2006-01-02"))

// 2. Get totals
QueryRow(`
    SELECT COALESCE(SUM(amount), 0), COUNT(*)
    FROM transactions
    WHERE billing_cycle = ?
      AND category != 'Income/Transfer'
`, currentCycle)

// 3. Get category breakdown
Query(`
    SELECT category, SUM(amount), COUNT(*)
    FROM transactions
    WHERE billing_cycle = ?
    GROUP BY category
    ORDER BY total DESC
`, currentCycle)

// 4. For each category, get transactions
Query(`
    SELECT id, description, amount, ...
    FROM transactions
    WHERE billing_cycle = ? AND category = ?
    ORDER BY transaction_date DESC, created_at DESC
`, currentCycle, category)

// 5. Get last transaction
QueryRow(`
    SELECT description, amount, transaction_date
    FROM transactions
    WHERE billing_cycle = ?
    ORDER BY transaction_date DESC, created_at DESC
    LIMIT 1
`, currentCycle)

// 6. Build formatted message
// 7. Return StatsResponse
```

**Performance**:
- Uses indexed queries (fast even with 10,000+ transactions)
- Efficient grouping and sorting
- Single cycle focus (current only)

**Why Important**: Powers entire dashboard

---

### 4. SaveTransaction()

**Location**: database.go

**Purpose**: Persist transaction to SQLite

**Signature**:
```go
func (c *DatabaseClient) SaveTransaction(tx Transaction) error
```

**Implementation**:
```go
func (c *DatabaseClient) SaveTransaction(tx Transaction) error {
    query := `
        INSERT INTO transactions
        (description, amount, transaction_date, category,
         confidence, billing_cycle, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `

    log.Printf("[Database] Saving transaction: %s (%.2f AED)",
        tx.Description, tx.Amount)

    _, err := c.db.Exec(
        query,
        tx.Description,
        tx.Amount,
        tx.Date,
        tx.Category,
        tx.Confidence,
        tx.BillingCycle,
        tx.Timestamp,
    )

    if err != nil {
        log.Printf("[Database] Failed to save: %v", err)
        return fmt.Errorf("failed to save transaction: %w", err)
    }

    log.Printf("[Database] Transaction saved successfully")
    return nil
}
```

**Duplicate Handling**:
```go
// SQLite returns error if UNIQUE constraint violated
// Application logs error but continues processing
```

**Why Important**: Data persistence layer

---

### 5. enrichTransaction()

**Location**: main.go

**Purpose**: Add metadata to parsed transaction

**Signature**:
```go
func enrichTransaction(tx Transaction) Transaction
```

**Implementation**:
```go
func enrichTransaction(tx Transaction) Transaction {
    // Add timestamp
    tx.Timestamp = time.Now().Format(time.RFC3339)

    // Calculate billing cycle
    tx.BillingCycle = calculateBillingCycle(tx.Date)

    return tx
}
```

**Before**:
```go
Transaction{
    Date:        "2026-01-25",
    Description: "Starbucks",
    Amount:      50.00,
    Category:    "Food & Dining",
    Confidence:  95,
}
```

**After**:
```go
Transaction{
    Date:         "2026-01-25",
    Description:  "Starbucks",
    Amount:       50.00,
    Category:     "Food & Dining",
    Confidence:   95,
    Timestamp:    "2026-01-25T14:30:00Z",    // Added
    BillingCycle: "Jan 2026",                // Added
}
```

**Why Important**: Prepares transaction for database storage

---

### 6. updateTransactionInDOM()

**Location**: dashboard.go (JavaScript)

**Purpose**: Update DOM without page refresh after edit

**Signature**:
```javascript
function updateTransactionInDOM(txId, oldTx, newTx)
```

**Complex Logic**:

```javascript
function updateTransactionInDOM(txId, oldTx, newTx) {
    const oldCategory = oldTx.category;
    const newCategory = newTx.category;
    const categoryChanged = oldCategory !== newCategory;

    // Find transaction in categoriesData
    for (let cat of categoriesData) {
        const txIndex = cat.transactions.findIndex(t => t.id === txId);

        if (txIndex !== -1) {
            if (categoryChanged) {
                // CATEGORY CHANGE FLOW

                // 1. Remove from old category
                const removedTx = cat.transactions.splice(txIndex, 1)[0];
                cat.total -= oldTx.amount;
                cat.count--;

                // 2. Update transaction data
                removedTx.description = newTx.description;
                removedTx.amount = newTx.amount;
                removedTx.date = newTx.date;
                removedTx.category = newTx.category;

                // 3. Find or create new category
                let newCat = categoriesData.find(c => c.category === newCategory);
                if (!newCat) {
                    newCat = {
                        category: newCategory,
                        emoji: getCategoryEmojiByName(newCategory),
                        total: 0,
                        count: 0,
                        transactions: []
                    };
                    categoriesData.push(newCat);
                }

                // 4. Add to new category
                newCat.transactions.push(removedTx);
                newCat.total += newTx.amount;
                newCat.count++;

                // 5. Remove old category if empty
                if (cat.count === 0) {
                    const catIndex = categoriesData.indexOf(cat);
                    categoriesData.splice(catIndex, 1);
                }
            } else {
                // SAME CATEGORY FLOW

                // Just update fields
                cat.transactions[txIndex].description = newTx.description;
                cat.transactions[txIndex].amount = newTx.amount;
                cat.transactions[txIndex].date = newTx.date;

                // Update category total
                cat.total = cat.total - oldTx.amount + newTx.amount;
            }
            break;
        }
    }

    // Update global total
    statsData.total = statsData.total - oldTx.amount + newTx.amount;

    // Refresh DOM
    updateTotalsInDOM();
    updateCategoriesInDOM();

    // Show success message
    showToast('Transaction updated successfully');
}
```

**Why Important**: Provides instant feedback without reload

---

### 7. updateCategoriesInDOM()

**Location**: dashboard.go (JavaScript)

**Purpose**: Rebuild categories grid while preserving UI state

**Signature**:
```javascript
function updateCategoriesInDOM()
```

**Smart Rebuild**:

```javascript
function updateCategoriesInDOM() {
    // 1. Sort by total
    categoriesData.sort((a, b) => b.total - a.total);

    // 2. Remember expanded categories
    const expandedCategories = new Set();
    document.querySelectorAll('.transactions-list.expanded').forEach(list => {
        const index = list.id.replace('transactions-', '');
        const categoryName = categoriesData[parseInt(index)]?.category;
        if (categoryName) expandedCategories.add(categoryName);
    });

    // 3. Rebuild HTML
    let html = '';
    categoriesData.forEach((cat, index) => {
        const isExpanded = expandedCategories.has(cat.category);

        html += buildCategoryHTML(cat, index, isExpanded);
    });

    // 4. Replace grid content
    gridContainer.innerHTML = html;

    // Expanded state preserved automatically via class names
}
```

**Why Important**: Smooth UX with preserved state

---

## Recent Changes

### Edit/Delete Feature (Latest Update)

**What Changed**:
1. Added PUT and DELETE endpoints
2. Added edit and delete modals to dashboard
3. Implemented DOM update functions (no page refresh)
4. Added toast notifications for success feedback

**New Backend Functions**:
- `transactionDetailHandler()` - Routes to update/delete
- `updateTransactionHandler()` - PUT handler
- `deleteTransactionHandler()` - DELETE handler
- `UpdateTransaction()` - Database update
- `DeleteTransaction()` - Database delete

**New Frontend Functions**:
- `openEditModal()` - Show edit form
- `saveTransaction()` - PUT request
- `updateTransactionInDOM()` - Update data & DOM
- `openDeleteModal()` - Show confirmation
- `confirmDelete()` - DELETE request
- `deleteTransactionFromDOM()` - Remove from data & DOM
- `updateTotalsInDOM()` - Refresh header totals
- `updateCategoriesInDOM()` - Rebuild categories
- `showToast()` - Success notifications

**Benefits**:
- No page refresh needed
- Instant feedback
- Preserves UI state (expanded categories)
- Professional UX with modals and toasts

---

### DOM Updates Without Refresh

**Previous Behavior**:
- Edit transaction → Save → Refresh page
- Delete transaction → Confirm → Refresh page

**New Behavior**:
- Edit transaction → Save → Instant DOM update
- Delete transaction → Confirm → Instant DOM update
- Categories automatically reorganize
- Totals recalculate
- Empty categories removed
- Toast notification shows

**Implementation**:

**State Management**:
```javascript
// Global state tracks all data
let statsData = null;
let categoriesData = [];

// Updates modify state first, then DOM
updateTransactionInDOM() {
    // 1. Update categoriesData
    // 2. Update statsData
    // 3. Rebuild affected DOM
}
```

**Incremental DOM Updates**:
```javascript
// Instead of full reload:
updateTotalsInDOM()        // Only updates header
updateCategoriesInDOM()    // Only updates categories grid

// Avoids:
// - Flickering
// - Lost scroll position
// - Lost expansion state
// - Unnecessary network requests
```

---

### Toast Notifications

**Implementation**:

```javascript
function showToast(message) {
    // Create element
    const toast = document.createElement('div');
    toast.className = 'toast';
    toast.textContent = message;
    document.body.appendChild(toast);

    // Animate in
    setTimeout(() => toast.classList.add('show'), 10);

    // Auto-hide after 3s
    setTimeout(() => {
        toast.classList.add('hide');
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}
```

**CSS Animation**:
```css
.toast {
    position: fixed;
    top: 20px;
    right: 20px;
    opacity: 0;
    transform: translateX(400px);
    transition: all 0.3s ease;
}

.toast.show {
    opacity: 1;
    transform: translateX(0);
}
```

**Usage**:
```javascript
showToast('Transaction updated successfully');
showToast('Transaction deleted successfully');
```

---

### Billing Cycle Recalculation on Edit

**Why Important**: Editing transaction date may change billing cycle

**Implementation**:

```go
func updateTransactionHandler(db *DatabaseClient, id int64) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var tx Transaction
        json.NewDecoder(r.Body).Decode(&tx)

        // Recalculate billing cycle from new date
        tx.BillingCycle = calculateBillingCycle(tx.Date)

        db.UpdateTransaction(id, tx)
    }
}
```

**Example**:
```
Original: date=2026-01-15, cycle=Dec 2025
Edit to:  date=2026-01-25, cycle=Jan 2026 (auto-recalculated)
```

**Result**: Transaction moves to correct cycle in stats

---

## Development Workflow

### Local Setup

```bash
# 1. Clone repository
git clone <repo-url>
cd transaction-tracker

# 2. Install Go dependencies
go mod download

# 3. Configure environment
cp .env.example .env
nano .env  # Add your OPENAI_API_KEY

# 4. Run
go run *.go

# 5. Test
curl http://localhost:8080/health
```

### Development Cycle

```bash
# 1. Make code changes
nano main.go

# 2. Run (auto-recompiles)
go run *.go

# 3. Test endpoint
curl -X POST http://localhost:8080/transaction \
  -H "Content-Type: application/json" \
  -d '{"text": "AED 50 at test merchant"}'

# 4. View logs in terminal

# 5. Repeat
```

### Testing API

**Health Check**:
```bash
curl http://localhost:8080/health
```

**Log Transaction**:
```bash
curl -X POST http://localhost:8080/transaction \
  -H "Content-Type: application/json" \
  -d '{
    "text": "AED 50 at Starbucks. AED 30 at Careem"
  }'
```

**Get Stats**:
```bash
curl http://localhost:8080/stats | jq
```

**Update Transaction**:
```bash
curl -X PUT http://localhost:8080/transaction/1 \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated description",
    "amount": 60.00,
    "date": "2026-01-25",
    "category": "Food & Dining"
  }'
```

**Delete Transaction**:
```bash
curl -X DELETE http://localhost:8080/transaction/1
```

**View Dashboard**:
```bash
open http://localhost:8080
```

### Building

**Development Build**:
```bash
go build -o transaction-tracker
./transaction-tracker
```

**Production Build** (static binary):
```bash
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
  -a -installsuffix cgo \
  -o transaction-tracker .
```

**Why CGO_ENABLED=1**: SQLite requires C bindings

**Cross-Platform Builds**:
```bash
# macOS
GOOS=darwin GOARCH=amd64 go build -o transaction-tracker-mac

# Linux
GOOS=linux GOARCH=amd64 go build -o transaction-tracker-linux

# Windows
GOOS=windows GOARCH=amd64 go build -o transaction-tracker.exe
```

### Docker Development

**Build**:
```bash
docker build -t transaction-tracker .
```

**Run with Volume**:
```bash
docker run -p 8080:8080 \
  --env-file .env \
  -v $(pwd)/data:/data \
  transaction-tracker
```

**View Logs**:
```bash
docker logs -f <container-id>
```

**Shell Access**:
```bash
docker exec -it <container-id> /bin/sh
```

**Cleanup**:
```bash
docker stop <container-id>
docker rm <container-id>
docker rmi transaction-tracker
```

### Database Management

**View Database**:
```bash
sqlite3 transactions.db

# SQLite shell:
.tables
.schema transactions
SELECT * FROM transactions LIMIT 10;
SELECT COUNT(*) FROM transactions;
.quit
```

**Backup Database**:
```bash
cp transactions.db transactions-backup-$(date +%Y%m%d).db
```

**Reset Database**:
```bash
rm transactions.db
# Restart app (will recreate with migrations)
```

### Debugging

**Enable Verbose Logging**:
```go
// Already enabled by default
log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
```

**Debug OpenAI Responses**:
```go
// In openai.go ParseTransactions()
log.Printf("[OpenAI] Raw response: %s", content)
```

**Debug Database Queries**:
```go
// In database.go
log.Printf("[Database] Query: %s, Args: %v", query, args)
```

**Check Database File**:
```bash
ls -lh transactions.db
# Should show file size growing as transactions added
```

### Deployment

**Deploy to Fly.io**:
```bash
# First time
fly launch
fly secrets set OPENAI_API_KEY=...
fly volumes create transactions_data --size 1
fly deploy

# Updates
fly deploy

# Rollback
fly releases
fly releases rollback <version>
```

**Monitor Production**:
```bash
# Logs
fly logs -a transaction-tracker

# Status
fly status

# SSH
fly ssh console

# Metrics
fly dashboard
```

---

## Summary

This document provides a complete technical overview of the Transaction Tracker system:

1. **Purpose**: AI-powered expense tracking with automatic SMS parsing
2. **Architecture**: Go HTTP server → OpenAI API + SQLite database
3. **Key Features**:
   - Multi-transaction parsing
   - Currency conversion
   - Smart categorization
   - Billing cycle tracking
   - Real-time dashboard
   - Edit/delete with DOM updates
4. **Tech Stack**: Go stdlib + SQLite + OpenAI GPT-4o-mini + Vanilla JS
5. **Deployment**: Docker + Fly.io (free tier)
6. **Recent Updates**: Full CRUD operations, toast notifications, instant UI updates

**For LLMs**: This document provides all context needed to:
- Understand the entire system architecture
- Debug issues
- Add new features
- Modify existing functionality
- Deploy and maintain the application

**For Developers**: Use this as:
- Onboarding guide
- Reference documentation
- Architecture decision record
- Development handbook
