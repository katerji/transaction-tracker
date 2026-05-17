# Frontend Implementation Plan — Merchant Rules UI & Category Overhaul

## Overview
Add a Rules tab (4th tab), update the edit modal to a 3-step flow for auto-learning rules, update all category references to the new 11-category list, and add a toast after retroactive application.

The app uses Alpine.js (via CDN), vanilla JS modules, and Tailwind CSS. The entry point is `static/index.html`. App state lives in `static/js/app.js`. API calls are in `static/js/api.js`.

---

## Step 1 — Update Category List

### 1a. `static/js/app.js` — `categoryOptions` array
Replace the existing array:
```js
categoryOptions: [
  'Groceries',
  'Dining Out',
  'Transport',
  'Shopping',
  'Subscriptions',
  'Bills & Utilities',
  'Health',
  'Travel',
  'Entertainment',
  'Cash Withdrawal',
  'Income/Transfer',
],
```
Also update the default values:
- `editForm` default category: `'Groceries'`
- `addForm` default category: `'Groceries'`

### 1b. `static/js/utils.js` — `getCategoryEmoji` function
Locate the function and replace the emoji map with:
```js
const map = {
  'Groceries':         '🛒',
  'Dining Out':        '🍔',
  'Transport':         '🚗',
  'Shopping':          '🛍️',
  'Subscriptions':     '📱',
  'Bills & Utilities': '💳',
  'Health':            '💊',
  'Travel':            '✈️',
  'Entertainment':     '🎬',
  'Cash Withdrawal':   '💵',
  'Income/Transfer':   '💰',
};
return map[category] || '📌';
```

---

## Step 2 — New API Functions

### `static/js/api.js` — append these exports:

```js
export async function fetchRules() {
  const res = await fetch('/rules');
  if (!res.ok) throw new Error('Failed to fetch rules');
  return res.json();
}

export async function createRule(data) {
  const res = await fetch('/rules', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!res.ok) throw new Error('Failed to create rule');
  return res.json();
}

export async function updateRule(id, data) {
  const res = await fetch('/rules/' + id, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!res.ok) throw new Error('Failed to update rule');
  return res.json();
}

export async function deleteRule(id) {
  const res = await fetch('/rules/' + id, { method: 'DELETE' });
  if (!res.ok) throw new Error('Failed to delete rule');
  return res.json();
}

export async function applyRuleSingle(id) {
  const res = await fetch('/rules/' + id + '/apply', { method: 'POST' });
  if (!res.ok) throw new Error('Failed to apply rule');
  return res.json();
}

export async function applyAllRules() {
  const res = await fetch('/rules/apply-all', { method: 'POST' });
  if (!res.ok) throw new Error('Failed to apply all rules');
  return res.json();
}

export async function moveRulePriority(id, direction) {
  const res = await fetch('/rules/' + id + '/move', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ direction }),
  });
  if (!res.ok) throw new Error('Failed to move rule');
  return res.json();
}
```

---

## Step 3 — New `rules.js` Tab Module

Create `static/js/tabs/rules.js`:
```js
// Rules tab — no logic needed here (all state in app.js)
// Exported as placeholder for future tab-specific helpers if needed.
export {};
```

---

## Step 4 — Update `static/js/app.js`

### 4a. Add imports at the top
Add to the existing import from `./api.js`:
```js
import { fetchRules, createRule, updateRule, deleteRule, applyRuleSingle, applyAllRules, moveRulePriority } from './api.js';
```

### 4b. New state properties (add inside the returned object, after existing state)

