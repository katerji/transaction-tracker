package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DatabaseClient struct {
	db *sql.DB
}

func NewDatabaseClient(dbPath string) (*DatabaseClient, error) {
	log.Printf("[Database] Connecting to database at: %s", dbPath)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite only supports one writer at a time
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	log.Printf("[Database] Connection established successfully")

	client := &DatabaseClient{db: db}

	// Run migrations to create tables
	log.Printf("[Database] Running migrations...")
	if err := client.runMigrations(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	log.Printf("[Database] Migrations completed successfully")

	return client, nil
}

func (c *DatabaseClient) runMigrations() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			description TEXT NOT NULL,
			amount REAL NOT NULL,
			transaction_date TEXT NOT NULL,
			category TEXT NOT NULL,
			confidence INTEGER,
			billing_cycle TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(description, amount, transaction_date)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_billing_cycle ON transactions(billing_cycle)`,
		`CREATE INDEX IF NOT EXISTS idx_transaction_date ON transactions(transaction_date)`,
		`CREATE INDEX IF NOT EXISTS idx_category ON transactions(category)`,
	}

	for _, migration := range migrations {
		if _, err := c.db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

func (c *DatabaseClient) SaveTransaction(tx Transaction) (int64, error) {
	query := `
		INSERT INTO transactions
		(description, amount, transaction_date, category, confidence, billing_cycle, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	log.Printf("[Database] Saving transaction: %s (%.2f AED)", tx.Description, tx.Amount)

	result, err := c.db.Exec(
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
		log.Printf("[Database] Failed to save transaction: %v", err)
		return 0, fmt.Errorf("failed to save transaction: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("[Database] Failed to get last insert ID: %v", err)
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	log.Printf("[Database] Transaction saved successfully with ID %d", id)
	return id, nil
}

func (c *DatabaseClient) GetStats() (*StatsResponse, error) {
	currentCycle := calculateBillingCycle(time.Now().Format("2006-01-02"))
	log.Printf("[Database] Fetching stats for billing cycle: %s", currentCycle)

	// Query total and count
	var total float64
	var count int

	err := c.db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0), COUNT(*)
		FROM transactions
		WHERE billing_cycle = ? AND category != 'Income/Transfer'
	`, currentCycle).Scan(&total, &count)

	if err != nil {
		log.Printf("[Database] Failed to get totals: %v", err)
		return nil, fmt.Errorf("failed to get totals: %w", err)
	}

	log.Printf("[Database] Found %d transactions, total: %.2f AED", count, total)

	// Handle empty state
	if count == 0 {
		message := fmt.Sprintf("📊 Billing Cycle: %s\n\nNo transactions found for this cycle yet.\n\nStart logging your expenses!", currentCycle)
		return &StatsResponse{
			Success:    true,
			Message:    message,
			Cycle:      currentCycle,
			Total:      0,
			Count:      0,
			Categories: []CategoryStats{},
		}, nil
	}

	// Query category breakdown
	rows, err := c.db.Query(`
		SELECT category, SUM(amount) as total, COUNT(*) as count
		FROM transactions
		WHERE billing_cycle = ?
		GROUP BY category
		ORDER BY total DESC
	`, currentCycle)

	if err != nil {
		return nil, fmt.Errorf("failed to get category stats: %w", err)
	}
	defer rows.Close()

	var categories []CategoryStats
	for rows.Next() {
		var cat CategoryStats
		if err := rows.Scan(&cat.Category, &cat.Total, &cat.Count); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		cat.Emoji = getCategoryEmoji(cat.Category)
		categories = append(categories, cat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating categories: %w", err)
	}

	// Fetch transactions for each category
	for i := range categories {
		txRows, err := c.db.Query(`
			SELECT id, description, amount, transaction_date, category, confidence, billing_cycle, created_at
			FROM transactions
			WHERE billing_cycle = ? AND category = ?
			ORDER BY transaction_date DESC, created_at DESC
		`, currentCycle, categories[i].Category)

		if err != nil {
			return nil, fmt.Errorf("failed to get transactions for category %s: %w", categories[i].Category, err)
		}

		var transactions []Transaction
		for txRows.Next() {
			var tx Transaction
			if err := txRows.Scan(&tx.ID, &tx.Description, &tx.Amount, &tx.Date, &tx.Category, &tx.Confidence, &tx.BillingCycle, &tx.Timestamp); err != nil {
				txRows.Close()
				return nil, fmt.Errorf("failed to scan transaction: %w", err)
			}
			transactions = append(transactions, tx)
		}
		txRows.Close()

		categories[i].Transactions = transactions
	}

	// Sort categories by total (descending)
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Total > categories[j].Total
	})

	// Query all transactions sorted by date descending for the flat list
	allTxRows, err := c.db.Query(`
		SELECT id, description, amount, transaction_date, category, confidence, billing_cycle, created_at
		FROM transactions
		WHERE billing_cycle = ?
		ORDER BY transaction_date DESC, created_at DESC
	`, currentCycle)

	if err != nil {
		return nil, fmt.Errorf("failed to get all transactions: %w", err)
	}
	defer allTxRows.Close()

	var allTransactions []Transaction
	for allTxRows.Next() {
		var tx Transaction
		if err := allTxRows.Scan(&tx.ID, &tx.Description, &tx.Amount, &tx.Date, &tx.Category, &tx.Confidence, &tx.BillingCycle, &tx.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan all transaction: %w", err)
		}
		allTransactions = append(allTransactions, tx)
	}

	if err := allTxRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating all transactions: %w", err)
	}

	// Query last transaction
	var lastTx TransactionSummary
	err = c.db.QueryRow(`
		SELECT description, amount, transaction_date
		FROM transactions
		WHERE billing_cycle = ?
		ORDER BY transaction_date DESC, created_at DESC
		LIMIT 1
	`, currentCycle).Scan(&lastTx.Description, &lastTx.Amount, &lastTx.Date)

	var lastTransaction *TransactionSummary
	if err == nil {
		lastTransaction = &lastTx
	} else if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get last transaction: %w", err)
	}

	// Build message
	message := fmt.Sprintf("📊 Billing Cycle: %s (23rd - 22nd)\n", currentCycle)
	message += "━━━━━━━━━━━━━━━\n"
	message += fmt.Sprintf("💰 Total Spent: %.2f AED\n\n", total)
	message += "By Category:\n"

	for _, cat := range categories {
		message += fmt.Sprintf("%s %s: %.2f AED (%d transaction%s)\n",
			cat.Emoji, cat.Category, cat.Total, cat.Count, pluralize(cat.Count))
	}

	if lastTransaction != nil {
		txDate, _ := time.Parse("2006-01-02", lastTransaction.Date)
		today := time.Now()
		dateStr := "today"
		if txDate.Format("2006-01-02") != today.Format("2006-01-02") {
			dateStr = txDate.Format("Jan 2")
		}

		message += "\n━━━━━━━━━━━━━━━\n"
		message += "🕐 Last transaction:\n"
		message += fmt.Sprintf("   %s - %.2f AED (%s)", lastTransaction.Description, lastTransaction.Amount, dateStr)
	}

	return &StatsResponse{
		Success:         true,
		Message:         message,
		Cycle:           currentCycle,
		Total:           total,
		Count:           count,
		Categories:      categories,
		LastTransaction: lastTransaction,
		AllTransactions: allTransactions,
	}, nil
}

func (c *DatabaseClient) GetAllTransactionsGroupedByCycle() ([]Transaction, error) {
	rows, err := c.db.Query(`
		SELECT id, description, amount, transaction_date, category, confidence, billing_cycle, created_at
		FROM transactions
		ORDER BY transaction_date DESC, created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(&tx.ID, &tx.Description, &tx.Amount, &tx.Date, &tx.Category, &tx.Confidence, &tx.BillingCycle, &tx.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

func (c *DatabaseClient) UpdateTransaction(id int64, tx Transaction) error {
	query := `
		UPDATE transactions
		SET description = ?, amount = ?, transaction_date = ?, category = ?, billing_cycle = ?
		WHERE id = ?
	`

	log.Printf("[Database] Updating transaction ID %d: %s (%.2f AED)", id, tx.Description, tx.Amount)

	result, err := c.db.Exec(
		query,
		tx.Description,
		tx.Amount,
		tx.Date,
		tx.Category,
		tx.BillingCycle,
		id,
	)

	if err != nil {
		log.Printf("[Database] Failed to update transaction: %v", err)
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("transaction not found")
	}

	log.Printf("[Database] Transaction updated successfully")
	return nil
}

func (c *DatabaseClient) DeleteTransaction(id int64) error {
	query := `DELETE FROM transactions WHERE id = ?`

	log.Printf("[Database] Deleting transaction ID %d", id)

	result, err := c.db.Exec(query, id)
	if err != nil {
		log.Printf("[Database] Failed to delete transaction: %v", err)
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("transaction not found")
	}

	log.Printf("[Database] Transaction deleted successfully")
	return nil
}

func (c *DatabaseClient) Close() error {
	log.Printf("[Database] Closing database connection")
	return c.db.Close()
}
