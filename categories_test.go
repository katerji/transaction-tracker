package main

import (
	"testing"
)

func TestGetAllCategories_ReturnsSeededDefaults(t *testing.T) {
	db := setupTestDB(t)
	cats, err := db.GetAllCategories()
	if err != nil {
		t.Fatalf("GetAllCategories failed: %v", err)
	}
	if len(cats) != 11 {
		t.Errorf("expected 11 seeded categories, got %d", len(cats))
	}
}

func TestCreateCategory_Success(t *testing.T) {
	db := setupTestDB(t)
	cat, err := db.CreateCategory("Fitness", "🏋️", false)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	if cat.Name != "Fitness" {
		t.Errorf("expected name Fitness, got %s", cat.Name)
	}
	if cat.Emoji != "🏋️" {
		t.Errorf("expected emoji 🏋️, got %s", cat.Emoji)
	}
	if cat.ExcludeFromTotals != false {
		t.Errorf("expected excludeFromTotals false")
	}
}

func TestCreateCategory_DuplicateName(t *testing.T) {
	db := setupTestDB(t)
	_, err := db.CreateCategory("Groceries", "🛒", false)
	if err == nil {
		t.Error("expected error for duplicate category name, got nil")
	}
}

func TestUpdateCategory_CascadesRename(t *testing.T) {
	db := setupTestDB(t)

	// Insert a transaction and a rule using "Groceries"
	_, err := db.SaveTransaction(Transaction{
		Description:  "Test TX",
		Amount:       50,
		Date:         "2026-01-01",
		Category:     "Groceries",
		Confidence:   90,
		BillingCycle: "Dec 2025",
		Timestamp:    "2026-01-01T00:00:00Z",
		Source:       "openai",
	})
	if err != nil {
		t.Fatalf("SaveTransaction failed: %v", err)
	}
	_, _, _, err = db.CreateRule("carrefour", "Groceries", 10)
	if err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}

	// Find Groceries category ID
	cats, _ := db.GetAllCategories()
	var grocID int64
	for _, c := range cats {
		if c.Name == "Groceries" {
			grocID = c.ID
			break
		}
	}

	// Rename Groceries → Food
	if err := db.UpdateCategory(grocID, "Food", "🥘", false); err != nil {
		t.Fatalf("UpdateCategory failed: %v", err)
	}

	// Verify transaction was updated
	rule, err := db.FindMatchingRule("carrefour")
	if err != nil {
		t.Fatalf("FindMatchingRule failed: %v", err)
	}
	if rule == nil || rule.Category != "Food" {
		t.Errorf("expected merchant rule category to be Food, got %v", rule)
	}
}

func TestDeleteCategory_BlockedIfTransactionsExist(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.SaveTransaction(Transaction{
		Description:  "Test TX",
		Amount:       50,
		Date:         "2026-01-01",
		Category:     "Shopping",
		Confidence:   90,
		BillingCycle: "Dec 2025",
		Timestamp:    "2026-01-01T00:00:00Z",
		Source:       "openai",
	})
	if err != nil {
		t.Fatalf("SaveTransaction failed: %v", err)
	}

	cats, _ := db.GetAllCategories()
	var shopID int64
	for _, c := range cats {
		if c.Name == "Shopping" {
			shopID = c.ID
			break
		}
	}

	err = db.DeleteCategory(shopID)
	if err == nil {
		t.Error("expected error when deleting category with active transactions, got nil")
	}
}

func TestDeleteCategory_Success(t *testing.T) {
	db := setupTestDB(t)

	cat, err := db.CreateCategory("Temporary", "🗑️", false)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	if err := db.DeleteCategory(cat.ID); err != nil {
		t.Fatalf("DeleteCategory failed: %v", err)
	}

	cats, _ := db.GetAllCategories()
	for _, c := range cats {
		if c.Name == "Temporary" {
			t.Error("expected category to be deleted, but it still exists")
		}
	}
}

func TestDeleteCategory_NotFound(t *testing.T) {
	db := setupTestDB(t)
	err := db.DeleteCategory(99999)
	if err == nil {
		t.Error("expected error for nonexistent category ID, got nil")
	}
}