```js
// Rules tab
rulesLoading: false,
rules: [],

// Edit modal multi-step
editStep: 'edit', // 'edit' | 'confirm_rule' | 'confirm_retroactive'
editRuleKeyword: '',
editRuleMatchCount: 0,
editRuleProtectedCount: 0,
editRuleSavedId: null,

// Rule add modal
ruleAddOpen: false,
ruleAddForm: { keyword: '', category: 'Groceries', priority: 0 },

// Rule edit modal
ruleEditOpen: false,
ruleEditId: null,
ruleEditForm: { keyword: '', category: 'Groceries', priority: 0 },

// Rule delete confirm
ruleDeleteOpen: false,
ruleDeleteId: null,
ruleDeleteKeyword: '',

// Apply-all confirm modal
applyAllOpen: false,
applyAllLoading: false,
```

### 4c. New methods (add inside the returned object)

**Rules tab loading:**
```js
async loadRules() {
  this.rulesLoading = true;
  try {
    const data = await fetchRules();
    this.rules = data.rules || [];
  } catch (e) {
    this.showToast('Error loading rules: ' + e.message);
  } finally {
    this.rulesLoading = false;
  }
},
```

**Tab switching — update `switchTab`:**
In the existing `switchTab(tab)` method, add before `this.currentTab = tab`:
```js
if (tab === 'rules') {
  this.loadRules();
}
```

**Rule add modal:**
```js
openRuleAdd() {
  this.ruleAddForm = { keyword: '', category: 'Groceries', priority: 0 };
  this.ruleAddOpen = true;
  hapticFeedback('light');
},
closeRuleAdd() {
  this.ruleAddOpen = false;
},
async saveRuleAdd() {
  const keyword = this.ruleAddForm.keyword.trim();
  if (!keyword) return this.showToast('Keyword is required');
  try {
    const result = await createRule({
      keyword,
      category: this.ruleAddForm.category,
      priority: parseInt(this.ruleAddForm.priority) || 0,
    });
    this.rules.unshift(result.rule);
    this.closeRuleAdd();
    hapticFeedback('success');
    this.showToast('Rule created');
    await this.loadRules();
  } catch (e) {
    this.showToast('Error: ' + e.message);
  }
},
```

**Rule edit modal:**
```js
openRuleEdit(rule) {
  this.ruleEditId = rule.id;
  this.ruleEditForm = { keyword: rule.keyword, category: rule.category, priority: rule.priority };
  this.ruleEditOpen = true;
  hapticFeedback('light');
},
closeRuleEdit() {
  this.ruleEditOpen = false;
  this.ruleEditId = null;
},
async saveRuleEdit() {
  if (!this.ruleEditId) return;
  try {
    await updateRule(this.ruleEditId, this.ruleEditForm);
    this.closeRuleEdit();
    hapticFeedback('success');
    this.showToast('Rule updated');
    await this.loadRules();
  } catch (e) {
    this.showToast('Error: ' + e.message);
  }
},
```

**Rule delete:**
```js
openRuleDelete(rule) {
  this.ruleDeleteId = rule.id;
  this.ruleDeleteKeyword = rule.keyword;
  this.ruleDeleteOpen = true;
  hapticFeedback('medium');
},
closeRuleDelete() {
  this.ruleDeleteOpen = false;
  this.ruleDeleteId = null;
},
async confirmRuleDelete() {
  if (!this.ruleDeleteId) return;
  try {
    await deleteRule(this.ruleDeleteId);
    this.closeRuleDelete();
    hapticFeedback('success');
    this.showToast('Rule deleted');
    await this.loadRules();
  } catch (e) {
    this.showToast('Error: ' + e.message);
  }
},
```

**Priority move:**
```js
async moveRule(id, direction) {
  try {
    await moveRulePriority(id, direction);
    await this.loadRules();
    hapticFeedback('light');
  } catch (e) {
    this.showToast('Error: ' + e.message);
  }
},
```

**Apply-all modal:**
```js
openApplyAll() {
  this.applyAllOpen = true;
  hapticFeedback('medium');
},
closeApplyAll() {
  this.applyAllOpen = false;
},
async confirmApplyAll() {
  this.applyAllLoading = true;
  try {
    const result = await applyAllRules();
    this.closeApplyAll();
    hapticFeedback('success');
    this.showToast('Updated ' + result.updated + ' transactions (' + result.protected + ' protected)');
    await this.loadStats();
  } catch (e) {
    this.showToast('Error: ' + e.message);
  } finally {
    this.applyAllLoading = false;
  }
},
```

