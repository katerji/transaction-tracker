package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	OpenAIKey    string
	DatabasePath string
	Port         string
}

type TransactionRequest struct {
	Text string `json:"text"`
}

type ManualTransactionRequest struct {
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Date        string  `json:"date"`
	Category    string  `json:"category"`
}

type TransactionResponse struct {
	Success      bool          `json:"success"`
	Message      string        `json:"message"`
	Count        int           `json:"count"`
	Total        float64       `json:"total"`
	Transactions []Transaction `json:"transactions,omitempty"`
}

type StatsResponse struct {
	Success             bool                `json:"success"`
	Message             string              `json:"message"`
	Cycle               string              `json:"cycle"`
	CycleLabel          string              `json:"cycleLabel"`
	Total               float64             `json:"total"`
	Count               int                 `json:"count"`
	Categories          []CategoryStats     `json:"categories,omitempty"`
	LastTransaction     *TransactionSummary `json:"lastTransaction,omitempty"`
	AllTransactions     []Transaction       `json:"allTransactions,omitempty"`
	CategoryDefinitions []Category          `json:"categoryDefinitions,omitempty"`
	AvailableCycles     []CycleOption       `json:"availableCycles,omitempty"`
	FixedTotal   float64 `json:"fixed_total"`
	WantsTotal   float64 `json:"wants_total"`
	FixedBudget  float64 `json:"fixed_budget"`
	WantsBudget  float64 `json:"wants_budget"`
}

type CategoryStats struct {
	Category     string        `json:"category"`
	Emoji        string        `json:"emoji"`
	Total        float64       `json:"total"`
	Count        int           `json:"count"`
	Transactions []Transaction `json:"transactions,omitempty"`
}

type TransactionSummary struct {
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Date        string  `json:"date"`
}

