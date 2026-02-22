package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func createMultipartCSV(t *testing.T, csvContent string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "transactions.csv")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	part.Write([]byte(csvContent))
	writer.Close()
	return body, writer.FormDataContentType()
}

func TestImportHandler_CleanCSV(t *testing.T) {
	db := setupTestDB(t)
	handler := importHandler(db)

	csv := "Date,Description,Amount (AED),Category\n" +
		"2026-02-10,Grocery Store,150.00,Food & Dining\n" +
		"2026-02-11,Uber Ride,35.50,Transport\n" +
		"2026-02-12,Netflix,54.99,Entertainment\n"

	body, contentType := createMultipartCSV(t, csv)
	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ImportResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Imported != 3 {
		t.Errorf("expected 3 imported, got %d", resp.Imported)
	}
	if resp.Duplicates != 0 {
		t.Errorf("expected 0 duplicates, got %d", resp.Duplicates)
	}
	if len(resp.Errors) != 0 {
		t.Errorf("expected 0 errors, got %v", resp.Errors)
	}

	// Verify transactions are in the DB
	txs, err := db.GetAllTransactionsGroupedByCycle()
	if err != nil {
		t.Fatalf("failed to get transactions: %v", err)
	}
	if len(txs) != 3 {
		t.Errorf("expected 3 transactions in DB, got %d", len(txs))
	}
}

func TestImportHandler_ExportFormat(t *testing.T) {
	db := setupTestDB(t)
	handler := importHandler(db)

	csv := "Date,Description,Amount (AED),Category\n" +
		"--- Feb 2026 ---,,,\n" +
		"2026-02-10,Grocery Store,150.00,Food & Dining\n" +
		"2026-02-11,Uber Ride,35.50,Transport\n" +
		",Subtotal,185.50,\n" +
		",,,\n" +
		"--- Jan 2026 ---,,,\n" +
		"2026-01-25,Netflix,54.99,Entertainment\n" +
		",Subtotal,54.99,\n" +
		",,,\n" +
		",Grand Total,240.49,\n"

	body, contentType := createMultipartCSV(t, csv)
	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ImportResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Imported != 3 {
		t.Errorf("expected 3 imported, got %d", resp.Imported)
	}
	if resp.Duplicates != 0 {
		t.Errorf("expected 0 duplicates, got %d", resp.Duplicates)
	}

	txs, err := db.GetAllTransactionsGroupedByCycle()
	if err != nil {
		t.Fatalf("failed to get transactions: %v", err)
	}
	if len(txs) != 3 {
		t.Errorf("expected 3 transactions in DB, got %d", len(txs))
	}
}

func TestImportHandler_Duplicates(t *testing.T) {
	db := setupTestDB(t)

	// Pre-insert a transaction
	insertTestTransaction(t, db, Transaction{
		Description:  "Grocery Store",
		Amount:       150.00,
		Date:         "2026-02-10",
		Category:     "Food & Dining",
		Confidence:   100,
		BillingCycle: "Jan 2026",
		Timestamp:    "2026-02-10T10:00:00Z",
	})

	handler := importHandler(db)

	csv := "Date,Description,Amount (AED),Category\n" +
		"2026-02-10,Grocery Store,150.00,Food & Dining\n" +
		"2026-02-11,Uber Ride,35.50,Transport\n"

	body, contentType := createMultipartCSV(t, csv)
	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ImportResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", resp.Imported)
	}
	if resp.Duplicates != 1 {
		t.Errorf("expected 1 duplicate, got %d", resp.Duplicates)
	}
}

func TestImportHandler_InvalidRows(t *testing.T) {
	db := setupTestDB(t)
	handler := importHandler(db)

	csv := "Date,Description,Amount (AED),Category\n" +
		"2026-02-10,Grocery Store,abc,Food & Dining\n" +
		"2026-02-11,Uber Ride,35.50,Transport\n"

	body, contentType := createMultipartCSV(t, csv)
	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ImportResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", resp.Imported)
	}
	if len(resp.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(resp.Errors))
	}
	if resp.Errors[0] != "row 2: invalid amount 'abc'" {
		t.Errorf("unexpected error message: %s", resp.Errors[0])
	}
}

func TestImportHandler_MethodNotAllowed(t *testing.T) {
	db := setupTestDB(t)
	handler := importHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/import", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}
}

func TestImportHandler_NoFile(t *testing.T) {
	db := setupTestDB(t)
	handler := importHandler(db)

	// POST with empty multipart form (no file)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
