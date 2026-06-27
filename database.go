package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DatabaseClient struct {
	db *sql.DB
}

type MerchantRule struct {
	ID        int64  `json:"id"`
	Keyword   string `json:"keyword"`
	Category  string `json:"category"`
	Priority  int    `json:"priority"`
	CreatedAt string `json:"createdAt"`
}

type Category struct {
	ID                int64    `json:"id"`
	Name              string   `json:"name"`
	Emoji             string   `json:"emoji"`
	ExcludeFromTotals bool     `json:"excludeFromTotals"`
	CreatedAt         string   `json:"createdAt"`
	Type              string   `json:"type"`
	BudgetAmount      *float64 `json:"budgetAmount"`
	// Tracking is how spend is measured for the category:
	//   "actual"    — real transactions accumulate against the budget (default)
	//   "allocated" — funded by a per-cycle tick-off ("set aside"), counted at budget
	Tracking string `json:"tracking"`
}

func floatPtr(v float64) *float64 { return &v }

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

	if err := c.addColumnIfNotExists("transactions", "source TEXT NOT NULL DEFAULT 'openai'"); err != nil {
		return fmt.Errorf("failed to add source column: %w", err)
	}

	categoriesMigrations := []string{
		`CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			emoji TEXT NOT NULL DEFAULT '',
			exclude_from_totals INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_categories_name ON categories(name)`,
	}

	for _, migration := range categoriesMigrations {
		if _, err := c.db.Exec(migration); err != nil {
			return fmt.Errorf("categories migration failed: %w", err)
		}
	}

	if err := c.addColumnIfNotExists("categories", "type TEXT NOT NULL DEFAULT 'wants'"); err != nil {
		return fmt.Errorf("failed to add type column: %w", err)
	}
	if _, err := c.db.Exec("UPDATE categories SET type = 'wants' WHERE type = 'discretionary'"); err != nil {
		return fmt.Errorf("failed to migrate discretionary→wants: %w", err)
	}
	if err := c.addColumnIfNotExists("categories", "budget_amount REAL"); err != nil {
		return fmt.Errorf("failed to add budget_amount column: %w", err)
	}
	if err := c.addColumnIfNotExists("categories", "tracking TEXT NOT NULL DEFAULT 'actual'"); err != nil {
		return fmt.Errorf("failed to add tracking column: %w", err)
	}

	// cycle_funding records a "set aside" tick for an allocated category in a
	// given billing cycle. One row = funded this cycle; amount snapshots the
	// category's budget at tick time so later budget edits don't rewrite history.
	if _, err := c.db.Exec(`CREATE TABLE IF NOT EXISTS cycle_funding (
		cycle TEXT NOT NULL,
		category_id INTEGER NOT NULL,
		amount REAL NOT NULL,
		funded_at TEXT NOT NULL,
		PRIMARY KEY (cycle, category_id)
	)`); err != nil {
		return fmt.Errorf("cycle_funding migration failed: %w", err)
	}

	// settings is a tiny key/value store (currently just monthly_salary).
	if _, err := c.db.Exec(`CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("settings migration failed: %w", err)
	}

	if err := c.seedCategories(); err != nil {
		return fmt.Errorf("failed to seed categories: %w", err)
	}

	merchantRulesMigrations := []string{
		`CREATE TABLE IF NOT EXISTS merchant_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			keyword TEXT NOT NULL,
			category TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_rules_priority ON merchant_rules(priority DESC, id ASC)`,
	}

	for _, migration := range merchantRulesMigrations {
		if _, err := c.db.Exec(migration); err != nil {
			return fmt.Errorf("merchant rules migration failed: %w", err)
		}
	}

	if err := c.seedMerchantRules(); err != nil {
		return fmt.Errorf("failed to seed merchant rules: %w", err)
	}

	if err := c.runDataMigrations(); err != nil {
		return fmt.Errorf("failed to run data migrations: %w", err)
	}

	if err := c.runBudgetingSetup(); err != nil {
		return fmt.Errorf("failed to run budgeting setup: %w", err)
	}

	return nil
}

// runBudgetingSetup applies the salary-budget model (fixed / spending / goals,
// per-category tracking + targets) ONCE, guarded by a settings flag, so that
// afterwards the user's bucket moves and budget edits persist across reboots.
// It converges both fresh and existing databases onto the June 2026 budget.
func (c *DatabaseClient) runBudgetingSetup() error {
	done, _ := c.GetSetting("budgeting_setup_v1")
	if done == "1" {
		return nil
	}

	// Canonical budget lines. tracking: "allocated" = set-aside tick-off,
	// "actual" = real transactions. Goals are a separate, non-"spent" bucket.
	cats := []struct {
		name     string
		emoji    string
		catType  string // fixed | wants | goal | other
		tracking string
		budget   *float64
	}{
		// Fixed — set aside
		{"Rent", "🏠", "fixed", "allocated", floatPtr(9000)},
		{"Car loan", "🚙", "fixed", "allocated", floatPtr(1742)},
		{"Family support", "👨‍👩‍👧", "fixed", "allocated", floatPtr(1469)},
		{"Therapy", "🧠", "fixed", "allocated", floatPtr(1762)},
		{"Wife Electrolysis", "💆", "fixed", "allocated", floatPtr(1000)},
		{"Cleaning", "🧹", "fixed", "allocated", floatPtr(400)},
		{"Phone", "📞", "fixed", "allocated", floatPtr(160)},
		{"Car insurance", "🛡️", "fixed", "allocated", floatPtr(117)},
		{"Car registration", "📋", "fixed", "allocated", floatPtr(33)},
		// Fixed — actual card charges
		{"DEWA", "⚡", "fixed", "actual", floatPtr(750)},
		{"E& Bill", "📡", "fixed", "actual", floatPtr(450)},
		{"Subscriptions", "📱", "fixed", "actual", floatPtr(579)},
		// Spending (stored as "wants") — all actual
		{"Groceries", "🛒", "wants", "actual", floatPtr(2000)},
		{"Dining Out & Delivery", "🍔", "wants", "actual", floatPtr(2000)},
		{"Transport", "🚗", "wants", "actual", floatPtr(1100)},
		{"Shopping & Gifts", "🛍️", "wants", "actual", floatPtr(1500)},
		{"Healthcare", "💊", "wants", "actual", floatPtr(500)},
		{"Beauty", "💅", "wants", "actual", floatPtr(400)},
		{"Entertainment & Going Out", "🎬", "wants", "actual", floatPtr(1000)},
		{"Misc / Buffer", "🗂️", "wants", "actual", floatPtr(1288)},
		// Goals — set aside
		{"Investment", "📈", "goal", "allocated", floatPtr(2500)},
		{"Savings", "🏦", "goal", "allocated", floatPtr(2000)},
		{"Emergency fund", "🆘", "goal", "allocated", floatPtr(750)},
	}

	now := time.Now().Format(time.RFC3339)
	for _, cat := range cats {
		if _, err := c.db.Exec(
			"INSERT OR IGNORE INTO categories (name, emoji, exclude_from_totals, type, budget_amount, tracking, created_at) VALUES (?, ?, 0, ?, ?, ?, ?)",
			cat.name, cat.emoji, cat.catType, cat.budget, cat.tracking, now,
		); err != nil {
			return fmt.Errorf("insert budgeting category %s: %w", cat.name, err)
		}
		if _, err := c.db.Exec(
			"UPDATE categories SET type=?, tracking=?, budget_amount=? WHERE name=?",
			cat.catType, cat.tracking, cat.budget, cat.name,
		); err != nil {
			return fmt.Errorf("update budgeting category %s: %w", cat.name, err)
		}
	}

	// Default the monthly salary if unset.
	if _, err := c.db.Exec(
		"INSERT OR IGNORE INTO settings (key, value) VALUES ('monthly_salary', '32500')",
	); err != nil {
		return fmt.Errorf("seed salary: %w", err)
	}

	if err := c.SetSetting("budgeting_setup_v1", "1"); err != nil {
		return fmt.Errorf("mark budgeting setup done: %w", err)
	}
	return nil
}

func (c *DatabaseClient) addColumnIfNotExists(table, colDef string) error {
	_, err := c.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, colDef))
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		return err
	}
	return nil
}

func (c *DatabaseClient) runDataMigrations() error {
	// 2a. Category renames — cascade to transactions + merchant_rules
	renames := [][2]string{
		{"Dining Out", "Dining Out & Delivery"},
		{"Health", "Healthcare"},
		{"Entertainment", "Entertainment & Going Out"},
		{"Shopping", "Shopping & Gifts"},
	}
	for _, pair := range renames {
		tx, err := c.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin rename tx: %w", err)
		}
		if _, err := tx.Exec("UPDATE categories SET name=? WHERE name=?", pair[1], pair[0]); err != nil {
			tx.Rollback()
			return fmt.Errorf("rename category %s: %w", pair[0], err)
		}
		if _, err := tx.Exec("UPDATE transactions SET category=? WHERE category=?", pair[1], pair[0]); err != nil {
			tx.Rollback()
			return fmt.Errorf("cascade rename transactions %s: %w", pair[0], err)
		}
		if _, err := tx.Exec("UPDATE merchant_rules SET category=? WHERE category=?", pair[1], pair[0]); err != nil {
			tx.Rollback()
			return fmt.Errorf("cascade rename rules %s: %w", pair[0], err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit rename %s: %w", pair[0], err)
		}
	}

	// Note: category type/budget/tracking assignment is handled once by
	// runBudgetingSetup() (guarded), NOT here — so user bucket moves persist.

	// 2d. Delete stale merchant rules that pointed to Bills & Utilities
	if _, err := c.db.Exec(
		"DELETE FROM merchant_rules WHERE keyword IN ('dewa', 'etisalat', 'du telecom', 'addc') AND category = 'Bills & Utilities'",
	); err != nil {
		return fmt.Errorf("delete stale rules: %w", err)
	}

	// 2e. Delete Travel category if unused
	var travelCount int
	c.db.QueryRow("SELECT COUNT(*) FROM transactions WHERE category = 'Travel'").Scan(&travelCount)
	if travelCount == 0 {
		c.db.Exec("DELETE FROM categories WHERE name = 'Travel'")
	}

	return nil
}

func (c *DatabaseClient) SaveTransaction(tx Transaction) (int64, error) {
	query := `
		INSERT INTO transactions
		(description, amount, transaction_date, category, confidence, billing_cycle, created_at, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
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
		tx.Source,
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

func (c *DatabaseClient) GetStats(cycle string) (*StatsResponse, error) {
	currentCycle := cycle
	if currentCycle == "" {
		currentCycle = calculateBillingCycle(time.Now().Format("2006-01-02"))
	}
	log.Printf("[Database] Fetching stats for billing cycle: %s", currentCycle)

	availableCycles := selectableCycles()

	// Fetch all categories and build lookup maps
	allCats, err := c.GetAllCategories()
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	emojiMap := make(map[string]string, len(allCats))
	typeMap := make(map[string]string, len(allCats))
	trackingMap := make(map[string]string, len(allCats))
	excludeMap := make(map[string]bool, len(allCats))
	var fixedBudget, wantsBudget, goalsBudget float64
	for _, cat := range allCats {
		emojiMap[cat.Name] = cat.Emoji
		typeMap[cat.Name] = cat.Type
		trackingMap[cat.Name] = cat.Tracking
		excludeMap[cat.Name] = cat.ExcludeFromTotals
		if cat.BudgetAmount != nil {
			switch cat.Type {
			case "fixed":
				fixedBudget += *cat.BudgetAmount
			case "wants":
				wantsBudget += *cat.BudgetAmount
			case "goal":
				goalsBudget += *cat.BudgetAmount
			}
		}
	}

	// Per-cycle "set aside" funding for allocated categories.
	funded, err := c.GetFunding(currentCycle)
	if err != nil {
		return nil, fmt.Errorf("failed to get funding: %w", err)
	}
	fundedIDs := make([]int64, 0, len(funded))
	var fixedAllocated, goalsFunded float64
	for _, cat := range allCats {
		amt, ok := funded[cat.ID]
		if !ok {
			continue
		}
		fundedIDs = append(fundedIDs, cat.ID)
		switch cat.Type {
		case "fixed":
			fixedAllocated += amt
		case "goal":
			goalsFunded += amt
		}
	}

	salary := c.GetSalary()

	// Query total and count
	var total float64
	var count int

	err = c.db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0), COUNT(*)
		FROM transactions
		WHERE billing_cycle = ? AND category NOT IN (SELECT name FROM categories WHERE exclude_from_totals = 1)
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
			Success:             true,
			Message:             message,
			Cycle:               currentCycle,
			CycleLabel:          cycleDisplayLabel(currentCycle),
			Total:               0,
			Count:               0,
			Categories:          []CategoryStats{},
			CategoryDefinitions: allCats,
			AvailableCycles:     availableCycles,
			Salary:              salary,
			FixedTotal:          fixedAllocated,
			WantsTotal:          0,
			GoalsFunded:         goalsFunded,
			SalarySpent:         fixedAllocated,
			FixedBudget:         fixedBudget,
			WantsBudget:         wantsBudget,
			GoalsBudget:         goalsBudget,
			FundedCategoryIDs:   fundedIDs,
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
		emoji := emojiMap[cat.Category]
		if emoji == "" {
			emoji = "📌"
		}
		cat.Emoji = emoji
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

	// Compute fixed/spending totals. Allocated (set-aside) categories count via
	// their funding ticks (fixedAllocated, computed above); actual categories
	// count via real transactions. Spending (wants) is always transaction-driven.
	var fixedActualTotal, wantsTotal float64
	for _, cat := range categories {
		if excludeMap[cat.Category] {
			continue
		}
		switch typeMap[cat.Category] {
		case "fixed":
			if trackingMap[cat.Category] != "allocated" {
				fixedActualTotal += cat.Total
			}
		case "wants":
			wantsTotal += cat.Total
		}
	}
	fixedTotal := fixedActualTotal + fixedAllocated
	// "Spent from salary" excludes goals (savings is still your money).
	salarySpent := fixedTotal + wantsTotal

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
		Success:             true,
		Message:             message,
		Cycle:               currentCycle,
		CycleLabel:          cycleDisplayLabel(currentCycle),
		Total:               total,
		Count:               count,
		Categories:          categories,
		LastTransaction:     lastTransaction,
		AllTransactions:     allTransactions,
		CategoryDefinitions: allCats,
		AvailableCycles:     availableCycles,
		Salary:              salary,
		FixedTotal:          fixedTotal,
		WantsTotal:          wantsTotal,
		GoalsFunded:         goalsFunded,
		SalarySpent:         salarySpent,
		FixedBudget:         fixedBudget,
		WantsBudget:         wantsBudget,
		GoalsBudget:         goalsBudget,
		FundedCategoryIDs:   fundedIDs,
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
		SET description = ?, amount = ?, transaction_date = ?, category = ?, billing_cycle = ?, source = ?
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
		"manual",
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

func (c *DatabaseClient) seedCategories() error {
	var count int
	err := c.db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaults := []struct {
		name              string
		emoji             string
		excludeFromTotals int
	}{
		{"Groceries", "🛒", 0},
		{"Dining Out", "🍔", 0},
		{"Transport", "🚗", 0},
		{"Shopping", "🛍️", 0},
		{"Subscriptions", "📱", 0},
		{"Bills & Utilities", "💳", 0},
		{"Health", "💊", 0},
		{"Travel", "✈️", 0},
		{"Entertainment", "🎬", 0},
		{"Cash Withdrawal", "💵", 0},
		{"Income/Transfer", "💰", 1},
	}

	for _, d := range defaults {
		_, err := c.db.Exec(
			"INSERT INTO categories (name, emoji, exclude_from_totals, created_at) VALUES (?, ?, ?, ?)",
			d.name, d.emoji, d.excludeFromTotals, time.Now().Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("failed to seed category %s: %w", d.name, err)
		}
	}
	return nil
}

func (c *DatabaseClient) seedMerchantRules() error {
	var count int
	err := c.db.QueryRow("SELECT COUNT(*) FROM merchant_rules").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	rules := []struct {
		keyword  string
		category string
	}{
		{"carrefour", "Groceries"},
		{"spinneys", "Groceries"},
		{"lulu", "Groceries"},
		{"union coop", "Groceries"},
		{"talabat", "Dining Out"},
		{"deliveroo", "Dining Out"},
		{"zomato", "Dining Out"},
		{"starbucks", "Dining Out"},
		{"mcdonald", "Dining Out"},
		{"kfc", "Dining Out"},
		{"subway", "Dining Out"},
		{"hardee", "Dining Out"},
		{"pizza hut", "Dining Out"},
		{"careem", "Transport"},
		{"uber", "Transport"},
		{"salik", "Transport"},
		{"enoc", "Transport"},
		{"adnoc", "Transport"},
		{"emarat", "Transport"},
		{"noon", "Shopping"},
		{"amazon", "Shopping"},
		{"netflix", "Subscriptions"},
		{"spotify", "Subscriptions"},
		{"icloud", "Subscriptions"},
		{"chatgpt", "Subscriptions"},
		{"openai", "Subscriptions"},
		{"google one", "Subscriptions"},
		{"apple", "Subscriptions"},
		{"dewa", "Bills & Utilities"},
		{"addc", "Bills & Utilities"},
		{"etisalat", "Bills & Utilities"},
		{"du telecom", "Bills & Utilities"},
		{"aster", "Health"},
		{"boots", "Health"},
		{"medcare", "Health"},
		{"life pharmacy", "Health"},
	}

	for i, rule := range rules {
		priority := len(rules) - i
		_, err := c.db.Exec(
			"INSERT INTO merchant_rules (keyword, category, priority, created_at) VALUES (?, ?, ?, ?)",
			rule.keyword,
			rule.category,
			priority,
			time.Now().Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("failed to seed rule %s: %w", rule.keyword, err)
		}
	}

	return nil
}

func (c *DatabaseClient) CreateRule(keyword, category string, priority int) (*MerchantRule, int, int, error) {
	result, err := c.db.Exec(
		"INSERT INTO merchant_rules (keyword, category, priority, created_at) VALUES (?, ?, ?, ?)",
		keyword,
		category,
		priority,
		time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to create rule: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	matchCount, protectedCount, err := c.countRuleMatches(keyword)
	if err != nil {
		return nil, 0, 0, err
	}

	rule := &MerchantRule{
		ID:        id,
		Keyword:   keyword,
		Category:  category,
		Priority:  priority,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	return rule, matchCount, protectedCount, nil
}

func (c *DatabaseClient) countRuleMatches(keyword string) (int, int, error) {
	var matchCount int
	err := c.db.QueryRow(
		"SELECT COUNT(*) FROM transactions WHERE LOWER(description) LIKE '%' || LOWER(?) || '%' AND source != 'manual'",
		keyword,
	).Scan(&matchCount)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count rule matches: %w", err)
	}

	var protectedCount int
	err = c.db.QueryRow(
		"SELECT COUNT(*) FROM transactions WHERE LOWER(description) LIKE '%' || LOWER(?) || '%' AND source = 'manual'",
		keyword,
	).Scan(&protectedCount)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count protected transactions: %w", err)
	}

	return matchCount, protectedCount, nil
}

func (c *DatabaseClient) GetAllRules() ([]MerchantRule, error) {
	rows, err := c.db.Query("SELECT id, keyword, category, priority, created_at FROM merchant_rules ORDER BY priority DESC, id ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()

	var rules []MerchantRule
	for rows.Next() {
		var rule MerchantRule
		if err := rows.Scan(&rule.ID, &rule.Keyword, &rule.Category, &rule.Priority, &rule.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rules: %w", err)
	}

	return rules, nil
}

func (c *DatabaseClient) UpdateRule(id int64, keyword, category string, priority int) error {
	result, err := c.db.Exec(
		"UPDATE merchant_rules SET keyword=?, category=?, priority=? WHERE id=?",
		keyword,
		category,
		priority,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("rule not found")
	}

	return nil
}

func (c *DatabaseClient) DeleteRule(id int64) error {
	result, err := c.db.Exec("DELETE FROM merchant_rules WHERE id=?", id)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("rule not found")
	}

	return nil
}

func (c *DatabaseClient) MoveRulePriority(id int64, direction string) error {
	rules, err := c.GetAllRules()
	if err != nil {
		return err
	}

	var idx int
	var found bool
	for i, rule := range rules {
		if rule.ID == id {
			idx = i
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("rule not found")
	}

	var adjacentIdx int
	if direction == "up" {
		if idx == 0 {
			return nil
		}
		adjacentIdx = idx - 1
	} else if direction == "down" {
		if idx == len(rules)-1 {
			return nil
		}
		adjacentIdx = idx + 1
	} else {
		return fmt.Errorf("invalid direction: %s", direction)
	}

	rule := rules[idx]
	adjacent := rules[adjacentIdx]

	_, err = c.db.Exec("UPDATE merchant_rules SET priority=? WHERE id=?", adjacent.Priority, rule.ID)
	if err != nil {
		return fmt.Errorf("failed to update rule priority: %w", err)
	}

	_, err = c.db.Exec("UPDATE merchant_rules SET priority=? WHERE id=?", rule.Priority, adjacent.ID)
	if err != nil {
		return fmt.Errorf("failed to update adjacent rule priority: %w", err)
	}

	return nil
}

func (c *DatabaseClient) ApplyRuleSingle(ruleID int64) (int, int, error) {
	var rule MerchantRule
	err := c.db.QueryRow("SELECT id, keyword, category, priority, created_at FROM merchant_rules WHERE id=?", ruleID).Scan(
		&rule.ID, &rule.Keyword, &rule.Category, &rule.Priority, &rule.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, 0, fmt.Errorf("rule not found")
		}
		return 0, 0, fmt.Errorf("failed to fetch rule: %w", err)
	}

	result, err := c.db.Exec(
		"UPDATE transactions SET category=?, source='rule' WHERE LOWER(description) LIKE '%' || LOWER(?) || '%' AND source != 'manual'",
		rule.Category,
		rule.Keyword,
	)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to apply rule: %w", err)
	}

	updated, _ := result.RowsAffected()

	var protected int
	err = c.db.QueryRow(
		"SELECT COUNT(*) FROM transactions WHERE LOWER(description) LIKE '%' || LOWER(?) || '%' AND source = 'manual'",
		rule.Keyword,
	).Scan(&protected)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count protected transactions: %w", err)
	}

	return int(updated), protected, nil
}

func (c *DatabaseClient) ApplyAllRules() (int, int, error) {
	rules, err := c.GetAllRules()
	if err != nil {
		return 0, 0, err
	}

	rows, err := c.db.Query("SELECT id, description, source FROM transactions WHERE source != 'manual'")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	type txRecord struct {
		id   int64
		desc string
	}
	var transactions []txRecord
	for rows.Next() {
		var tx txRecord
		var source string
		if err := rows.Scan(&tx.id, &tx.desc, &source); err != nil {
			return 0, 0, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	updateMap := make(map[string][]int64)
	for _, tx := range transactions {
		for _, rule := range rules {
			if strings.Contains(strings.ToLower(tx.desc), strings.ToLower(rule.Keyword)) {
				updateMap[rule.Category] = append(updateMap[rule.Category], tx.id)
				break
			}
		}
	}

	var totalUpdated int
	for category, ids := range updateMap {
		placeholders := make([]string, len(ids))
		args := []interface{}{category, "rule"}
		for i, id := range ids {
			placeholders[i] = "?"
			args = append(args, id)
		}

		query := fmt.Sprintf(
			"UPDATE transactions SET category=?, source=? WHERE id IN (%s)",
			strings.Join(placeholders, ","),
		)
		result, err := c.db.Exec(query, args...)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to apply rules: %w", err)
		}
		updated, _ := result.RowsAffected()
		totalUpdated += int(updated)
	}

	var protected int
	err = c.db.QueryRow("SELECT COUNT(*) FROM transactions WHERE source = 'manual'").Scan(&protected)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count protected transactions: %w", err)
	}

	return totalUpdated, protected, nil
}

func (c *DatabaseClient) FindMatchingRule(description string) (*MerchantRule, error) {
	rows, err := c.db.Query("SELECT id, keyword, category, priority, created_at FROM merchant_rules ORDER BY priority DESC, id ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rule MerchantRule
		if err := rows.Scan(&rule.ID, &rule.Keyword, &rule.Category, &rule.Priority, &rule.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}

		if strings.Contains(strings.ToLower(description), strings.ToLower(rule.Keyword)) {
			return &rule, nil
		}
	}

	return nil, nil
}

func (c *DatabaseClient) GetAllCategories() ([]Category, error) {
	rows, err := c.db.Query(
		"SELECT id, name, emoji, exclude_from_totals, created_at, type, budget_amount, tracking FROM categories ORDER BY id ASC",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		var excl int
		var catType sql.NullString
		var budget sql.NullFloat64
		var tracking sql.NullString
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Emoji, &excl, &cat.CreatedAt, &catType, &budget, &tracking); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		cat.ExcludeFromTotals = excl == 1
		if catType.Valid {
			cat.Type = catType.String
		} else {
			cat.Type = "wants"
		}
		if budget.Valid {
			v := budget.Float64
			cat.BudgetAmount = &v
		}
		if tracking.Valid && tracking.String != "" {
			cat.Tracking = tracking.String
		} else {
			cat.Tracking = "actual"
		}
		categories = append(categories, cat)
	}
	return categories, rows.Err()
}