func loadConfig() (*Config, error) {
	config := &Config{
		OpenAIKey:    os.Getenv("OPENAI_API_KEY"),
		DatabasePath: os.Getenv("DATABASE_PATH"),
		Port:         os.Getenv("PORT"),
	}

	if config.Port == "" {
		config.Port = "8080"
	}

	if config.DatabasePath == "" {
		// Default to local path for development
		config.DatabasePath = "./transactions.db"
	}

	if config.OpenAIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	return config, nil
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Printf("[Server] Starting Transaction Tracker...")

	// Load .env file if it exists (for local development)
	// In production (Docker/Fly.io), environment variables are set directly
	if err := godotenv.Load(); err != nil {
		log.Printf("[Server] No .env file found or error loading it (this is normal in production): %v", err)
	} else {
		log.Printf("[Server] Loaded configuration from .env file")
	}

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("[Server] Configuration error: %v", err)
	}

	log.Printf("[Server] Initializing OpenAI client...")
	openAIClient := NewOpenAIClient(config.OpenAIKey)

	dbClient, err := NewDatabaseClient(config.DatabasePath)
	if err != nil {
		log.Fatalf("[Server] Database connection error: %v", err)
	}
	defer dbClient.Close()

	// Serve static JS files from embedded FS
	staticSub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("[Server] Failed to create sub filesystem: %v", err)
	}
	staticHandler := http.FileServer(http.FS(staticSub))

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/transaction/manual", manualTransactionHandler(dbClient))
	http.HandleFunc("/transaction", transactionHandler(openAIClient, dbClient))
	http.HandleFunc("/transaction/", transactionDetailHandler(dbClient))
	http.HandleFunc("/dashboard", dashboardHandler(dbClient))
	http.HandleFunc("/export", exportHandler(dbClient))
	http.HandleFunc("/import", importHandler(dbClient))
	http.HandleFunc("/rules", rulesHandler(dbClient))
	http.HandleFunc("/rules/", ruleDetailHandler(dbClient))
	http.HandleFunc("/categories", categoriesHandler(dbClient))
	http.HandleFunc("/categories/", categoryDetailHandler(dbClient))
	http.Handle("/js/", staticHandler)
	http.HandleFunc("/", indexHandler)

	addr := ":" + config.Port
	log.Printf("[Server] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("[Server] Transaction Tracker starting...")
	log.Printf("[Server] Database: %s", config.DatabasePath)
	log.Printf("[Server] Port: %s", config.Port)
	log.Printf("[Server] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("[Server] Endpoints:")
	log.Printf("[Server]   GET    /              - Dashboard UI")
	log.Printf("[Server]   POST   /transaction   - Log new transaction")
	log.Printf("[Server]   POST   /transaction/manual - Add manual transaction")
	log.Printf("[Server]   PUT    /transaction/:id - Update transaction")
	log.Printf("[Server]   DELETE /transaction/:id - Delete transaction")
	log.Printf("[Server]   GET    /dashboard     - Get dashboard data (renamed from /stats)")
	log.Printf("[Server]   GET    /export        - Export CSV")
	log.Printf("[Server]   POST   /import        - Import CSV")
	log.Printf("[Server]   GET    /categories    - Get all categories")
	log.Printf("[Server]   POST   /categories    - Create category")
	log.Printf("[Server]   PUT    /categories/:id - Update category")
	log.Printf("[Server]   DELETE /categories/:id - Delete category")
	log.Printf("[Server]   GET    /rules         - Get all merchant rules")
	log.Printf("[Server]   POST   /rules         - Create merchant rule")
	log.Printf("[Server]   PUT    /rules/:id     - Update merchant rule")
	log.Printf("[Server]   DELETE /rules/:id     - Delete merchant rule")
	log.Printf("[Server]   POST   /rules/:id/apply - Apply single rule")
	log.Printf("[Server]   POST   /rules/apply-all - Apply all rules")
	log.Printf("[Server]   POST   /rules/:id/move - Move rule priority")
	log.Printf("[Server]   GET    /health        - Health check")
	log.Printf("[Server] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("[Server] Server ready at http://localhost:%s", config.Port)
	log.Printf("[Server] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("[Server] Failed to start: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] GET /health - Health check from %s", r.RemoteAddr)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func transactionHandler(openAI *OpenAIClient, db *DatabaseClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API] POST /transaction - New transaction request from %s", r.RemoteAddr)

		if r.Method != http.MethodPost {
			log.Printf("[API] Method not allowed: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req TransactionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("[API] Invalid request body: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Text == "" {
			log.Printf("[API] Empty text field")
			http.Error(w, "Text field is required", http.StatusBadRequest)
			return
		}

		log.Printf("[API] Processing transaction text: %s", req.Text)

		// Fetch categories for OpenAI prompt
		categories, err := db.GetAllCategories()
		if err != nil {
			log.Printf("[API] Failed to get categories: %v", err)
			http.Error(w, "Failed to retrieve categories", http.StatusInternalServerError)
			return
		}

		if len(categories) == 0 {
			log.Printf("[API] No categories defined — cannot parse transactions")
			http.Error(w, "No categories defined", http.StatusInternalServerError)
			return
		}

		// Build emoji map for response
		emojiMap := make(map[string]string, len(categories))
		for _, cat := range categories {
			emojiMap[cat.Name] = cat.Emoji
		}

		// Parse transactions using OpenAI
		transactions, err := openAI.ParseTransactions(req.Text, categories)
		if err != nil {
			log.Printf("[API] OpenAI parsing error: %v", err)
			http.Error(w, "Failed to parse transactions", http.StatusInternalServerError)
			return
		}

		log.Printf("[API] OpenAI parsed %d transaction(s)", len(transactions))

		if len(transactions) == 0 {
			log.Printf("[API] No transactions found in text")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TransactionResponse{
				Success: false,
				Message: "No transactions found in the provided text",
				Count:   0,
			})
			return
		}

		// Process and save each transaction to database
		var savedTransactions []Transaction
		var total float64

		for i, tx := range transactions {
			log.Printf("[API] Processing transaction %d/%d", i+1, len(transactions))
			enriched := enrichTransaction(tx)

			rule, err := db.FindMatchingRule(enriched.Description)
			if err == nil && rule != nil {
				enriched.Category = rule.Category
				enriched.Source = "rule"
			} else {
				enriched.Source = "openai"
			}

			id, err := db.SaveTransaction(enriched)
			if err != nil {
				log.Printf("[API] Failed to save transaction to database: %v", err)
				continue
			}

			enriched.ID = id
			savedTransactions = append(savedTransactions, enriched)
			total += enriched.Amount
		}

		log.Printf("[API] Successfully saved %d/%d transaction(s), total: %.2f AED", len(savedTransactions), len(transactions), total)

		// Build response message
		message := fmt.Sprintf("✅ Added %d transaction%s!\n\n", len(savedTransactions), pluralize(len(savedTransactions)))
		for i, tx := range savedTransactions {
			message += fmt.Sprintf("%d. %s\n", i+1, tx.Description)
			message += fmt.Sprintf("   💰 Amount: %.2f AED\n", tx.Amount)
			emoji := emojiMap[tx.Category]
			if emoji == "" {
				emoji = "📌"
			}
			message += fmt.Sprintf("   📁 Category: %s %s (%d%% confidence)\n", emoji, tx.Category, tx.Confidence)
			message += fmt.Sprintf("   📅 Cycle: %s\n\n", tx.BillingCycle)
		}
		message += fmt.Sprintf("━━━━━━━━━━━━━━━\n💵 Total: %.2f AED", total)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TransactionResponse{
			Success:      true,
			Message:      message,
			Count:        len(savedTransactions),
			Total:        total,
			Transactions: savedTransactions,
		})
	}
}

