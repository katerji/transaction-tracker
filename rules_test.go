package main

import (
	"testing"
)

func TestFindMatchingRule(t *testing.T) {
	db := setupTestDB(t)

	db.CreateRule("keywordA", "Groceries", 10)
	db.CreateRule("keywordB", "Transport", 5)

	tests := []struct {
		name     string
		desc     string
		expected string
	}{
		{
			name:     "exact match with higher priority",
			desc:     "keywordA City Centre",
			expected: "Groceries",
		},
		{
			name:     "partial match with lower priority",
			desc:     "keywordB Ride",
			expected: "Transport",
		},
		{
			name:     "no match returns nil",
			desc:     "Something completely different",
			expected: "",
		},
		{
			name:     "case insensitive match",
			desc:     "KEYWORDA in uppercase",
			expected: "Groceries",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rule, err := db.FindMatchingRule(tc.desc)
			if err != nil {
				t.Fatalf("FindMatchingRule failed: %v", err)
			}

			if tc.expected == "" {
				if rule != nil {
					t.Errorf("Expected nil, got rule: %v", rule)
				}
			} else {
				if rule == nil {
					t.Errorf("Expected rule with category %s, got nil", tc.expected)
				} else if rule.Category != tc.expected {
					t.Errorf("Expected category %s, got %s", tc.expected, rule.Category)
				}
			}
		})
	}
}

func TestApplyRuleSingle(t *testing.T) {
	db := setupTestDB(t)

	for _, tx := range []Transaction{
		{
			Description: "Carrefour Mall",
			Amount:      50.00,
			Date:        "2026-01-25",
			Category:    "Shopping",
			Confidence:  95,
			Source:      "openai",
		},
		{
			Description: "Carrefour Online",
			Amount:      75.00,
			Date:        "2026-01-24",
			Category:    "Shopping",
			Confidence:  90,
			Source:      "rule",
		},
		{
			Description: "Carrefour Supermarket",
			Amount:      100.00,
			Date:        "2026-01-23",
			Category:    "Shopping",
			Confidence:  100,
			Source:      "manual",
		},
	} {
		insertTestTransaction(t, db, tx)
	}

	rule, _, _, err := db.CreateRule("carrefour", "Groceries", 0)
	if err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}

	updated, protected, err := db.ApplyRuleSingle(rule.ID)
	if err != nil {
		t.Fatalf("ApplyRuleSingle failed: %v", err)
	}

	if updated != 2 {
		t.Errorf("Expected updated=2, got %d", updated)
	}
	if protected != 1 {
		t.Errorf("Expected protected=1, got %d", protected)
	}

	rows, _ := db.db.Query("SELECT id, category, source FROM transactions WHERE LOWER(description) LIKE '%carrefour%' ORDER BY id")
	defer rows.Close()

	var results []struct {
		id       int64
		category string
		source   string
	}
	for rows.Next() {
		var r struct {
			id       int64
			category string
			source   string
		}
		rows.Scan(&r.id, &r.category, &r.source)
		results = append(results, r)
	}

	if len(results) < 2 {
		t.Fatalf("Expected at least 2 results, got %d", len(results))
	}

	for i := 0; i < 2; i++ {
		if results[i].category != "Groceries" {
			t.Errorf("Transaction %d: expected category Groceries, got %s", i, results[i].category)
		}
		if results[i].source != "rule" {
			t.Errorf("Transaction %d: expected source rule, got %s", i, results[i].source)
		}
	}

	if results[2].category != "Shopping" || results[2].source != "manual" {
		t.Error("Manual transaction should not be updated")
	}
}

func TestApplyAllRules(t *testing.T) {
	db := setupTestDB(t)
	db.db.Exec("DELETE FROM merchant_rules")

	db.CreateRule("noon", "Shopping", 10)
	db.CreateRule("carrefour", "Groceries", 5)

	for _, tx := range []Transaction{
		{
			Description: "Noon Order",
			Amount:      100.00,
			Date:        "2026-01-25",
			Category:    "Unknown",
			Confidence:  50,
			Source:      "openai",
		},
		{
			Description: "Carrefour Mall",
			Amount:      150.00,
			Date:        "2026-01-24",
			Category:    "Unknown",
			Confidence:  50,
			Source:      "openai",
		},
		{
			Description: "Manual Entry",
			Amount:      50.00,
			Date:        "2026-01-23",
			Category:    "Unknown",
			Confidence:  100,
			Source:      "manual",
		},
	} {
		insertTestTransaction(t, db, tx)
	}

	updated, protected, err := db.ApplyAllRules()
	if err != nil {
		t.Fatalf("ApplyAllRules failed: %v", err)
	}

	if updated != 2 {
		t.Errorf("Expected updated=2, got %d", updated)
	}
	if protected != 1 {
		t.Errorf("Expected protected=1, got %d", protected)
	}

	var noonCat, carrefourCat, manualCat string
	db.db.QueryRow("SELECT category FROM transactions WHERE LOWER(description) LIKE '%noon%'").Scan(&noonCat)
	db.db.QueryRow("SELECT category FROM transactions WHERE LOWER(description) LIKE '%carrefour%'").Scan(&carrefourCat)
	db.db.QueryRow("SELECT category FROM transactions WHERE description = 'Manual Entry'").Scan(&manualCat)

	if noonCat != "Shopping" {
		t.Errorf("Expected Noon to be Shopping, got %s", noonCat)
	}
	if carrefourCat != "Groceries" {
		t.Errorf("Expected Carrefour to be Groceries, got %s", carrefourCat)
	}
	if manualCat != "Unknown" {
		t.Errorf("Expected Manual Entry to remain Unknown, got %s", manualCat)
	}
}