### 4d. Update `openEdit` method
After setting `this.editOpen = true`, also reset the step state:
```js
this.editStep = 'edit';
this.editRuleKeyword = '';
this.editRuleMatchCount = 0;
this.editRuleProtectedCount = 0;
this.editRuleSavedId = null;
```

### 4e. Update `closeEdit` method
Add after `this.editId = null`:
```js
this.editStep = 'edit';
this.editRuleKeyword = '';
this.editRuleSavedId = null;
```

### 4f. Update `saveEdit` method
Replace the existing `saveEdit` with this new version:
```js
async saveEdit() {
  if (!this.editId) return;
  const payload = {
    description: this.editForm.description,
    amount: parseFloat(this.editForm.amount),
    date: inputValueToDate(this.editForm.date),
    category: this.editForm.category,
  };
  try {
    await updateTransaction(this.editId, payload);
    const txId = this.editId;
    const oldTx = this.editOld;
    this.applyUpdate(txId, oldTx, payload);
    hapticFeedback('success');
    this.showToast('Transaction updated');

    const categoryChanged = oldTx.category !== payload.category;
    if (categoryChanged) {
      // Suggest keyword: first meaningful word of description (lowercase)
      this.editRuleKeyword = payload.description.trim().split(/\s+/)[0].toLowerCase();
      this.editStep = 'confirm_rule';
    } else {
      this.closeEdit();
    }
  } catch (e) {
    this.showToast('Error: ' + e.message);
  }
},
```

### 4g. New methods for edit modal steps
```js
skipRuleStep() {
  this.closeEdit();
},

async saveEditRule() {
  const keyword = this.editRuleKeyword.trim();
  if (!keyword) return this.showToast('Keyword is required');
  try {
    const result = await createRule({
      keyword,
      category: this.editForm.category,
      priority: 0,
    });
    this.editRuleSavedId = result.rule.id;
    this.editRuleMatchCount = result.match_count;
    this.editRuleProtectedCount = result.protected_count;
    this.editStep = 'confirm_retroactive';
    hapticFeedback('light');
  } catch (e) {
    this.showToast('Error saving rule: ' + e.message);
  }
},

skipRetroactive() {
  this.closeEdit();
},

async applyRetroactive() {
  if (!this.editRuleSavedId) return;
  try {
    const result = await applyRuleSingle(this.editRuleSavedId);
    this.closeEdit();
    hapticFeedback('success');
    this.showToast('Updated ' + result.updated + ' transactions (' + result.protected + ' protected)');
    await this.loadStats();
  } catch (e) {
    this.showToast('Error: ' + e.message);
  }
},
```

### 4h. Update `initModalSwipe` to handle `rulesOpen` modals
In the `touchend` handler inside `initModalSwipe`, add `this.ruleAddOpen = false`, `this.ruleEditOpen = false`, `this.ruleDeleteOpen = false`, `this.applyAllOpen = false` alongside the existing assignments.

---

## Step 5 — Update `static/index.html`

### 5a. Add Rules tab content (4th panel)
Add after the closing `</div>` of the Categories tab panel (`x-show="currentTab === 'categories'"`):

