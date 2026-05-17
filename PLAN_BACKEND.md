# Backend Implementation Plan — Merchant Rules & Category Overhaul

## Overview
Implement a merchant rules engine that overrides OpenAI category assignments using a DB-backed keyword dictionary. Also update all category references to the new 11-category list and add a `source` field to transactions.

---

## Step 1 — Update Category List Everywhere

### 1a. `openai.go` — system prompt
Replace the category list in the `systemPrompt` const. Change the line that reads:
```
- category: exactly ONE of these categories: "Food & Dining", "Transport", "Shopping", "Bills & Utilities", "Entertainment", "Health & Fitness", "Travel", "Cash Withdrawal", "Income/Transfer", "Unknown"
```
To:
```
- category: exactly ONE of these categories: "Groceries", "Dining Out", "Transport", "Shopping", "Subscriptions", "Bills & Utilities", "Health", "Travel", "Entertainment", "Cash Withdrawal", "Income/Transfer"
```
Remove the `"Unknown"` category. Remove the rule `Only use "Unknown" category if confidence < 70`. Replace it with: `If confidence < 70, pick the closest matching category rather than leaving it uncategorized.`

### 1b. `main.go` — `getCategoryEmoji` function
Replace the entire `emojis` map with:
```go
emojis := map[string]string{
    "Groceries":         "🛒",
    "Dining Out":        "🍔",
    "Transport":         "🚗",
    "Shopping":          "🛍️",
    "Subscriptions":     "📱",
    "Bills & Utilities": "💳",
    "Health":            "💊",
    "Travel":            "✈️",
    "Entertainment":     "🎬",
    "Cash Withdrawal":   "💵",
    "Income/Transfer":   "💰",
}
```

---

## Step 2 — Add `source` Field to Transaction Struct

### 2a. `openai.go` — `Transaction` struct
Add field:
```go
Source string `json:"source,omitempty"`
```

---

## Step 3 — DB Migrations

### 3a. `database.go` — `runMigrations`
Append two new migration strings to the `migrations` slice (after the existing ones):

```go
`ALTER TABLE transactions ADD COLUMN source TEXT NOT NULL DEFAULT 'openai'`,
`CREATE TABLE IF NOT EXISTS merchant_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    keyword TEXT NOT NULL,
    category TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
)`,
`CREATE INDEX IF NOT EXISTS idx_rules_priority ON merchant_rules(priority DESC, id ASC)`,
```

**Important:** The `ALTER TABLE` migration will fail if the column already exists. Wrap it in a helper that ignores "duplicate column" errors:
```go
func (c *DatabaseClient) addColumnIfNotExists(table, colDef string) error {
    _, err := c.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, colDef))
    if err != nil && !strings.Contains(err.Error(), "duplicate column") {
        return err
    }
    return nil
}
```
Call this instead of `db.Exec` for the ALTER TABLE migration.

---

## Step 4 — MerchantRule Type & DB Operations

### 4a. New type in `database.go`
```go
type MerchantRule struct {
    ID        int64  `json:"id"`
    Keyword   string `json:"keyword"`
    Category  string `json:"category"`
    Priority  int    `json:"priority"`
    CreatedAt string `json:"createdAt"`
}
```

### 4b. New DB methods on `DatabaseClient`

**CreateRule** — inserts a rule, returns the new rule plus match/protected counts for existing transactions:
```go
func (c *DatabaseClient) CreateRule(keyword, category string, priority int) (*MerchantRule, int, int, error)
```
- INSERT INTO merchant_rules (keyword, category, priority, created_at) VALUES (?, ?, ?, ?)
- After insert, call `countRuleMatches(keyword)` to get match_count and protected_count
- Return the created rule, match_count, protected_count, error

**countRuleMatches** (private helper):
```go
func (c *DatabaseClient) countRuleMatches(keyword string) (matchCount int, protectedCount int, err error)
```
- `SELECT COUNT(*) FROM transactions WHERE LOWER(description) LIKE '%' || LOWER(?) || '%' AND source != 'manual'` → matchCount
- `SELECT COUNT(*) FROM transactions WHERE LOWER(description) LIKE '%' || LOWER(?) || '%' AND source = 'manual'` → protectedCount

**GetAllRules**:
```go
func (c *DatabaseClient) GetAllRules() ([]MerchantRule, error)
```
- `SELECT id, keyword, category, priority, created_at FROM merchant_rules ORDER BY priority DESC, id ASC`

**UpdateRule**:
```go
func (c *DatabaseClient) UpdateRule(id int64, keyword, category string, priority int) error
```
- `UPDATE merchant_rules SET keyword=?, category=?, priority=? WHERE id=?`
- Return "rule not found" error if RowsAffected == 0

**DeleteRule**:
```go
func (c *DatabaseClient) DeleteRule(id int64) error
```
- `DELETE FROM merchant_rules WHERE id=?`
- Return "rule not found" if RowsAffected == 0

