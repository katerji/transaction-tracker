import {
  formatCurrency,
  formatDateTime,
  getCategoryEmoji,
  escapeHtml,
  hapticFeedback,
  dateToInputValue,
  inputValueToDate,
  nowLocalInput,
} from './utils.js';

import { fetchStats, createTransaction, updateTransaction, removeTransaction, parseTransaction, fetchRules, createRule, updateRule, deleteRule, applyRuleSingle, applyAllRules, moveRulePriority } from './api.js';
import { computeTodaySpend, computeBiggestExpense, computeDailyAverage, computeTopCategory } from './tabs/dashboard.js';
import { computeSearchedAndSorted, computeGroupedByDate } from './tabs/transactions.js';

// Expose to Alpine templates
window.formatCurrency = formatCurrency;
window.formatDateTime = formatDateTime;
window.getCategoryEmoji = getCategoryEmoji;
window.escapeHtml = escapeHtml;

export default function app() {
  return {
    // Tab state
    currentTab: 'dashboard',

    // Data state
    loading: true,
    error: null,
    stats: null,
    categories: [],
    allTransactions: [],
    lastUpdate: 'Never',

    // Categories tab state
    expandedCategories: {},

    // Transactions tab state
    searchQuery: '',
    sortBy: 'date',
    sortDirection: 'desc',
    expandedTxId: null,

    // Edit modal
    editOpen: false,
    editId: null,
    editOld: null,
    editForm: { description: '', amount: 0, date: '', category: 'Groceries' },

    // Delete modal
    deleteOpen: false,
    deleteId: null,
    deletePreview: '',

    // Add modal
    addOpen: false,
    addForm: { description: '', amount: '', date: '', category: 'Groceries' },

    // Paste SMS
    canAutoClipboard: false,
    pasteOpen: false,
    pasteText: '',
    pasteLoading: false,

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

    // Toast
    toastMessage: '',
    toastVisible: false,

    // Pull to refresh
    pullStartY: 0,
    pullCurrentY: 0,
    isPulling: false,
    isRefreshing: false,

    // Category options
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

    async init() {
      this.canAutoClipboard = /Android.*Chrome\//.test(navigator.userAgent)
        && !!navigator.clipboard?.readText;
      await this.loadStats();
      this.initPullToRefresh();
    },

    async loadStats() {
      this.loading = true;
      this.error = null;
      try {
        const data = await fetchStats();
        this.stats = data;
        this.categories = data.categories || [];
        this.allTransactions = data.allTransactions || [];
        this.lastUpdate = new Date().toLocaleTimeString();
      } catch (e) {
        this.error = e.message;
      } finally {
        this.loading = false;
      }
    },

    // Tab switching
    switchTab(tab) {
      if (tab === 'add') {
        this.openAdd();
        return;
      }
      if (tab === 'rules') {
        this.loadRules();
      }
      this.currentTab = tab;
      hapticFeedback('light');
    },

    // Dashboard getters (defined directly for Alpine reactivity)
    get todaySpend() {
      return computeTodaySpend(this.allTransactions);
    },
    get biggestExpense() {
      return computeBiggestExpense(this.allTransactions);
    },
    get dailyAverage() {
      return computeDailyAverage(this.stats?.total || 0, this.stats?.cycle || '');
    },
    get topCategory() {
      return computeTopCategory(this.categories);
    },
    get recentTransactions() {
      return [...this.allTransactions]
        .sort((a, b) => b.date.localeCompare(a.date))
        .slice(0, 5);
    },

    // Transactions getters
    get searchedSortedTransactions() {
      return computeSearchedAndSorted(this.allTransactions, this.searchQuery, this.sortBy, this.sortDirection);
    },
    get groupedTransactions() {
      return computeGroupedByDate(this.searchedSortedTransactions);
    },

    toggleSort(field) {
      if (this.sortBy === field) {
        this.sortDirection = this.sortDirection === 'desc' ? 'asc' : 'desc';
      } else {
        this.sortBy = field;
        this.sortDirection = 'desc';
      }
      hapticFeedback('light');
    },

    toggleExpandTx(id) {
      this.expandedTxId = this.expandedTxId === id ? null : id;
    },

    // Category expand/collapse
    toggleCategory(index) {
      this.expandedCategories[index] = !this.expandedCategories[index];
    },

    isCategoryExpanded(index) {
      return !!this.expandedCategories[index];
    },

    // Edit modal
    openEdit(tx) {
      this.editId = tx.id;
      this.editOld = { ...tx };
      this.editForm = {
        description: tx.description,
        amount: tx.amount,
        date: dateToInputValue(tx.date),
        category: tx.category,
      };
      this.editOpen = true;
      this.editStep = 'edit';
      this.editRuleKeyword = '';
      this.editRuleMatchCount = 0;
      this.editRuleProtectedCount = 0;
      this.editRuleSavedId = null;
    },

    closeEdit() {
      this.editOpen = false;
      this.editId = null;
      this.editOld = null;
      this.editStep = 'edit';
      this.editRuleKeyword = '';
      this.editRuleSavedId = null;
    },

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

    applyUpdate(txId, oldTx, newTx) {
      const oldCat = oldTx.category;
      const newCat = newTx.category;
      const catChanged = oldCat !== newCat;

      for (let cat of this.categories) {
        const txIdx = cat.transactions.findIndex((t) => t.id === txId);
        if (txIdx === -1) continue;

        if (catChanged) {
          const removed = cat.transactions.splice(txIdx, 1)[0];
          cat.total -= oldTx.amount;
          cat.count--;
          removed.description = newTx.description;
          removed.amount = newTx.amount;
          removed.date = newTx.date;
          removed.category = newTx.category;

          let dest = this.categories.find((c) => c.category === newCat);
          if (!dest) {
            dest = {
              category: newCat,
              emoji: getCategoryEmoji(newCat),
              total: 0,
              count: 0,
              transactions: [],
            };
            this.categories.push(dest);
          }
          dest.transactions.push(removed);
          dest.total += newTx.amount;
          dest.count++;

          if (cat.count === 0) {
            this.categories = this.categories.filter((c) => c !== cat);
          }
        } else {
          cat.transactions[txIdx].description = newTx.description;
          cat.transactions[txIdx].amount = newTx.amount;
          cat.transactions[txIdx].date = newTx.date;
          cat.total = cat.total - oldTx.amount + newTx.amount;
        }
        break;
      }

      if (oldTx.category !== 'Income/Transfer') {
        this.stats.total = this.stats.total - oldTx.amount + newTx.amount;
      }

      const allIdx = this.allTransactions.findIndex((t) => t.id === txId);
      if (allIdx !== -1) {
        this.allTransactions[allIdx] = { ...this.allTransactions[allIdx], ...newTx };
      }

      this.categories.sort((a, b) => b.total - a.total);
    },

    // Delete modal
    openDelete(tx) {
      this.deleteId = tx.id;
      this.deletePreview = tx.description + ' - ' + formatCurrency(tx.amount);
      this.deleteOpen = true;
      hapticFeedback('medium');
    },

    closeDelete() {
      this.deleteOpen = false;
      this.deleteId = null;
    },

    async confirmDelete() {
      if (!this.deleteId) return;
      try {
        await removeTransaction(this.deleteId);
        const txId = this.deleteId;
        this.closeDelete();
        this.applyDelete(txId);
        hapticFeedback('success');
        this.showToast('Transaction deleted successfully');
      } catch (e) {
        this.showToast('Error: ' + e.message);
      }
    },

    applyDelete(txId) {
      for (let i = 0; i < this.categories.length; i++) {
        const cat = this.categories[i];
        const txIdx = cat.transactions.findIndex((t) => t.id === txId);
        if (txIdx === -1) continue;

        const deleted = cat.transactions[txIdx];
        cat.transactions.splice(txIdx, 1);
        cat.total -= deleted.amount;
        cat.count--;

        if (deleted.category !== 'Income/Transfer') {
          this.stats.total -= deleted.amount;
        }
        this.stats.count--;

        if (cat.count === 0) {
          this.categories.splice(i, 1);
        }
        break;
      }

      this.allTransactions = this.allTransactions.filter((t) => t.id !== txId);
      this.categories.sort((a, b) => b.total - a.total);
    },

    // Rules tab
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

    // Rule add modal
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

    // Rule edit modal
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

    // Rule delete
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

    // Priority move
    async moveRule(id, direction) {
      try {
        await moveRulePriority(id, direction);
        await this.loadRules();
        hapticFeedback('light');
      } catch (e) {
        this.showToast('Error: ' + e.message);
      }
    },

    // Apply-all modal
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

    // Edit modal steps
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

    // Add modal
    openAdd() {
      this.addForm = {
        description: '',
        amount: '',
        date: nowLocalInput(),
        category: 'Groceries',
      };
      this.addOpen = true;
      hapticFeedback('light');
    },

    closeAdd() {
      this.addOpen = false;
    },

    async saveNew() {
      const desc = this.addForm.description.trim();
      const amount = parseFloat(this.addForm.amount);
      if (!desc) return this.showToast('Please enter a description');
      if (isNaN(amount) || amount <= 0) return this.showToast('Please enter a valid amount');
      if (!this.addForm.date) return this.showToast('Please enter a date');

      const payload = {
        description: desc,
        amount: amount,
        date: inputValueToDate(this.addForm.date),
        category: this.addForm.category,
      };

      try {
        const result = await createTransaction(payload);
        this.closeAdd();
        const added = {
          id: result.transaction.id,
          description: payload.description,
          amount: payload.amount,
          date: payload.date,
          category: payload.category,
          confidence: 100,
        };
        this.applyAdd(added);
        hapticFeedback('success');
        this.showToast('Transaction added successfully');
      } catch (e) {
        this.showToast('Error: ' + e.message);
      }
    },

    applyAdd(tx) {
      let cat = this.categories.find((c) => c.category === tx.category);
      if (!cat) {
        cat = {
          category: tx.category,
          emoji: getCategoryEmoji(tx.category),
          total: 0,
          count: 0,
          transactions: [],
        };
        this.categories.push(cat);
      }
      cat.transactions.push(tx);
      cat.total += tx.amount;
      cat.count++;

      if (tx.category !== 'Income/Transfer') {
        this.stats.total += tx.amount;
      }
      this.stats.count++;

      this.allTransactions.unshift(tx);
      this.categories.sort((a, b) => b.total - a.total);
    },

    // Paste SMS
    async handlePasteFab() {
      hapticFeedback('light');
      if (this.canAutoClipboard) {
        try {
          const text = await navigator.clipboard.readText();
          await this.submitPasteText(text);
        } catch (e) {
          this.showToast('Error: ' + e.message);
        }
      } else {
        this.pasteText = '';
        this.pasteOpen = true;
      }
    },

    async submitPasteText(text) {
      if (!text || !text.trim()) {
        this.showToast('Please paste some text first');
        return;
      }
      this.pasteLoading = true;
      try {
        const result = await parseTransaction(text.trim());
        this.pasteOpen = false;
        this.pasteText = '';
        await this.loadStats();
        const count = result.transactions?.length || 1;
        hapticFeedback('success');
        this.showToast('Added ' + count + ' transaction' + (count !== 1 ? 's' : ''));
      } catch (e) {
        this.showToast('Error: ' + e.message);
      } finally {
        this.pasteLoading = false;
      }
    },

    async submitPaste() {
      await this.submitPasteText(this.pasteText);
    },

    closePaste() {
      this.pasteOpen = false;
      this.pasteText = '';
      this.pasteLoading = false;
    },

    // Toast
    showToast(message) {
      this.toastMessage = message;
      this.toastVisible = true;
      setTimeout(() => {
        this.toastVisible = false;
      }, 3000);
    },

    // Pull to refresh
    initPullToRefresh() {
      const container = this.$refs.container;
      if (!container) return;

      container.addEventListener(
        'touchstart',
        (e) => {
          if (window.scrollY === 0 && !this.isRefreshing) {
            this.pullStartY = e.touches[0].clientY;
            this.isPulling = true;
          }
        },
        { passive: true }
      );

      container.addEventListener(
        'touchmove',
        (e) => {
          if (!this.isPulling || this.isRefreshing) return;
          this.pullCurrentY = e.touches[0].clientY;
        },
        { passive: true }
      );

      container.addEventListener('touchend', async () => {
        if (!this.isPulling || this.isRefreshing) return;
        const dist = this.pullCurrentY - this.pullStartY;
        if (dist > 80) {
          this.isRefreshing = true;
          if (navigator.vibrate) navigator.vibrate(50);
          await this.loadStats();
          setTimeout(() => {
            this.isRefreshing = false;
          }, 500);
        }
        this.isPulling = false;
        this.pullStartY = 0;
        this.pullCurrentY = 0;
      });
    },

    get pullDistance() {
      if (!this.isPulling) return 0;
      return Math.max(0, this.pullCurrentY - this.pullStartY);
    },

    // Modal swipe to close
    initModalSwipe(el) {
      let startY = 0;
      let currentY = 0;
      let dragging = false;

      el.addEventListener(
        'touchstart',
        (e) => {
          const t = e.target;
          if (
            t.classList.contains('modal-drag-handle') ||
            t.classList.contains('modal-header') ||
            t === el
          ) {
            startY = e.touches[0].clientY;
            dragging = true;
            el.style.transition = 'none';
          }
        },
        { passive: true }
      );

      el.addEventListener(
        'touchmove',
        (e) => {
          if (!dragging) return;
          currentY = e.touches[0].clientY;
          const dy = currentY - startY;
          if (dy > 0) el.style.transform = `translateY(${dy}px)`;
        },
        { passive: true }
      );

      el.addEventListener('touchend', () => {
        if (!dragging) return;
        const dy = currentY - startY;
        el.style.transition = 'transform 0.3s ease-out';
        if (dy > 100) {
          el.style.transform = 'translateY(100%)';
          setTimeout(() => {
            this.editOpen = false;
            this.deleteOpen = false;
            this.addOpen = false;
            this.pasteOpen = false;
            this.ruleAddOpen = false;
            this.ruleEditOpen = false;
            this.ruleDeleteOpen = false;
            this.applyAllOpen = false;
            el.style.transform = 'translateY(0)';
          }, 300);
        } else {
          el.style.transform = 'translateY(0)';
        }
        dragging = false;
        startY = 0;
        currentY = 0;
      });
    },
  };
}