```html
<!-- ==================== RULES TAB ==================== -->
<div x-show="currentTab === 'rules' && !rulesLoading"
     x-transition:enter="transition ease-out duration-200"
     x-transition:enter-start="opacity-0"
     x-transition:enter-end="opacity-100"
     class="px-4 pt-4 pb-36">
  <div class="flex items-center justify-between mb-4">
    <h2 class="text-lg font-semibold text-gray-100">Merchant Rules</h2>
  </div>

  <!-- Empty state -->
  <template x-if="rules.length === 0">
    <div class="text-center text-gray-500 text-sm py-12">
      No rules yet. Add one below or edit a transaction category to create one.
    </div>
  </template>

  <!-- Rules list -->
  <div class="space-y-2 mb-6">
    <template x-for="(rule, idx) in rules" :key="rule.id">
      <div class="bg-card border border-card-border rounded-xl px-4 py-3 flex items-center gap-3">
        <!-- Priority arrows -->
        <div class="flex flex-col gap-0.5">
          <button @click="moveRule(rule.id, 'up')"
                  class="text-gray-500 hover:text-gray-200 transition leading-none text-xs"
                  :disabled="idx === 0">▲</button>
          <button @click="moveRule(rule.id, 'down')"
                  class="text-gray-500 hover:text-gray-200 transition leading-none text-xs"
                  :disabled="idx === rules.length - 1">▼</button>
        </div>

        <!-- Rule info -->
        <div class="flex-1 min-w-0">
          <div class="text-sm font-medium text-gray-100 truncate" x-text="rule.keyword"></div>
          <div class="text-xs text-gray-400 flex items-center gap-1">
            <span x-text="getCategoryEmoji(rule.category)"></span>
            <span x-text="rule.category"></span>
          </div>
        </div>

        <!-- Actions -->
        <div class="flex gap-2">
          <button @click="openRuleEdit(rule)"
                  class="text-xs px-2 py-1 bg-gray-700 text-gray-300 rounded-lg hover:bg-gray-600 transition">Edit</button>
          <button @click="openRuleDelete(rule)"
                  class="text-xs px-2 py-1 bg-red-900/40 text-red-400 rounded-lg hover:bg-red-900/70 transition">Del</button>
        </div>
      </div>
    </template>
  </div>

  <!-- Loading state -->
  <template x-if="rulesLoading">
    <div class="text-center text-gray-500 text-sm py-12">Loading rules...</div>
  </template>
</div>

<!-- Rules tab sticky footer -->
<div x-show="currentTab === 'rules'"
     class="fixed bottom-16 left-0 right-0 px-4 pb-2 z-40">
  <button @click="openApplyAll()"
          class="w-full py-3 bg-surface border border-card-border text-gray-300 rounded-xl text-sm font-semibold hover:bg-gray-700 transition">
    Re-apply All Rules
  </button>
</div>

<!-- Rules FAB -->
<button x-show="currentTab === 'rules'"
        @click="openRuleAdd()"
        class="fixed bottom-20 right-4 z-50 w-12 h-12 bg-primary rounded-full shadow-lg flex items-center justify-center text-white text-xl hover:bg-primary-dark transition">
  +
</button>
```

### 5b. Add Rules tab button to bottom nav
After the Categories tab button in the bottom nav, add:
```html
<button @click="switchTab('rules')"
        class="flex flex-col items-center justify-center flex-1 py-2 transition-colors"
        :class="currentTab === 'rules' ? 'text-primary' : 'text-gray-500'">
  <svg class="w-5 h-5 mb-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
          d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"/>
  </svg>
  <span class="text-[10px] font-medium">Rules</span>
</button>
```

### 5c. Update Edit modal to support 3 steps
Replace the inner content of the Edit modal (between the drag handle and the closing `</div>` of the modal sheet) with:

```html
<!-- Step: edit -->
<div x-show="editStep === 'edit'">
  <div class="text-xl font-semibold text-gray-100 mb-5 modal-header">Edit Transaction</div>

  <!-- Source badge -->
  <div class="mb-4 text-xs text-gray-500" x-show="editOld && editOld.source">
    Categorized by:
    <span class="capitalize font-medium text-gray-400" x-text="editOld && editOld.source === 'rule' ? 'Merchant Rule' : editOld && editOld.source === 'manual' ? 'You' : 'AI'"></span>
  </div>

  <div class="space-y-4">
    <div>
      <label class="block text-sm font-medium text-gray-400 mb-1.5">Description</label>
      <input x-model="editForm.description" type="text" placeholder="Transaction description"
             class="w-full px-3 py-2.5 border-2 border-card-border bg-surface rounded-xl text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:border-primary transition">
    </div>
    <div>
      <label class="block text-sm font-medium text-gray-400 mb-1.5">Amount (AED)</label>
      <input x-model="editForm.amount" type="number" step="0.01" placeholder="0.00"
             class="w-full px-3 py-2.5 border-2 border-card-border bg-surface rounded-xl text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:border-primary transition">
    </div>
    <div>
      <label class="block text-sm font-medium text-gray-400 mb-1.5">Date & Time</label>
      <input x-model="editForm.date" type="datetime-local"
             class="w-full px-3 py-2.5 border-2 border-card-border bg-surface rounded-xl text-sm text-gray-100 focus:outline-none focus:border-primary transition">
    </div>
    <div>
      <label class="block text-sm font-medium text-gray-400 mb-1.5">Category</label>
      <select x-model="editForm.category"
              class="w-full px-3 py-2.5 border-2 border-card-border bg-surface rounded-xl text-sm text-gray-100 focus:outline-none focus:border-primary transition appearance-none">
        <template x-for="opt in categoryOptions" :key="opt">
          <option :value="opt" x-text="getCategoryEmoji(opt) + ' ' + opt"></option>
        </template>
      </select>
    </div>
  </div>
  <div class="flex gap-3 mt-6">
    <button @click="closeEdit()" class="flex-1 py-3 bg-gray-700 text-gray-300 rounded-xl text-sm font-semibold hover:bg-gray-600 transition">Cancel</button>
    <button @click="saveEdit()" class="flex-1 py-3 bg-primary text-white rounded-xl text-sm font-semibold hover:bg-primary-dark transition">Save Changes</button>
  </div>
</div>

<!-- Step: confirm_rule -->
<div x-show="editStep === 'confirm_rule'">
  <div class="text-xl font-semibold text-gray-100 mb-2 modal-header">Save as Rule?</div>
  <p class="text-sm text-gray-400 mb-5">
    You changed the category to <span class="text-gray-100 font-medium" x-text="getCategoryEmoji(editForm.category) + ' ' + editForm.category"></span>.
    Save a rule to auto-categorize similar transactions in future?
  </p>
  <div class="mb-5">
    <label class="block text-sm font-medium text-gray-400 mb-1.5">Keyword to match</label>
    <input x-model="editRuleKeyword" type="text" placeholder="e.g. carrefour"
           class="w-full px-3 py-2.5 border-2 border-card-border bg-surface rounded-xl text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:border-primary transition">
    <p class="text-xs text-gray-500 mt-1">Case-insensitive. Matches any description containing this keyword.</p>
  </div>
  <div class="flex gap-3">
    <button @click="skipRuleStep()" class="flex-1 py-3 bg-gray-700 text-gray-300 rounded-xl text-sm font-semibold hover:bg-gray-600 transition">Skip</button>
    <button @click="saveEditRule()" class="flex-1 py-3 bg-primary text-white rounded-xl text-sm font-semibold hover:bg-primary-dark transition">Save Rule</button>
  </div>
</div>

<!-- Step: confirm_retroactive -->
<div x-show="editStep === 'confirm_retroactive'">
  <div class="text-xl font-semibold text-gray-100 mb-2 modal-header">Apply Retroactively?</div>
  <div class="bg-surface border border-card-border rounded-xl p-4 mb-5 space-y-1">
    <p class="text-sm text-gray-100">
      <span class="font-semibold" x-text="editRuleMatchCount"></span>
      existing transactions match this rule and will be updated.
    </p>
    <p class="text-xs text-gray-500">
      <span x-text="editRuleProtectedCount"></span> manually-edited transactions are protected and will not be changed.
    </p>
  </div>
  <div class="flex gap-3">
    <button @click="skipRetroactive()" class="flex-1 py-3 bg-gray-700 text-gray-300 rounded-xl text-sm font-semibold hover:bg-gray-600 transition">Skip</button>
    <button @click="applyRetroactive()" class="flex-1 py-3 bg-primary text-white rounded-xl text-sm font-semibold hover:bg-primary-dark transition">Apply</button>
  </div>
</div>
```