func (c *DatabaseClient) CreateCategory(name, emoji string, excludeFromTotals bool, catType string, budgetAmount *float64, tracking string) (*Category, error) {
	excl := 0
	if excludeFromTotals {
		excl = 1
	}
	if catType == "" {
		catType = "wants"
	}
	if tracking != "allocated" {
		tracking = "actual"
	}
	result, err := c.db.Exec(
		"INSERT INTO categories (name, emoji, exclude_from_totals, type, budget_amount, tracking, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		name, emoji, excl, catType, budgetAmount, tracking, time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}
	return &Category{
		ID:                id,
		Name:              name,
		Emoji:             emoji,
		ExcludeFromTotals: excludeFromTotals,
		Type:              catType,
		BudgetAmount:      budgetAmount,
		Tracking:          tracking,
		CreatedAt:         time.Now().Format(time.RFC3339),
	}, nil
}

func (c *DatabaseClient) UpdateCategory(id int64, name, emoji string, excludeFromTotals bool, catType string, budgetAmount *float64, tracking string) error {
	var oldName string
	err := c.db.QueryRow("SELECT name FROM categories WHERE id = ?", id).Scan(&oldName)
	if err == sql.ErrNoRows {
		return fmt.Errorf("category not found")
	}
	if err != nil {
		return fmt.Errorf("failed to fetch category: %w", err)
	}

	excl := 0
	if excludeFromTotals {
		excl = 1
	}
	if catType == "" {
		catType = "wants"
	}
	if tracking != "allocated" {
		tracking = "actual"
	}

	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		"UPDATE categories SET name=?, emoji=?, exclude_from_totals=?, type=?, budget_amount=?, tracking=? WHERE id=?",
		name, emoji, excl, catType, budgetAmount, tracking, id,
	); err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}

	if name != oldName {
		if _, err := tx.Exec(
			"UPDATE transactions SET category=? WHERE category=?", name, oldName,
		); err != nil {
			return fmt.Errorf("failed to cascade rename to transactions: %w", err)
		}
		if _, err := tx.Exec(
			"UPDATE merchant_rules SET category=? WHERE category=?", name, oldName,
		); err != nil {
			return fmt.Errorf("failed to cascade rename to merchant_rules: %w", err)
		}
	}

	return tx.Commit()
}

