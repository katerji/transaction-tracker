package main

import (
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupTestDB(t *testing.T) *DatabaseClient {
	t.Helper()
	db, err := NewDatabaseClient(":memory:")
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func insertTestTransaction(t *testing.T, db *DatabaseClient, tx Transaction) {
	t.Helper()
	_, err := db.SaveTransaction(tx)
	if err != nil {
		t.Fatalf("failed to insert test transaction: %v", err)
	}
}

// --- DB layer tests ---

func TestGetAllTransactionsGroupedByCycle_Empty(t *testing.T) {
	db := setupTestDB(t)

	txs, err := db.GetAllTransactionsGroupedByCycle()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != 0 {
		t.Errorf("expected 0 transactions, got %d", len(txs))
	}
}

func TestGetAllTransactionsGroupedByCycle_MultipleCycles(t *testing.T) {
	db := setupTestDB(t)

	insertTestTransaction(t, db, Transaction{
		Description:  "Grocery Jan",
		Amount:       100,
		Date:         "2026-01-25",
		Category:     "Food & Dining",
		Confidence:   90,
		BillingCycle: "Jan 2026",
		Timestamp:    "2026-01-25T10:00:00Z",
	})
	insertTestTransaction(t, db, Transaction{
		Description:  "Grocery Feb",
		Amount:       200,
		Date:         "2026-02-15",
		Category:     "Food & Dining",
		Confidence:   90,
		BillingCycle: "Feb 2026",
		Timestamp:    "2026-02-15T10:00:00Z",
	})
	insertTestTransaction(t, db, Transaction{
		Description:  "Uber Dec",
		Amount:       50,
		Date:         "2025-12-24",
		Category:     "Transport",
		Confidence:   85,
		BillingCycle: "Dec 2025",
		Timestamp:    "2025-12-24T10:00:00Z",
	})

	txs, err := db.GetAllTransactionsGroupedByCycle()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != 3 {
		t.Fatalf("expected 3 transactions, got %d", len(txs))
	}

	// Verify newest first
	if txs[0].Description != "Grocery Feb" {
		t.Errorf("expected first tx to be 'Grocery Feb', got %q", txs[0].Description)
	}
	if txs[1].Description != "Grocery Jan" {
		t.Errorf("expected second tx to be 'Grocery Jan', got %q", txs[1].Description)
	}
	if txs[2].Description != "Uber Dec" {
		t.Errorf("expected third tx to be 'Uber Dec', got %q", txs[2].Description)
	}
}

func TestGetAllTransactionsGroupedByCycle_IncludesIncomeTransfer(t *testing.T) {
	db := setupTestDB(t)

	insertTestTransaction(t, db, Transaction{
		Description:  "Salary",
		Amount:       5000,
		Date:         "2026-01-25",
		Category:     "Income/Transfer",
		Confidence:   100,
		BillingCycle: "Jan 2026",
		Timestamp:    "2026-01-25T10:00:00Z",
	})
	insertTestTransaction(t, db, Transaction{
		Description:  "Coffee",
		Amount:       15,
		Date:         "2026-01-26",
		Category:     "Food & Dining",
		Confidence:   90,
		BillingCycle: "Jan 2026",
		Timestamp:    "2026-01-26T10:00:00Z",
	})

	txs, err := db.GetAllTransactionsGroupedByCycle()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txs))
	}

	hasIncome := false
	for _, tx := range txs {
		if tx.Category == "Income/Transfer" {
			hasIncome = true
		}
	}
	if !hasIncome {
		t.Error("expected Income/Transfer transaction to be included")
	}
}

// --- Handler tests ---

