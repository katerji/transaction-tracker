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
	// 11 seeded + 8 new from data migrations - 1 Travel (deleted, no transactions) = 18
	if len(cats) != 18 {
		t.Errorf("expected 18 categories after migrations, got %d", len(cats))
	}
}

func TestCreateCategory_Success(t *testing.T) {
	db := setupTestDB(t)
	cat, err := db.CreateCategory("Fitness", "🏋️", false, "wants", nil)
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
	_, err := db.CreateCategory("Groceries", "🛒", false, "wants", nil)
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
	if err := db.UpdateCategory(grocID, "Food", "🥘", false, "wants", nil); err != nil {
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

func TestDeleteCategory_AllowedWithTransactions(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.SaveTransaction(Transaction{
		Description:  "Test TX",
		Amount:       50,
		Date:         "2026-01-01",
		Category:     "Shopping & Gifts",
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
		if c.Name == "Shopping & Gifts" {
			shopID = c.ID
			break
		}
	}

	if err = db.DeleteCategory(shopID); err != nil {
		t.Errorf("expected deletion to succeed even with transactions, got: %v", err)
	}
}

func TestDeleteCategory_Success(t *testing.T) {
	db := setupTestDB(t)

	cat, err := db.CreateCategory("Temporary", "🗑️", false, "wants", nil)
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

func TestSetCategoryTarget_SetEditAndRemove(t *testing.T) {
	db := setupTestDB(t)

	cat, err := db.CreateCategory("Shopping Target", "🛍️", false, "wants", nil)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	if cat.BudgetAmount != nil {
		t.Fatalf("expected new category to have no target, got %v", *cat.BudgetAmount)
	}

	// Set a target.
	if err := db.SetCategoryTarget(cat.ID, floatPtr(2000)); err != nil {
		t.Fatalf("SetCategoryTarget (set) failed: %v", err)
	}
	if got := targetOf(t, db, cat.ID); got == nil || *got != 2000 {
		t.Errorf("expected target 2000 after set, got %v", got)
	}

	// Edit the target.
	if err := db.SetCategoryTarget(cat.ID, floatPtr(1500)); err != nil {
		t.Fatalf("SetCategoryTarget (edit) failed: %v", err)
	}
	if got := targetOf(t, db, cat.ID); got == nil || *got != 1500 {
		t.Errorf("expected target 1500 after edit, got %v", got)
	}

	// Remove the target.
	if err := db.SetCategoryTarget(cat.ID, nil); err != nil {
		t.Fatalf("SetCategoryTarget (remove) failed: %v", err)
	}
	if got := targetOf(t, db, cat.ID); got != nil {
		t.Errorf("expected no target after remove, got %v", *got)
	}
}

func TestSetCategoryTarget_NotFound(t *testing.T) {
	db := setupTestDB(t)
	if err := db.SetCategoryTarget(99999, floatPtr(100)); err == nil {
		t.Error("expected error for nonexistent category ID, got nil")
	}
}

// targetOf returns the current budget_amount for a category by ID.
func targetOf(t *testing.T, db *DatabaseClient, id int64) *float64 {
	t.Helper()
	cats, err := db.GetAllCategories()
	if err != nil {
		t.Fatalf("GetAllCategories failed: %v", err)
	}
	for _, c := range cats {
		if c.ID == id {
			return c.BudgetAmount
		}
	}
	t.Fatalf("category %d not found", id)
	return nil
}