func (c *DatabaseClient) DeleteCategory(id int64) error {
	result, err := c.db.Exec("DELETE FROM categories WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("category not found")
	}
	return nil
}

// SetCategoryTarget sets, edits, or clears a category's spending target
// (budget_amount). A non-nil amount sets/edits the target; nil clears it.
func (c *DatabaseClient) SetCategoryTarget(id int64, amount *float64) error {
	result, err := c.db.Exec("UPDATE categories SET budget_amount=? WHERE id=?", amount, id)
	if err != nil {
		return fmt.Errorf("failed to set category target: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("category not found")
	}
	return nil
}

// --- Settings ---

func (c *DatabaseClient) GetSetting(key string) (string, error) {
	var value string
	err := c.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get setting %s: %w", key, err)
	}
	return value, nil
}

func (c *DatabaseClient) SetSetting(key, value string) error {
	_, err := c.db.Exec(
		"INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		key, value,
	)
	if err != nil {
		return fmt.Errorf("failed to set setting %s: %w", key, err)
	}
	return nil
}

// GetSalary returns the configured monthly salary, defaulting to 32500.
func (c *DatabaseClient) GetSalary() float64 {
	v, err := c.GetSetting("monthly_salary")
	if err != nil || v == "" {
		return 32500
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 32500
	}
	return f
}