### 5d. Add Rule Add modal (before closing `</body>`)
```html
<!-- ==================== RULE ADD MODAL ==================== -->
<div x-show="ruleAddOpen" x-transition.opacity
     @click.self="closeRuleAdd()"
     class="fixed inset-0 bg-black/70 z-[1000] flex items-end md:items-center justify-center">
  <div x-show="ruleAddOpen"
       x-transition:enter="transition ease-out duration-300"
       x-transition:enter-start="translate-y-full md:translate-y-0 md:scale-95 md:opacity-0"
       x-transition:enter-end="translate-y-0 md:scale-100 md:opacity-100"
       x-init="initModalSwipe($el)"
       class="bg-card rounded-t-2xl md:rounded-2xl p-6 w-full md:max-w-[500px] md:w-[90%] shadow-2xl relative modal-sheet">
    <div class="w-10 h-1 bg-gray-600 rounded-full mx-auto -mt-2 mb-5 md:hidden cursor-grab active:cursor-grabbing modal-drag-handle"></div>
    <div class="text-xl font-semibold text-gray-100 mb-5 modal-header">Add Rule</div>
    <div class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-gray-400 mb-1.5">Keyword</label>
        <input x-model="ruleAddForm.keyword" type="text" placeholder="e.g. carrefour"
               class="w-full px-3 py-2.5 border-2 border-card-border bg-surface rounded-xl text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:border-primary transition">
        <p class="text-xs text-gray-500 mt-1">Case-insensitive substring match against transaction descriptions.</p>
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-400 mb-1.5">Category</label>
        <select x-model="ruleAddForm.category"
                class="w-full px-3 py-2.5 border-2 border-card-border bg-surface rounded-xl text-sm text-gray-100 focus:outline-none focus:border-primary transition appearance-none">
          <template x-for="opt in categoryOptions" :key="opt">
            <option :value="opt" x-text="getCategoryEmoji(opt) + ' ' + opt"></option>
          </template>
        </select>
      </div>
    </div>
    <div class="flex gap-3 mt-6">
      <button @click="closeRuleAdd()" class="flex-1 py-3 bg-gray-700 text-gray-300 rounded-xl text-sm font-semibold hover:bg-gray-600 transition">Cancel</button>
      <button @click="saveRuleAdd()" class="flex-1 py-3 bg-primary text-white rounded-xl text-sm font-semibold hover:bg-primary-dark transition">Add Rule</button>
    </div>
  </div>
</div>
```

### 5e. Add Rule Edit modal
Same structure as Rule Add modal above, but:
- `x-show="ruleEditOpen"`, `@click.self="closeRuleEdit()"`
- Title: "Edit Rule"
- Models: `ruleEditForm.keyword`, `ruleEditForm.category`
- Cancel button: `@click="closeRuleEdit()"`
- Save button: `@click="saveRuleEdit()"` with label "Save Changes"

