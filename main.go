package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type Config struct {
	OpenAIKey    string
	DatabasePath string
	Port         string
}

type TransactionRequest struct {
	Text string `json:"text"`
}

type TransactionResponse struct {
	Success      bool          `json:"success"`
	Message      string        `json:"message"`
	Count        int           `json:"count"`
	Total        float64       `json:"total"`
	Transactions []Transaction `json:"transactions,omitempty"`
}

type StatsResponse struct {
	Success         bool                `json:"success"`
	Message         string              `json:"message"`
	Cycle           string              `json:"cycle"`
	Total           float64             `json:"total"`
	Count           int                 `json:"count"`
	Categories      []CategoryStats     `json:"categories,omitempty"`
	LastTransaction *TransactionSummary `json:"lastTransaction,omitempty"`
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

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/transaction", transactionHandler(openAIClient, dbClient))
	http.HandleFunc("/transaction/", transactionDetailHandler(dbClient))
	http.HandleFunc("/stats", statsHandler(dbClient))
	http.HandleFunc("/", dashboardHandler)

	addr := ":" + config.Port
	log.Printf("[Server] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("[Server] Transaction Tracker starting...")
	log.Printf("[Server] Database: %s", config.DatabasePath)
	log.Printf("[Server] Port: %s", config.Port)
	log.Printf("[Server] ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("[Server] Endpoints:")
	log.Printf("[Server]   GET    /              - Dashboard UI")
	log.Printf("[Server]   POST   /transaction   - Log new transaction")
	log.Printf("[Server]   PUT    /transaction/:id - Update transaction")
	log.Printf("[Server]   DELETE /transaction/:id - Delete transaction")
	log.Printf("[Server]   GET    /stats         - Get spending statistics")
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

		// Parse transactions using OpenAI
		transactions, err := openAI.ParseTransactions(req.Text)
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

			if err := db.SaveTransaction(enriched); err != nil {
				log.Printf("[API] Failed to save transaction to database: %v", err)
				continue
			}

			savedTransactions = append(savedTransactions, enriched)
			total += enriched.Amount
		}

		log.Printf("[API] Successfully saved %d/%d transaction(s), total: %.2f AED", len(savedTransactions), len(transactions), total)

		// Build response message
		message := fmt.Sprintf("✅ Added %d transaction%s!\n\n", len(savedTransactions), pluralize(len(savedTransactions)))
		for i, tx := range savedTransactions {
			message += fmt.Sprintf("%d. %s\n", i+1, tx.Description)
			message += fmt.Sprintf("   💰 Amount: %.2f AED\n", tx.Amount)
			message += fmt.Sprintf("   📁 Category: %s %s (%d%% confidence)\n", getCategoryEmoji(tx.Category), tx.Category, tx.Confidence)
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

func statsHandler(db *DatabaseClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[API] GET /stats - Request from %s", r.RemoteAddr)

		if r.Method != http.MethodGet {
			log.Printf("[API] Method not allowed: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		stats, err := db.GetStats()
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

func getCategoryEmoji(category string) string {
	emojis := map[string]string{
		"Food & Dining":     "🍔",
		"Transport":         "🚗",
		"Shopping":          "🛍️",
		"Bills & Utilities": "💳",
		"Entertainment":     "🎬",
		"Health & Fitness":  "💪",
		"Travel":            "✈️",
		"Cash Withdrawal":   "💵",
		"Income/Transfer":   "💰",
		"Unknown":           "❓",
	}
	if emoji, ok := emojis[category]; ok {
		return emoji
	}
	return "📌"
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboardHTML))
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