**MoveRulePriority** — swaps priority with adjacent rule:
```go
func (c *DatabaseClient) MoveRulePriority(id int64, direction string) error
```
- Fetch all rules ordered by `priority DESC, id ASC`
- Find the index of the rule with the given id
- If direction == "up": swap priority with the rule at index-1 (if exists)
- If direction == "down": swap priority with the rule at index+1 (if exists)
- Execute two UPDATE statements to swap priority values
- If no adjacent rule exists, return nil (no-op)

**ApplyRuleSingle** — applies one rule retroactively:
```go
func (c *DatabaseClient) ApplyRuleSingle(ruleID int64) (updated int, protected int, err error)
```
- Fetch the rule by ID
- `UPDATE transactions SET category=?, source='rule' WHERE LOWER(description) LIKE '%' || LOWER(keyword) || '%' AND source != 'manual'`
- `updated` = RowsAffected
- `protected` = result of `SELECT COUNT(*) FROM transactions WHERE LOWER(description) LIKE '%' || LOWER(keyword) || '%' AND source = 'manual'`

**ApplyAllRules** — applies all rules with priority ordering (first match wins):
```go
func (c *DatabaseClient) ApplyAllRules() (updated int, protected int, err error)
```
- Fetch all rules ordered by `priority DESC, id ASC`
- Fetch all non-manual transactions: `SELECT id, description, source FROM transactions WHERE source != 'manual'`
- For each transaction, iterate rules in order. First rule whose keyword case-insensitively appears in description wins.
- Batch the updates: collect a map of `ruleCategory → []transactionIDs`
- Execute one UPDATE per category: `UPDATE transactions SET category=?, source='rule' WHERE id IN (...)`
- `updated` = total transactions updated
- `protected` = `SELECT COUNT(*) FROM transactions WHERE source = 'manual'`

**FindMatchingRule** — used in transaction parsing:
```go
func (c *DatabaseClient) FindMatchingRule(description string) (*MerchantRule, error)
```
- `SELECT id, keyword, category, priority, created_at FROM merchant_rules ORDER BY priority DESC, id ASC`
- Iterate: return the first rule where `strings.Contains(strings.ToLower(description), strings.ToLower(rule.Keyword))`
- Return nil, nil if no match

---

## Step 5 — Update Transaction Handlers in `main.go`

### 5a. `transactionHandler` — SMS parsing flow
After OpenAI returns transactions, for each `tx` before calling `enrichTransaction`:
```go
rule, err := db.FindMatchingRule(tx.Description)
if err == nil && rule != nil {
    tx.Category = rule.Category
    tx.Source = "rule"
} else {
    tx.Source = "openai"
}
```

### 5b. `manualTransactionHandler`
When constructing the `Transaction` before `enrichTransaction`, set:
```go
tx.Source = "manual"
```

### 5c. `updateTransactionHandler`
In `database.go` `UpdateTransaction`, add `source = 'manual'` to the UPDATE SET clause:
```go
UPDATE transactions SET description=?, amount=?, transaction_date=?, category=?, billing_cycle=?, source='manual' WHERE id=?
```

### 5d. `enrichTransaction` in `main.go`
No change needed — source is set before calling enrichTransaction.

### 5e. `SaveTransaction` in `database.go`
Update the INSERT query to include `source`:
```go
INSERT INTO transactions (description, amount, transaction_date, category, confidence, billing_cycle, created_at, source)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
```
Add `tx.Source` as the last argument to `db.Exec`.

---

## Step 6 — New API Endpoints

### 6a. New handler types in `main.go`

**`rulesHandler(db)`** — routes GET and POST on `/rules`:
```
GET  /rules       → db.GetAllRules() → JSON {success, rules: [...]}
POST /rules       → decode {keyword, category, priority} → db.CreateRule() → JSON {success, rule, match_count, protected_count}
```

**`ruleDetailHandler(db)`** — routes `/rules/` prefix:
- Parse ID from path `/rules/:id`
- Special case: if path is `/rules/apply-all`, route to applyAllHandler regardless of method
- PUT → updateRule handler
- DELETE → deleteRule handler
- POST on `/rules/:id/apply` → applyRuleSingle handler
- POST on `/rules/:id/move` → moveRulePriority handler

**Response shapes:**

`GET /rules`:
```json
{"success": true, "rules": [{"id":1,"keyword":"carrefour","category":"Groceries","priority":10,"createdAt":"..."}]}
```

`POST /rules`:
```json
{"success": true, "rule": {...}, "match_count": 15, "protected_count": 2}
```

`POST /rules/:id/apply`:
```json
{"success": true, "updated": 15, "protected": 2}
```

`POST /rules/apply-all`:
```json
{"success": true, "updated": 42, "protected": 5}
```

`PUT /rules/:id`:
```json
{"success": true}
```

`DELETE /rules/:id`:
```json
{"success": true}
```

`POST /rules/:id/move` body `{"direction":"up"}` or `{"direction":"down"}`:
```json
{"success": true}
```

### 6b. Register routes in `main.go` `main()` function
Add after existing `http.HandleFunc` calls:
```go
http.HandleFunc("/rules", rulesHandler(dbClient))
http.HandleFunc("/rules/", ruleDetailHandler(dbClient))
```

---

## Step 7 — Seed Data