### 5f. Add Rule Delete confirm modal
```html
<!-- ==================== RULE DELETE MODAL ==================== -->
<div x-show="ruleDeleteOpen" x-transition.opacity
     @click.self="closeRuleDelete()"
     class="fixed inset-0 bg-black/70 z-[1000] flex items-end md:items-center justify-center">
  <div x-show="ruleDeleteOpen"
       x-transition:enter="transition ease-out duration-300"
       x-transition:enter-start="translate-y-full md:translate-y-0 md:scale-95 md:opacity-0"
       x-transition:enter-end="translate-y-0 md:scale-100 md:opacity-100"
       x-init="initModalSwipe($el)"
       class="bg-card rounded-t-2xl md:rounded-2xl p-6 w-full md:max-w-[500px] md:w-[90%] shadow-2xl relative modal-sheet">
    <div class="w-10 h-1 bg-gray-600 rounded-full mx-auto -mt-2 mb-5 md:hidden cursor-grab active:cursor-grabbing modal-drag-handle"></div>
    <div class="text-xl font-semibold text-gray-100 mb-5 modal-header">Delete Rule</div>
    <div class="bg-red-900/30 border-l-4 border-danger p-4 rounded-xl mb-4 text-sm text-gray-400">
      Delete the rule for keyword:
      <div class="font-semibold text-gray-100 mt-1" x-text='"' + ruleDeleteKeyword + '"'></div>
    </div>
    <p class="text-xs text-gray-500 mb-4">This will not change any existing transactions.</p>
    <div class="flex gap-3">
      <button @click="closeRuleDelete()" class="flex-1 py-3 bg-gray-700 text-gray-300 rounded-xl text-sm font-semibold hover:bg-gray-600 transition">Cancel</button>
      <button @click="confirmRuleDelete()" class="flex-1 py-3 bg-danger text-white rounded-xl text-sm font-semibold hover:bg-red-600 transition">Delete</button>
    </div>
  </div>
</div>
```

### 5g. Add Apply-All confirm modal
```html
<!-- ==================== APPLY ALL RULES MODAL ==================== -->
<div x-show="applyAllOpen" x-transition.opacity
     @click.self="closeApplyAll()"
     class="fixed inset-0 bg-black/70 z-[1000] flex items-end md:items-center justify-center">
  <div x-show="applyAllOpen"
       x-transition:enter="transition ease-out duration-300"
       x-transition:enter-start="translate-y-full md:translate-y-0 md:scale-95 md:opacity-0"
       x-transition:enter-end="translate-y-0 md:scale-100 md:opacity-100"
       x-init="initModalSwipe($el)"
       class="bg-card rounded-t-2xl md:rounded-2xl p-6 w-full md:max-w-[500px] md:w-[90%] shadow-2xl relative modal-sheet">
    <div class="w-10 h-1 bg-gray-600 rounded-full mx-auto -mt-2 mb-5 md:hidden cursor-grab active:cursor-grabbing modal-drag-handle"></div>
    <div class="text-xl font-semibold text-gray-100 mb-4 modal-header">Re-apply All Rules</div>
    <p class="text-sm text-gray-400 mb-3">
      All merchant rules will be applied to your transaction history using priority order (first match wins).
    </p>
    <div class="bg-surface border border-card-border rounded-xl p-3 mb-5 text-xs text-gray-500">
      Manually-edited transactions are protected and will not be changed.
    </div>
    <div class="flex gap-3">
      <button @click="closeApplyAll()" class="flex-1 py-3 bg-gray-700 text-gray-300 rounded-xl text-sm font-semibold hover:bg-gray-600 transition">Cancel</button>
      <button @click="confirmApplyAll()" :disabled="applyAllLoading"
              class="flex-1 py-3 bg-primary text-white rounded-xl text-sm font-semibold hover:bg-primary-dark transition disabled:opacity-50"
              x-text="applyAllLoading ? 'Applying...' : 'Re-apply All'"></button>
    </div>
  </div>
</div>
```

---

## File Change Summary

| File | Changes |
|---|---|
| `static/js/app.js` | Update `categoryOptions`; update defaults; add all new state; add all new methods; update `openEdit`, `closeEdit`, `saveEdit`, `switchTab`, `initModalSwipe` |
| `static/js/api.js` | Add 7 new exported functions for rules endpoints |
| `static/js/utils.js` | Update `getCategoryEmoji` map |
| `static/js/tabs/rules.js` | New file (placeholder export) |
| `static/index.html` | Add Rules tab panel; add Rules tab nav button; replace edit modal inner content with 3-step version; add Rule Add modal; add Rule Edit modal; add Rule Delete modal; add Apply-All modal; add Rules sticky footer; add Rules FAB |