func (c *DatabaseClient) SetSalary(amount float64) error {
	return c.SetSetting("monthly_salary", strconv.FormatFloat(amount, 'f', -1, 64))
}

// --- Per-cycle funding (set-aside ticks for allocated categories) ---

// GetFunding returns category_id → funded amount for a billing cycle.
func (c *DatabaseClient) GetFunding(cycle string) (map[int64]float64, error) {
	rows, err := c.db.Query("SELECT category_id, amount FROM cycle_funding WHERE cycle = ?", cycle)
	if err != nil {
		return nil, fmt.Errorf("failed to query funding: %w", err)
	}
	defer rows.Close()

	funded := make(map[int64]float64)
	for rows.Next() {
		var id int64
		var amount float64
		if err := rows.Scan(&id, &amount); err != nil {
			return nil, fmt.Errorf("failed to scan funding: %w", err)
		}
		funded[id] = amount
	}
	return funded, rows.Err()
}

// SetFunding marks an allocated category as funded for a cycle, snapshotting amount.
func (c *DatabaseClient) SetFunding(cycle string, categoryID int64, amount float64) error {
	_, err := c.db.Exec(
		`INSERT INTO cycle_funding (cycle, category_id, amount, funded_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT(cycle, category_id) DO UPDATE SET amount = excluded.amount, funded_at = excluded.funded_at`,
		cycle, categoryID, amount, time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to set funding: %w", err)
	}
	return nil
}

// DeleteFunding clears the funded tick for a category in a cycle.
func (c *DatabaseClient) DeleteFunding(cycle string, categoryID int64) error {
	_, err := c.db.Exec("DELETE FROM cycle_funding WHERE cycle = ? AND category_id = ?", cycle, categoryID)
	if err != nil {
		return fmt.Errorf("failed to delete funding: %w", err)
	}
	return nil
}

// GetCategory returns a single category by id.
func (c *DatabaseClient) GetCategory(id int64) (*Category, error) {
	cats, err := c.GetAllCategories()
	if err != nil {
		return nil, err
	}
	for i := range cats {
		if cats[i].ID == id {
			return &cats[i], nil
		}
	}
	return nil, fmt.Errorf("category not found")
}

func (c *DatabaseClient) Close() error {
	log.Printf("[Database] Closing database connection")
	return c.db.Close()
}
