# Database Migration Guide - Notion to Database

This guide outlines the steps to migrate from Notion to a proper database backend.

## Recommended Database: Supabase (PostgreSQL)

**Why Supabase?**
- ✅ Free tier: 500MB database (sufficient for years of transactions)
- ✅ PostgreSQL (powerful, reliable, open-source)
- ✅ Auto-generated REST API
- ✅ Built-in authentication (if needed later)
- ✅ Web UI for managing data
- ✅ Easy deployment
- ✅ Can use standard PostgreSQL drivers in Go

**Alternative Options:**
- **SQLite** - Simplest, single file, no external service needed (great for Fly.io)
- **Railway PostgreSQL** - Free tier, auto-provisioned
- **MongoDB Atlas** - NoSQL option, 512MB free

---

## Migration Steps

### Step 1: Choose Your Database

**Option A: Supabase (Recommended for cloud)**
```bash
# 1. Sign up at https://supabase.com
# 2. Create a new project
# 3. Get connection string from project settings
```

**Option B: SQLite (Recommended for simplicity)**
- No external service needed
- Perfect for Fly.io deployment
- Single file database
- Good for < 100K transactions

---

### Step 2: Design Database Schema

**Transactions Table:**
```sql
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    description VARCHAR(255) NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    transaction_date DATE NOT NULL,
    category VARCHAR(50) NOT NULL,
    confidence INTEGER,
    billing_cycle VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),

    -- Optional: Add hash for duplicate detection
    transaction_hash VARCHAR(64) UNIQUE,

    -- Indexes for performance
    INDEX idx_billing_cycle (billing_cycle),
    INDEX idx_transaction_date (transaction_date),
    INDEX idx_category (category)
);
```

**Optional: Categories Table (normalized)**
```sql
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    emoji VARCHAR(10) NOT NULL
);

-- Add foreign key to transactions
ALTER TABLE transactions
ADD COLUMN category_id INTEGER REFERENCES categories(id);
```

---

### Step 3: Install Database Driver

**For PostgreSQL (Supabase):**
```bash
go get github.com/lib/pq
```

Update `go.mod`:
```go
module transaction-tracker

go 1.22

require github.com/lib/pq v1.10.9
```

**For SQLite:**
```bash
go get github.com/mattn/go-sqlite3
```

---

### Step 4: Create Database Client

**Create `database.go`:**
```go
package main

import (
    "database/sql"
    "fmt"
    "time"
    _ "github.com/lib/pq" // PostgreSQL driver
)

type DatabaseClient struct {
    db *sql.DB
}

func NewDatabaseClient(connectionString string) (*DatabaseClient, error) {
    db, err := sql.Open("postgres", connectionString)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    // Test connection
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    // Set connection pool settings
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)

    return &DatabaseClient{db: db}, nil
}

func (c *DatabaseClient) SaveTransaction(tx Transaction) error {
    query := `
        INSERT INTO transactions
        (description, amount, transaction_date, category, confidence, billing_cycle, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `

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

    return err
}

func (c *DatabaseClient) GetStats() (*StatsResponse, error) {
    currentCycle := calculateBillingCycle(time.Now().Format("2006-01-02"))

    // Query total and count
    var total float64
    var count int

    err := c.db.QueryRow(`
        SELECT COALESCE(SUM(amount), 0), COUNT(*)
        FROM transactions
        WHERE billing_cycle = $1 AND category != 'Income/Transfer'
    `, currentCycle).Scan(&total, &count)

    if err != nil {
        return nil, err
    }

    // Query category breakdown
    rows, err := c.db.Query(`
        SELECT category, SUM(amount) as total, COUNT(*) as count
        FROM transactions
        WHERE billing_cycle = $1
        GROUP BY category
        ORDER BY total DESC
    `, currentCycle)

    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var categories []CategoryStats
    for rows.Next() {
        var cat CategoryStats
        if err := rows.Scan(&cat.Category, &cat.Total, &cat.Count); err != nil {
            return nil, err
        }
        cat.Emoji = getCategoryEmoji(cat.Category)
        categories = append(categories, cat)
    }

    // Query last transaction
    var lastTx TransactionSummary
    err = c.db.QueryRow(`
        SELECT description, amount, transaction_date
        FROM transactions
        WHERE billing_cycle = $1
        ORDER BY transaction_date DESC, created_at DESC
        LIMIT 1
    `, currentCycle).Scan(&lastTx.Description, &lastTx.Amount, &lastTx.Date)

    var lastTransaction *TransactionSummary
    if err == nil {
        lastTransaction = &lastTx
    }

    // Build response (similar to Notion client)
    // ... (same message formatting logic)

    return &StatsResponse{
        Success:         true,
        Cycle:           currentCycle,
        Total:           total,
        Count:           count,
        Categories:      categories,
        LastTransaction: lastTransaction,
    }, nil
}

func (c *DatabaseClient) Close() error {
    return c.db.Close()
}
```

---

### Step 5: Add Duplicate Detection

**Add method to check for duplicates:**
```go
func (c *DatabaseClient) CheckDuplicate(tx Transaction) (bool, error) {
    var exists bool

    // Check for transaction with same description, amount, and date
    err := c.db.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM transactions
            WHERE description = $1
            AND amount = $2
            AND transaction_date = $3
        )
    `, tx.Description, tx.Amount, tx.Date).Scan(&exists)

    return exists, err
}

