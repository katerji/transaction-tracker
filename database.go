package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
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
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Emoji             string `json:"emoji"`
	ExcludeFromTotals bool   `json:"excludeFromTotals"`
	CreatedAt         string `json:"createdAt"`
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

	return nil
}

func (c *DatabaseClient) addColumnIfNotExists(table, colDef string) error {
	_, err := c.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, colDef))
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		return err
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

func (c *DatabaseClient) GetStats() (*StatsResponse, error) {
	currentCycle := calculateBillingCycle(time.Now().Format("2006-01-02"))
	log.Printf("[Database] Fetching stats for billing cycle: %s", currentCycle)

	// Fetch all categories and build emoji map
	allCats, err := c.GetAllCategories()
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	emojiMap := make(map[string]string, len(allCats))
	for _, cat := range allCats {
		emojiMap[cat.Name] = cat.Emoji
	}

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
			Total:               0,
			Count:               0,
			Categories:          []CategoryStats{},
			CategoryDefinitions: allCats,
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
		Total:               total,
		Count:               count,
		Categories:          categories,
		LastTransaction:     lastTransaction,
		AllTransactions:     allTransactions,
		CategoryDefinitions: allCats,
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
		"SELECT id, name, emoji, exclude_from_totals, created_at FROM categories ORDER BY id ASC",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		var excl int
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Emoji, &excl, &cat.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		cat.ExcludeFromTotals = excl == 1
		categories = append(categories, cat)
	}
	return categories, rows.Err()
}

func (c *DatabaseClient) CreateCategory(name, emoji string, excludeFromTotals bool) (*Category, error) {
	excl := 0
	if excludeFromTotals {
		excl = 1
	}
	result, err := c.db.Exec(
		"INSERT INTO categories (name, emoji, exclude_from_totals, created_at) VALUES (?, ?, ?, ?)",
		name, emoji, excl, time.Now().Format(time.RFC3339),
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
		CreatedAt:         time.Now().Format(time.RFC3339),
	}, nil
}

func (c *DatabaseClient) UpdateCategory(id int64, name, emoji string, excludeFromTotals bool) error {
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

	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		"UPDATE categories SET name=?, emoji=?, exclude_from_totals=? WHERE id=?",
		name, emoji, excl, id,
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
	var name string
	err := c.db.QueryRow("SELECT name FROM categories WHERE id = ?", id).Scan(&name)
	if err == sql.ErrNoRows {
		return fmt.Errorf("category not found")
	}
	if err != nil {
		return fmt.Errorf("failed to fetch category: %w", err)
	}

	var count int
	if err := c.db.QueryRow(
		"SELECT COUNT(*) FROM transactions WHERE category = ?", name,
	).Scan(&count); err != nil {
		return fmt.Errorf("failed to count transactions: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete: %d transaction(s) use this category — reassign them first", count)
	}

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

func (c *DatabaseClient) Close() error {
	log.Printf("[Database] Closing database connection")
	return c.db.Close()
}