Add a function `seedMerchantRules(db *DatabaseClient)` called once after migrations in `NewDatabaseClient`. It should only insert if the table is empty (`SELECT COUNT(*) FROM merchant_rules`). Seed these rules (priority 0 for all):

| Keyword | Category |
|---|---|
| carrefour | Groceries |
| spinneys | Groceries |
| lulu | Groceries |
| union coop | Groceries |
| talabat | Dining Out |
| deliveroo | Dining Out |
| zomato | Dining Out |
| starbucks | Dining Out |
| mcdonald | Dining Out |
| kfc | Dining Out |
| subway | Dining Out |
| hardee | Dining Out |
| pizza hut | Dining Out |
| careem | Transport |
| uber | Transport |
| salik | Transport |
| enoc | Transport |
| adnoc | Transport |
| emarat | Transport |
| noon | Shopping |
| amazon | Shopping |
| netflix | Subscriptions |
| spotify | Subscriptions |
| icloud | Subscriptions |
| chatgpt | Subscriptions |
| openai | Subscriptions |
| google one | Subscriptions |
| apple | Subscriptions |
| dewa | Bills & Utilities |
| addc | Bills & Utilities |
| etisalat | Bills & Utilities |
| du telecom | Bills & Utilities |
| aster | Health |
| boots | Health |
| medcare | Health |
| life pharmacy | Health |

---

## Step 8 — Tests

Create file `rules_test.go`.

### Test: `TestFindMatchingRule`
```
Setup: insert two rules — {keyword:"carrefour", category:"Groceries", priority:10} and {keyword:"car", category:"Transport", priority:5}
Test 1: FindMatchingRule("Carrefour City Centre") → should return Groceries (priority 10 wins)
Test 2: FindMatchingRule("Careem Ride") → should return Transport (matches "car" but not carrefour)
Test 3: FindMatchingRule("Starbucks") → should return nil (no match)
Test 4: FindMatchingRule("CARREFOUR") → should return Groceries (case-insensitive)
```

### Test: `TestApplyRuleSingle`
```
Setup: insert 3 transactions:
  - {description:"Carrefour Mall", source:"openai", category:"Shopping"}
  - {description:"Carrefour Online", source:"rule", category:"Shopping"}
  - {description:"Carrefour Supermarket", source:"manual", category:"Shopping"}
Insert rule {keyword:"carrefour", category:"Groceries"}
Call ApplyRuleSingle(ruleID)
Assert: updated=2, protected=1
Assert: first two transactions now have category="Groceries" and source="rule"
Assert: third transaction (manual) unchanged
```

### Test: `TestApplyAllRules`
```
Setup: insert rules:
  - {keyword:"noon", category:"Shopping", priority:10}
  - {keyword:"carrefour", category:"Groceries", priority:5}
Insert transactions:
  - {description:"Noon Order", source:"openai", category:"Unknown"}
  - {description:"Carrefour Mall", source:"openai", category:"Unknown"}
  - {description:"Manual Entry", source:"manual", category:"Unknown"}
Call ApplyAllRules()
Assert: updated=2, protected=1
Assert: "Noon Order" → Shopping, "Carrefour Mall" → Groceries
Assert: "Manual Entry" unchanged
```

### Test: `TestApplyAllRulesFirstMatchWins`
```
Setup: insert rules:
  - {keyword:"carrefour market", category:"Groceries", priority:10}
  - {keyword:"carrefour", category:"Shopping", priority:5}
Insert transaction: {description:"Carrefour Market Dubai", source:"openai"}
Call ApplyAllRules()
Assert: transaction category = "Groceries" (higher priority rule matched first)
```

### Test: `TestRulePriorityMove`
```
Setup: insert rules with priorities 10, 5, 0 (IDs 1, 2, 3)
Call MoveRulePriority(2, "up") → rule 2 should swap priority with rule 1 → priorities become 5, 10, 0
Call MoveRulePriority(1, "down") → rule 1 (now priority 5) should swap with rule 2 (priority 10) → back to 10, 5, 0
Call MoveRulePriority(3, "down") → no-op (already last), return nil
```

### Test: `TestCreateRuleReturnsMatchCounts`
```
Setup: insert 3 openai transactions with "carrefour" in description, 1 manual with "carrefour"
Call CreateRule("carrefour", "Groceries", 0)
Assert: match_count=3, protected_count=1
```

### Existing tests
Run `go test ./...` after changes to ensure `export_test.go` and `import_test.go` still pass. The `Transaction` struct change (adding `Source`) is backward-compatible since it uses `omitempty`.

---

## File Change Summary

| File | Changes |
|---|---|
| `openai.go` | Add `Source` to Transaction struct; update category list in system prompt |
| `main.go` | Update `getCategoryEmoji`; update `transactionHandler`, `manualTransactionHandler`; add `rulesHandler`, `ruleDetailHandler`; register new routes |
| `database.go` | Add migrations for `source` column and `merchant_rules` table; add `MerchantRule` type; add all new DB methods; update `SaveTransaction` and `UpdateTransaction` to handle `source`; add `seedMerchantRules` |
| `rules_test.go` | New file with all tests above |