func TestApplyAllRulesFirstMatchWins(t *testing.T) {
	db := setupTestDB(t)
	db.db.Exec("DELETE FROM merchant_rules")

	db.CreateRule("carrefour market", "Groceries", 10)
	db.CreateRule("carrefour", "Shopping", 5)

	insertTestTransaction(t, db, Transaction{
		Description: "Carrefour Market Dubai",
		Amount:      200.00,
		Date:        "2026-01-25",
		Category:    "Unknown",
		Confidence:  50,
		Source:      "openai",
	})

	updated, _, err := db.ApplyAllRules()
	if err != nil {
		t.Fatalf("ApplyAllRules failed: %v", err)
	}

	if updated != 1 {
		t.Errorf("Expected updated=1, got %d", updated)
	}

	var cat string
	db.db.QueryRow("SELECT category FROM transactions WHERE description = 'Carrefour Market Dubai'").Scan(&cat)

	if cat != "Groceries" {
		t.Errorf("Expected higher priority rule (Groceries), got %s", cat)
	}
}

func TestRulePriorityMove(t *testing.T) {
	db := setupTestDB(t)

	db.db.Exec("DELETE FROM merchant_rules")

	r1, _, _, _ := db.CreateRule("rule1", "Groceries", 100)
	r2, _, _, _ := db.CreateRule("rule2", "Dining Out", 50)
	r3, _, _, _ := db.CreateRule("rule3", "Transport", 0)

	db.MoveRulePriority(r2.ID, "up")

	var p1, p2, p3 int
	db.db.QueryRow("SELECT priority FROM merchant_rules WHERE id = ?", r1.ID).Scan(&p1)
	db.db.QueryRow("SELECT priority FROM merchant_rules WHERE id = ?", r2.ID).Scan(&p2)
	db.db.QueryRow("SELECT priority FROM merchant_rules WHERE id = ?", r3.ID).Scan(&p3)

	if p1 != 50 || p2 != 100 {
		t.Errorf("After move up: r1=%d (expected 50), r2=%d (expected 100)", p1, p2)
	}

	db.MoveRulePriority(r1.ID, "down")
	db.db.QueryRow("SELECT priority FROM merchant_rules WHERE id = ?", r1.ID).Scan(&p1)
	db.db.QueryRow("SELECT priority FROM merchant_rules WHERE id = ?", r2.ID).Scan(&p2)
	db.db.QueryRow("SELECT priority FROM merchant_rules WHERE id = ?", r3.ID).Scan(&p3)

	if p1 != 0 || p3 != 50 {
		t.Errorf("After move down: r1=%d (expected 0), r3=%d (expected 50)", p1, p3)
	}

	err := db.MoveRulePriority(r3.ID, "down")
	if err != nil {
		t.Errorf("Move last rule down should be no-op, got error: %v", err)
	}
}

func TestCreateRuleReturnsMatchCounts(t *testing.T) {
	db := setupTestDB(t)

	for _, tx := range []Transaction{
		{Description: "Carrefour 1", Amount: 50.00, Date: "2026-01-25", Category: "Shopping", Confidence: 95, Source: "openai"},
		{Description: "Carrefour 2", Amount: 75.00, Date: "2026-01-24", Category: "Shopping", Confidence: 90, Source: "openai"},
		{Description: "Carrefour 3", Amount: 100.00, Date: "2026-01-23", Category: "Shopping", Confidence: 100, Source: "openai"},
		{Description: "Carrefour Manual", Amount: 60.00, Date: "2026-01-22", Category: "Shopping", Confidence: 100, Source: "manual"},
	} {
		insertTestTransaction(t, db, tx)
	}

	_, matchCount, protectedCount, err := db.CreateRule("carrefour", "Groceries", 0)
	if err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}

	if matchCount != 3 {
		t.Errorf("Expected matchCount=3, got %d", matchCount)
	}
	if protectedCount != 1 {
		t.Errorf("Expected protectedCount=1, got %d", protectedCount)
	}
}