// Or use hash-based approach
func (c *DatabaseClient) GetTransactionHash(tx Transaction) string {
    data := fmt.Sprintf("%s|%.2f|%s", tx.Description, tx.Amount, tx.Date)
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}
```

---

### Step 6: Update Configuration

**Update `.env.example` and `.env`:**
```bash
# Database Configuration (choose one)
DATABASE_URL=postgresql://user:password@host:port/database
# or for SQLite
DATABASE_PATH=./transactions.db

# Keep OpenAI (still needed for parsing)
OPENAI_API_KEY=sk-proj-xxxxxxxxxxxxx

# Remove Notion variables
# NOTION_API_KEY=...
# NOTION_DATABASE_ID=...
```

**Update `main.go` config:**
```go
type Config struct {
    OpenAIKey   string
    DatabaseURL string // Replace Notion fields
    Port        string
}

func loadConfig() (*Config, error) {
    config := &Config{
        OpenAIKey:   os.Getenv("OPENAI_API_KEY"),
        DatabaseURL: os.Getenv("DATABASE_URL"),
        Port:        os.Getenv("PORT"),
    }

    if config.DatabaseURL == "" {
        // Default to SQLite for local development
        config.DatabaseURL = "file:./transactions.db?cache=shared&mode=rwc"
    }

    // ... validation

    return config, nil
}
```

---

### Step 7: Update Main Application

**Replace Notion client with Database client in `main.go`:**
```go
func main() {
    config, err := loadConfig()
    if err != nil {
        log.Fatalf("Configuration error: %v", err)
    }

    openAIClient := NewOpenAIClient(config.OpenAIKey)

    // Replace NotionClient with DatabaseClient
    dbClient, err := NewDatabaseClient(config.DatabaseURL)
    if err != nil {
        log.Fatalf("Database connection error: %v", err)
    }
    defer dbClient.Close()

    // Routes remain the same - just pass dbClient instead of notionClient
    http.HandleFunc("/health", healthHandler)
    http.HandleFunc("/transaction", transactionHandler(openAIClient, dbClient))
    http.HandleFunc("/stats", statsHandler(dbClient))
    http.HandleFunc("/", dashboardHandler)

    // ... rest of main
}
```

**Update handler signatures:**
```go
// Change from NotionClient to DatabaseClient (or create interface)
func transactionHandler(openAI *OpenAIClient, db *DatabaseClient) http.HandlerFunc {
    // Logic remains the same
}

func statsHandler(db *DatabaseClient) http.HandlerFunc {
    // Logic remains the same
}
```

---

### Step 8: Add Database Migrations

**Create `migrations.go`:**
```go
package main