func dashboardHandler(db *DatabaseClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API] GET /dashboard - Request from %s", r.RemoteAddr)

		if r.Method != http.MethodGet {
			log.Printf("[API] Method not allowed: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		cycle := r.URL.Query().Get("cycle")
		stats, err := db.GetStats(cycle)
		if err != nil {
			log.Printf("[API] Failed to get stats: %v", err)
			http.Error(w, "Failed to retrieve statistics", http.StatusInternalServerError)
			return
		}

		log.Printf("[API] Returning stats: %d transactions, %.2f AED total", stats.Count, stats.Total)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}

func exportHandler(db *DatabaseClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API] GET /export - CSV export request from %s", r.RemoteAddr)

		if r.Method != http.MethodGet {
			log.Printf("[API] Method not allowed: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		transactions, err := db.GetAllTransactionsGroupedByCycle()
		if err != nil {
			log.Printf("[API] Failed to get transactions for export: %v", err)
			http.Error(w, "Failed to export transactions", http.StatusInternalServerError)
			return
		}

		// Fetch categories and build excluded map
		allCats, err := db.GetAllCategories()
		if err != nil {
			log.Printf("[API] Failed to get categories for export: %v", err)
			http.Error(w, "Failed to export", http.StatusInternalServerError)
			return
		}
		excludedCats := make(map[string]bool)
		for _, cat := range allCats {
			if cat.ExcludeFromTotals {
				excludedCats[cat.Name] = true
			}
		}

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="transactions.csv"`)

		writer := csv.NewWriter(w)
		defer writer.Flush()

		// Header row
		writer.Write([]string{"Date", "Description", "Amount (AED)", "Category"})

		var grandTotal float64
		var currentCycle string
		var cycleSubtotal float64

		for _, tx := range transactions {
			if tx.BillingCycle != currentCycle {
				// Write subtotal for previous cycle (if any)
				if currentCycle != "" {
					writer.Write([]string{"", "Subtotal", fmt.Sprintf("%.2f", cycleSubtotal), ""})
					writer.Write([]string{"", "", "", ""})
					grandTotal += cycleSubtotal
					cycleSubtotal = 0
				}
				currentCycle = tx.BillingCycle
				writer.Write([]string{fmt.Sprintf("--- %s ---", currentCycle), "", "", ""})
			}

			writer.Write([]string{tx.Date, tx.Description, fmt.Sprintf("%.2f", tx.Amount), tx.Category})

			if !excludedCats[tx.Category] {
				cycleSubtotal += tx.Amount
			}
		}

		// Write final cycle subtotal
		if currentCycle != "" {
			writer.Write([]string{"", "Subtotal", fmt.Sprintf("%.2f", cycleSubtotal), ""})
			writer.Write([]string{"", "", "", ""})
			grandTotal += cycleSubtotal
		}

		// Grand total
		writer.Write([]string{"", "Grand Total", fmt.Sprintf("%.2f", grandTotal), ""})

		log.Printf("[API] CSV export completed: %d transactions", len(transactions))
	}
}

type ImportResponse struct {
	Success    bool     `json:"success"`
	Imported   int      `json:"imported"`
	Duplicates int      `json:"duplicates"`
	Errors     []string `json:"errors"`
	Message    string   `json:"message"`
}

func importHandler(db *DatabaseClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API] POST /import - CSV import request from %s", r.RemoteAddr)

		if r.Method != http.MethodPost {
			log.Printf("[API] Method not allowed: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			log.Printf("[API] Failed to parse multipart form: %v", err)
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			log.Printf("[API] No file uploaded: %v", err)
			http.Error(w, "No file uploaded", http.StatusBadRequest)
			return
		}
		defer file.Close()

		reader := csv.NewReader(file)
		reader.FieldsPerRecord = -1 // allow variable field counts

		var imported, duplicates int
		var errors []string
		rowNum := 0

		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			rowNum++

			if err != nil {
				errors = append(errors, fmt.Sprintf("row %d: %v", rowNum, err))
				continue
			}

			// Skip header row
			if rowNum == 1 {
				continue
			}

			// Skip rows with fewer than 4 columns
			if len(record) < 4 {
				continue
			}

			// Skip blank rows (all fields empty)
			allEmpty := true
			for _, field := range record {
				if strings.TrimSpace(field) != "" {
					allEmpty = false
					break
				}
			}
			if allEmpty {
				continue
			}

			// Skip cycle headers (Date starts with ---)
			if strings.HasPrefix(record[0], "---") {
				continue
			}

			// Skip Subtotal and Grand Total rows
			desc := strings.TrimSpace(record[1])
			if desc == "Subtotal" || desc == "Grand Total" {
				continue
			}

			// Parse data row: Date, Description, Amount, Category
			date := strings.TrimSpace(record[0])
			description := desc
			amountStr := strings.TrimSpace(record[2])
			category := strings.TrimSpace(record[3])

			if date == "" || description == "" || category == "" {
				continue
			}

			amount, err := strconv.ParseFloat(amountStr, 64)
			if err != nil {
				errors = append(errors, fmt.Sprintf("row %d: invalid amount '%s'", rowNum, amountStr))
				continue
			}

			tx := Transaction{
				Description: description,
				Amount:      amount,
				Date:        date,
				Category:    category,
				Confidence:  100,
			}
			enriched := enrichTransaction(tx)

			_, err = db.SaveTransaction(enriched)
			if err != nil {
				if strings.Contains(err.Error(), "UNIQUE constraint") {
					duplicates++
				} else {
					errors = append(errors, fmt.Sprintf("row %d: %v", rowNum, err))
				}
				continue
			}

			imported++
		}

		message := fmt.Sprintf("Imported %d transactions (%d duplicates skipped, %d errors)", imported, duplicates, len(errors))
		log.Printf("[API] Import completed: %s", message)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ImportResponse{
			Success:    true,
			Imported:   imported,
			Duplicates: duplicates,
			Errors:     errors,
			Message:    message,
		})
	}
}

func manualTransactionHandler(db *DatabaseClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API] POST /transaction/manual - Manual transaction request from %s", r.RemoteAddr)

		if r.Method != http.MethodPost {
			log.Printf("[API] Method not allowed: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ManualTransactionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("[API] Invalid request body: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.Description == "" {
			log.Printf("[API] Missing required field: description")
			http.Error(w, "Description is required", http.StatusBadRequest)
			return
		}
		if req.Amount == 0 {
			log.Printf("[API] Missing required field: amount")
			http.Error(w, "Amount is required", http.StatusBadRequest)
			return
		}
		if req.Date == "" {
			log.Printf("[API] Missing required field: date")
			http.Error(w, "Date is required", http.StatusBadRequest)
			return
		}
		if req.Category == "" {
			log.Printf("[API] Missing required field: category")
			http.Error(w, "Category is required", http.StatusBadRequest)
			return
		}

		log.Printf("[API] Creating manual transaction: %s (%.2f AED, %s, %s)",
			req.Description, req.Amount, req.Date, req.Category)

		// Create transaction with manual data
		tx := Transaction{
			Description: req.Description,
			Amount:      req.Amount,
			Date:        req.Date,
			Category:    req.Category,
			Confidence:  100, // Manual entry has 100% confidence
			Source:      "manual",
		}

		// Enrich with timestamp and billing cycle
		enriched := enrichTransaction(tx)

		// Save to database
		id, err := db.SaveTransaction(enriched)
		if err != nil {
			log.Printf("[API] Failed to save transaction to database: %v", err)
			http.Error(w, "Failed to save transaction", http.StatusInternalServerError)
			return
		}

		// Set the ID on the transaction for the response
		enriched.ID = id

		log.Printf("[API] Manual transaction saved successfully with ID %d", id)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":     true,
			"message":     "Transaction added successfully",
			"transaction": enriched,
		})
	}
}

func enrichTransaction(tx Transaction) Transaction {
	tx.Timestamp = time.Now().Format(time.RFC3339)
	tx.BillingCycle = calculateBillingCycle(tx.Date)
	return tx
}

func calculateBillingCycle(dateStr string) string {
	txDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		txDate = time.Now()
	}

	cycleStart := txDate
	if txDate.Day() < 23 {
		cycleStart = txDate.AddDate(0, -1, 0)
	}
	cycleStart = time.Date(cycleStart.Year(), cycleStart.Month(), 23, 0, 0, 0, 0, time.UTC)

	return cycleStart.Format("Jan 2006")
}

// cycleDisplayLabel converts an internal start-month cycle key ("Jun 2026") into
// the end-month label shown to users ("July 2026"). A billing cycle runs from the
// 23rd to the 22nd, so it ends in the month after its start month.
func cycleDisplayLabel(cycle string) string {
	t, err := time.Parse("Jan 2006", cycle)
	if err != nil {
		return cycle
	}
	return t.AddDate(0, 1, 0).Format("January 2006")
}

// CycleOption is a selectable billing period: Cycle is the internal start-month key
// (sent back as ?cycle=), Label is the end-month string shown in the UI.
type CycleOption struct {
	Cycle string `json:"cycle"`
	Label string `json:"label"`
}

// selectableCycles generates the list of billing periods offered in the dashboard
// picker — every consecutive period from the current one down to a hard floor,
// newest-first. The list is generated (not derived from stored data), so it rolls
// forward automatically as time passes.
func selectableCycles() []CycleOption {
	current := calculateBillingCycle(time.Now().Format("2006-01-02"))
	start, err := time.Parse("Jan 2006", current)
	if err != nil {
		return nil
	}

	// Floor: never show a period whose end month is before January 2026.
	floor := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	var opts []CycleOption
	for s := start; ; s = s.AddDate(0, -1, 0) {
		end := s.AddDate(0, 1, 0)
		if end.Before(floor) {
			break
		}
		opts = append(opts, CycleOption{
			Cycle: s.Format("Jan 2006"),
			Label: end.Format("January 2006"),
		})
	}
	return opts
}

func rulesHandler(db *DatabaseClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			log.Printf("[API] GET /rules - Request from %s", r.RemoteAddr)
			rules, err := db.GetAllRules()
			if err != nil {
				log.Printf("[API] Failed to get rules: %v", err)
				http.Error(w, "Failed to retrieve rules", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"rules":   rules,
			})

		case http.MethodPost:
			log.Printf("[API] POST /rules - Create rule request from %s", r.RemoteAddr)
			var req struct {
				Keyword  string `json:"keyword"`
				Category string `json:"category"`
				Priority int    `json:"priority"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				log.Printf("[API] Invalid request body: %v", err)
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			rule, matchCount, protectedCount, err := db.CreateRule(req.Keyword, req.Category, req.Priority)
			if err != nil {
				log.Printf("[API] Failed to create rule: %v", err)
				http.Error(w, "Failed to create rule", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":         true,
				"rule":            rule,
				"match_count":     matchCount,
				"protected_count": protectedCount,
			})

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func ruleDetailHandler(db *DatabaseClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/rules/apply-all" {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			log.Printf("[API] POST /rules/apply-all - Apply all rules from %s", r.RemoteAddr)
			updated, protected, err := db.ApplyAllRules()
			if err != nil {
				log.Printf("[API] Failed to apply all rules: %v", err)
				http.Error(w, "Failed to apply rules", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":   true,
				"updated":   updated,
				"protected": protected,
			})
			return
		}

		if path == "/rules/" || path == "/rules" {
			http.NotFound(w, r)
			return
		}

		idStr := path[len("/rules/"):]
		var id int64
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
			http.Error(w, "Invalid rule ID", http.StatusBadRequest)
			return
		}

		if strings.HasSuffix(path, "/apply") {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			log.Printf("[API] POST /rules/%d/apply - Apply rule from %s", id, r.RemoteAddr)
			updated, protected, err := db.ApplyRuleSingle(id)
			if err != nil {
				log.Printf("[API] Failed to apply rule: %v", err)
				if err.Error() == "rule not found" {
					http.Error(w, "Rule not found", http.StatusNotFound)
				} else {
					http.Error(w, "Failed to apply rule", http.StatusInternalServerError)
				}
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":   true,
				"updated":   updated,
				"protected": protected,
			})
			return
		}

		if strings.HasSuffix(path, "/move") {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			log.Printf("[API] POST /rules/%d/move - Move rule from %s", id, r.RemoteAddr)
			var req struct {
				Direction string `json:"direction"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				log.Printf("[API] Invalid request body: %v", err)
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			if err := db.MoveRulePriority(id, req.Direction); err != nil {
				log.Printf("[API] Failed to move rule: %v", err)
				http.Error(w, "Failed to move rule", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
			})
			return
		}

		switch r.Method {
		case http.MethodPut:
			log.Printf("[API] PUT /rules/%d - Update rule from %s", id, r.RemoteAddr)
			var req struct {
				Keyword  string `json:"keyword"`
				Category string `json:"category"`
				Priority int    `json:"priority"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				log.Printf("[API] Invalid request body: %v", err)
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			if err := db.UpdateRule(id, req.Keyword, req.Category, req.Priority); err != nil {
				log.Printf("[API] Failed to update rule: %v", err)
				if err.Error() == "rule not found" {
					http.Error(w, "Rule not found", http.StatusNotFound)
				} else {
					http.Error(w, "Failed to update rule", http.StatusInternalServerError)
				}
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
			})

		case http.MethodDelete:
			log.Printf("[API] DELETE /rules/%d - Delete rule from %s", id, r.RemoteAddr)
			if err := db.DeleteRule(id); err != nil {
				log.Printf("[API] Failed to delete rule: %v", err)
				if err.Error() == "rule not found" {
					http.Error(w, "Rule not found", http.StatusNotFound)
				} else {
					http.Error(w, "Failed to delete rule", http.StatusInternalServerError)
				}
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
			})

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func categoriesHandler(db *DatabaseClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			log.Printf("[API] GET /categories - Request from %s", r.RemoteAddr)
			cats, err := db.GetAllCategories()
			if err != nil {
				log.Printf("[API] Failed to get categories: %v", err)
				http.Error(w, "Failed to retrieve categories", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    true,
				"categories": cats,
			})

		case http.MethodPost:
			log.Printf("[API] POST /categories - Create category from %s", r.RemoteAddr)
			var req struct {
				Name              string   `json:"name"`
				Emoji             string   `json:"emoji"`
				ExcludeFromTotals bool     `json:"excludeFromTotals"`
				Type              string   `json:"type"`
				BudgetAmount      *float64 `json:"budgetAmount"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}
			if req.Name == "" {
				http.Error(w, "Name is required", http.StatusBadRequest)
				return
			}
			cat, err := db.CreateCategory(req.Name, req.Emoji, req.ExcludeFromTotals, req.Type, req.BudgetAmount)
			if err != nil {
				log.Printf("[API] Failed to create category: %v", err)
				http.Error(w, "Failed to create category", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":  true,
				"category": cat,
			})

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func categoryDetailHandler(db *DatabaseClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		idStr := path[len("/categories/"):]
		var id int64
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
			http.Error(w, "Invalid category ID", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodPut:
			log.Printf("[API] PUT /categories/%d - Update from %s", id, r.RemoteAddr)
			var req struct {
				Name              string   `json:"name"`
				Emoji             string   `json:"emoji"`
				ExcludeFromTotals bool     `json:"excludeFromTotals"`
				Type              string   `json:"type"`
				BudgetAmount      *float64 `json:"budgetAmount"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}
			if req.Name == "" {
				http.Error(w, "Name is required", http.StatusBadRequest)
				return
			}
			if err := db.UpdateCategory(id, req.Name, req.Emoji, req.ExcludeFromTotals, req.Type, req.BudgetAmount); err != nil {
				log.Printf("[API] Failed to update category: %v", err)
				if err.Error() == "category not found" {
					http.Error(w, "Category not found", http.StatusNotFound)
				} else {
					http.Error(w, "Failed to update category", http.StatusInternalServerError)
				}
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true})

		case http.MethodDelete:
			log.Printf("[API] DELETE /categories/%d - Delete from %s", id, r.RemoteAddr)
			if err := db.DeleteCategory(id); err != nil {
				log.Printf("[API] Failed to delete category: %v", err)
				errMsg := err.Error()
				if strings.HasPrefix(errMsg, "cannot delete:") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusConflict)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"message": errMsg,
					})
				} else if errMsg == "category not found" {
					http.Error(w, "Category not found", http.StatusNotFound)
				} else {
					http.Error(w, "Failed to delete category", http.StatusInternalServerError)
				}
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true})

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Only serve dashboard on root path
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	log.Printf("[API] GET / - Dashboard request from %s", r.RemoteAddr)

	indexHTML, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "Dashboard not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func transactionDetailHandler(db *DatabaseClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract ID from path /transaction/:id
		path := r.URL.Path
		if path == "/transaction/" || path == "/transaction" {
			http.NotFound(w, r)
			return
		}

		idStr := path[len("/transaction/"):]
		id, err := fmt.Sscanf(idStr, "%d", new(int64))
		if err != nil || id == 0 {
			http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
			return
		}

		var transactionID int64
		fmt.Sscanf(idStr, "%d", &transactionID)

		switch r.Method {
		case http.MethodPut:
			updateTransactionHandler(db, transactionID)(w, r)
		case http.MethodDelete:
			deleteTransactionHandler(db, transactionID)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func updateTransactionHandler(db *DatabaseClient, id int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API] PUT /transaction/%d - Update request from %s", id, r.RemoteAddr)

		var tx Transaction
		if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
			log.Printf("[API] Invalid request body: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Recalculate billing cycle based on new date
		tx.BillingCycle = calculateBillingCycle(tx.Date)

		if err := db.UpdateTransaction(id, tx); err != nil {
			log.Printf("[API] Failed to update transaction: %v", err)
			if err.Error() == "transaction not found" {
				http.Error(w, "Transaction not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to update transaction", http.StatusInternalServerError)
			}
			return
		}

		log.Printf("[API] Transaction %d updated successfully", id)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Transaction updated successfully",
		})
	}
}

func deleteTransactionHandler(db *DatabaseClient, id int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API] DELETE /transaction/%d - Delete request from %s", id, r.RemoteAddr)

		if err := db.DeleteTransaction(id); err != nil {
			log.Printf("[API] Failed to delete transaction: %v", err)
			if err.Error() == "transaction not found" {
				http.Error(w, "Transaction not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to delete transaction", http.StatusInternalServerError)
			}
			return
		}

		log.Printf("[API] Transaction %d deleted successfully", id)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Transaction deleted successfully",
		})
	}
}