func TestExportHandler_CSVFormat(t *testing.T) {
	db := setupTestDB(t)

	insertTestTransaction(t, db, Transaction{
		Description:  "Carrefour Grocery",
		Amount:       145.50,
		Date:         "2026-02-20",
		Category:     "Food & Dining",
		Confidence:   90,
		BillingCycle: "Feb 2026",
		Timestamp:    "2026-02-20T14:30:00Z",
	})
	insertTestTransaction(t, db, Transaction{
		Description:  "Uber Ride",
		Amount:       35.00,
		Date:         "2026-02-18",
		Category:     "Transport",
		Confidence:   85,
		BillingCycle: "Feb 2026",
		Timestamp:    "2026-02-18T09:15:00Z",
	})
	insertTestTransaction(t, db, Transaction{
		Description:  "Salary",
		Amount:       10000,
		Date:         "2026-02-01",
		Category:     "Income/Transfer",
		Confidence:   100,
		BillingCycle: "Jan 2026",
		Timestamp:    "2026-02-01T08:00:00Z",
	})
	insertTestTransaction(t, db, Transaction{
		Description:  "Netflix",
		Amount:       54.99,
		Date:         "2026-01-30",
		Category:     "Entertainment",
		Confidence:   95,
		BillingCycle: "Jan 2026",
		Timestamp:    "2026-01-30T20:00:00Z",
	})

	req := httptest.NewRequest(http.MethodGet, "/export", nil)
	rec := httptest.NewRecorder()

	handler := exportHandler(db)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// Check headers
	ct := rec.Header().Get("Content-Type")
	if ct != "text/csv; charset=utf-8" {
		t.Errorf("expected Content-Type 'text/csv; charset=utf-8', got %q", ct)
	}
	cd := rec.Header().Get("Content-Disposition")
	if cd != `attachment; filename="transactions.csv"` {
		t.Errorf("expected Content-Disposition attachment, got %q", cd)
	}

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(rec.Body.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV: %v", err)
	}

	// Header row
	if records[0][0] != "Date" || records[0][1] != "Description" || records[0][2] != "Amount (AED)" || records[0][3] != "Category" {
		t.Errorf("unexpected header row: %v", records[0])
	}

	// Check cycle separator for Feb 2026
	if records[1][0] != "--- Feb 2026 ---" {
		t.Errorf("expected Feb 2026 cycle header, got %q", records[1][0])
	}

	// Verify Feb 2026 subtotal excludes Income/Transfer
	// Feb has: 145.50 + 35.00 = 180.50
	febSubtotalFound := false
	janSubtotalFound := false
	grandTotalFound := false

	for _, row := range records {
		if row[1] == "Subtotal" {
			switch {
			case !febSubtotalFound:
				febSubtotalFound = true
				if row[2] != "180.50" {
					t.Errorf("expected Feb subtotal 180.50, got %s", row[2])
				}
			case !janSubtotalFound:
				janSubtotalFound = true
				// Jan has: Netflix 54.99 (Salary is Income/Transfer, excluded)
				if row[2] != "54.99" {
					t.Errorf("expected Jan subtotal 54.99, got %s", row[2])
				}
			}
		}
		if row[1] == "Grand Total" {
			grandTotalFound = true
			if row[2] != "235.49" {
				t.Errorf("expected grand total 235.49, got %s", row[2])
			}
		}
	}

	if !febSubtotalFound {
		t.Error("Feb subtotal row not found")
	}
	if !janSubtotalFound {
		t.Error("Jan subtotal row not found")
	}
	if !grandTotalFound {
		t.Error("Grand Total row not found")
	}
}

func TestExportHandler_EmptyDB(t *testing.T) {
	db := setupTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/export", nil)
	rec := httptest.NewRecorder()

	handler := exportHandler(db)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	reader := csv.NewReader(strings.NewReader(rec.Body.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV: %v", err)
	}

	// Should have only the header row and the grand total row
	if len(records) != 2 {
		t.Errorf("expected 2 rows (header + grand total), got %d", len(records))
	}
	if records[0][0] != "Date" {
		t.Errorf("expected header row, got %v", records[0])
	}
	if records[1][1] != "Grand Total" || records[1][2] != "0.00" {
		t.Errorf("expected Grand Total 0.00, got %v", records[1])
	}
}

func TestExportHandler_MethodNotAllowed(t *testing.T) {
	db := setupTestDB(t)

	req := httptest.NewRequest(http.MethodPost, "/export", nil)
	rec := httptest.NewRecorder()

	handler := exportHandler(db)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}
}