func (c *DatabaseClient) RunMigrations() error {
    migrations := []string{
        `CREATE TABLE IF NOT EXISTS transactions (
            id SERIAL PRIMARY KEY,
            description VARCHAR(255) NOT NULL,
            amount DECIMAL(10, 2) NOT NULL,
            transaction_date DATE NOT NULL,
            category VARCHAR(50) NOT NULL,
            confidence INTEGER,
            billing_cycle VARCHAR(20) NOT NULL,
            created_at TIMESTAMP DEFAULT NOW()
        )`,
        `CREATE INDEX IF NOT EXISTS idx_billing_cycle ON transactions(billing_cycle)`,
        `CREATE INDEX IF NOT EXISTS idx_transaction_date ON transactions(transaction_date)`,
    }

    for _, migration := range migrations {
        if _, err := c.db.Exec(migration); err != nil {
            return fmt.Errorf("migration failed: %w", err)
        }
    }

    return nil
}
```

**Run migrations on startup:**
```go
func main() {
    // ... after creating dbClient

    if err := dbClient.RunMigrations(); err != nil {
        log.Fatalf("Failed to run migrations: %v", err)
    }

    // ... continue with server setup
}
```

---

### Step 9: Optional - Create Interface for Flexibility

**Create `storage.go`:**
```go
package main

// StorageClient interface allows swapping between Notion, DB, etc.
type StorageClient interface {
    SaveTransaction(tx Transaction) error
    GetStats() (*StatsResponse, error)
    CheckDuplicate(tx Transaction) (bool, error)
}

// Both NotionClient and DatabaseClient implement this interface
// Then handlers use StorageClient instead of specific type
func transactionHandler(openAI *OpenAIClient, storage StorageClient) http.HandlerFunc {
    // Works with any storage backend
}
```

---

### Step 10: Deploy with Database

**For Fly.io with Supabase:**
```bash
# Set database URL as secret
fly secrets set DATABASE_URL="postgresql://user:pass@host:5432/db"

# Deploy
fly deploy
```

**For Fly.io with SQLite:**
```toml
# Add to fly.toml
[mounts]
  source = "transactions_data"
  destination = "/data"
```

Update code to use:
```go
DATABASE_PATH=/data/transactions.db
```

---

## Migration Checklist

- [ ] Choose database (Supabase/SQLite/other)
- [ ] Design schema (SQL table structure)
- [ ] Install Go database driver
- [ ] Create `database.go` with client implementation
- [ ] Implement `SaveTransaction()` method
- [ ] Implement `GetStats()` method
- [ ] Add duplicate detection
- [ ] Update configuration (environment variables)
- [ ] Replace NotionClient with DatabaseClient in main.go
- [ ] Add database migrations
- [ ] Test locally with SQLite
- [ ] Update deployment configuration
- [ ] Deploy and test in production
- [ ] Optional: Export existing Notion data and import to database

---

## Estimated Time

- **With SQLite (simplest):** 2-3 hours
- **With Supabase/PostgreSQL:** 3-4 hours
- **With full migration from Notion:** Add 1-2 hours

---

## Benefits After Migration

✅ **Better Performance** - Direct database queries vs API calls
✅ **No API Rate Limits** - No external service restrictions
✅ **Duplicate Detection** - Easy to implement with SQL
✅ **Advanced Queries** - Complex filtering, aggregations, analytics
✅ **Data Ownership** - Full control over your data
✅ **Offline Capability** - SQLite works without internet
✅ **Easier Backups** - Simple database dumps

---

## When to Migrate?

**Migrate if:**
- You want duplicate detection
- You need better performance
- You want advanced analytics/reporting
- You don't like Notion's UI
- You want full data ownership

**Stay with Notion if:**
- Current setup works well
- You like Notion's UI for viewing data
- You want minimal maintenance
- Integration is already working

---

## Recommendation

Given your requirements (duplicate detection, better dashboard), I'd suggest:

**Short-term:** Fix timeout issues, keep Notion for now
**Medium-term:** Switch to **SQLite** (simplest migration, no external dependencies)
**Long-term:** Consider Supabase if you need cloud features or want to build more advanced features

Would you like me to implement the SQLite migration? It's the quickest path and requires no external services.
