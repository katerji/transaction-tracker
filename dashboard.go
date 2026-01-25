package main

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Transaction Tracker Dashboard</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
            display: flex;
            justify-content: center;
            align-items: center;
        }

        .container {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            max-width: 1000px;
            width: 100%;
            padding: 40px;
        }

        .header {
            text-align: center;
            margin-bottom: 30px;
        }

        .header h1 {
            color: #333;
            font-size: 28px;
            margin-bottom: 10px;
        }

        .cycle-info {
            color: #666;
            font-size: 14px;
        }

        .total-section {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            border-radius: 15px;
            text-align: center;
            margin-bottom: 30px;
        }

        .total-label {
            font-size: 14px;
            opacity: 0.9;
            margin-bottom: 5px;
        }

        .total-amount {
            font-size: 48px;
            font-weight: bold;
            margin-bottom: 5px;
        }

        .total-count {
            font-size: 14px;
            opacity: 0.8;
        }

        .categories-section {
            margin-bottom: 30px;
        }

        .categories-grid {
            display: grid;
            grid-template-columns: 1fr;
            gap: 10px;
        }

        @media (min-width: 768px) {
            .categories-grid {
                grid-template-columns: repeat(2, 1fr);
            }
        }

        @media (min-width: 1200px) {
            .categories-grid {
                grid-template-columns: repeat(3, 1fr);
            }
        }

        .section-title {
            font-size: 16px;
            font-weight: 600;
            color: #333;
            margin-bottom: 15px;
            padding-bottom: 10px;
            border-bottom: 2px solid #f0f0f0;
        }

        .category-wrapper {
            margin-bottom: 10px;
        }

        .category-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 15px;
            background: #f8f9fa;
            border-radius: 10px;
            transition: transform 0.2s;
        }

        .category-item:hover {
            transform: translateX(5px);
            background: #f0f1f3;
        }

        .category-left {
            display: flex;
            align-items: center;
            gap: 10px;
        }

        .category-emoji {
            font-size: 24px;
        }

        .category-name {
            font-weight: 500;
            color: #333;
        }

        .category-count {
            font-size: 12px;
            color: #666;
            margin-top: 2px;
        }

        .category-amount {
            font-size: 18px;
            font-weight: bold;
            color: #667eea;
        }

        .category-item {
            cursor: pointer;
            user-select: none;
        }

        .expand-icon {
            font-size: 14px;
            margin-left: 10px;
            transition: transform 0.3s ease;
            color: #999;
        }

        .expand-icon.expanded {
            transform: rotate(90deg);
        }

        .transactions-list {
            max-height: 0;
            overflow: hidden;
            transition: max-height 0.3s ease;
            margin-top: 0;
            padding: 0 15px;
        }

        .transactions-list.expanded {
            max-height: 1000px;
            margin-top: 10px;
            padding: 0 15px;
        }

        .transaction-item {
            background: white;
            padding: 12px 15px;
            margin: 8px 0;
            border-radius: 8px;
            border-left: 3px solid #667eea;
            font-size: 14px;
        }

        .transaction-desc {
            font-weight: 500;
            color: #333;
            margin-bottom: 4px;
        }

        .transaction-details {
            display: flex;
            justify-content: space-between;
            color: #666;
            font-size: 12px;
        }

        .transaction-meta {
            display: flex;
            gap: 10px;
        }

        .transaction-amount {
            font-weight: 600;
            color: #667eea;
        }

        .transaction-actions {
            display: flex;
            gap: 8px;
            margin-top: 8px;
        }

        .btn-edit, .btn-delete {
            padding: 6px 12px;
            border: none;
            border-radius: 6px;
            font-size: 12px;
            cursor: pointer;
            transition: all 0.2s;
            font-weight: 500;
        }

        .btn-edit {
            background: #667eea;
            color: white;
        }

        .btn-edit:hover {
            background: #5568d3;
        }

        .btn-delete {
            background: #fee;
            color: #c33;
        }

        .btn-delete:hover {
            background: #fdd;
        }

        /* Modal Styles */
        .modal-overlay {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(0, 0, 0, 0.6);
            z-index: 1000;
            align-items: center;
            justify-content: center;
        }

        .modal-overlay.active {
            display: flex;
        }

        .modal {
            background: white;
            border-radius: 15px;
            padding: 30px;
            max-width: 500px;
            width: 90%;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
        }

        .modal-header {
            font-size: 20px;
            font-weight: 600;
            margin-bottom: 20px;
            color: #333;
        }

        .form-group {
            margin-bottom: 15px;
        }

        .form-group label {
            display: block;
            font-size: 14px;
            font-weight: 500;
            color: #666;
            margin-bottom: 6px;
        }

        .form-group input, .form-group select {
            width: 100%;
            padding: 10px;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-size: 14px;
            transition: border-color 0.2s;
        }

        .form-group input:focus, .form-group select:focus {
            outline: none;
            border-color: #667eea;
        }

        .modal-actions {
            display: flex;
            gap: 10px;
            justify-content: flex-end;
            margin-top: 20px;
        }

        .btn-cancel, .btn-save, .btn-confirm {
            padding: 10px 20px;
            border: none;
            border-radius: 8px;
            font-size: 14px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s;
        }

        .btn-cancel {
            background: #f0f0f0;
            color: #666;
        }

        .btn-cancel:hover {
            background: #e0e0e0;
        }

        .btn-save {
            background: #667eea;
            color: white;
        }

        .btn-save:hover {
            background: #5568d3;
        }

        .btn-confirm {
            background: #f44;
            color: white;
        }

        .btn-confirm:hover {
            background: #e33;
        }

        .delete-warning {
            background: #fff9e6;
            border-left: 4px solid #ffa500;
            padding: 15px;
            border-radius: 8px;
            margin-bottom: 20px;
            font-size: 14px;
            color: #666;
        }

        .delete-item {
            font-weight: 600;
            color: #333;
            margin-top: 8px;
        }

        /* Toast Notification */
        .toast {
            position: fixed;
            top: 20px;
            right: 20px;
            background: #4caf50;
            color: white;
            padding: 16px 24px;
            border-radius: 8px;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
            font-size: 14px;
            font-weight: 500;
            z-index: 2000;
            opacity: 0;
            transform: translateX(400px);
            transition: all 0.3s ease;
        }

        .toast.show {
            opacity: 1;
            transform: translateX(0);
        }

        .toast.hide {
            opacity: 0;
            transform: translateX(400px);
        }

        .last-transaction {
            background: #fff9e6;
            border-left: 4px solid #ffd700;
            padding: 20px;
            border-radius: 10px;
            margin-bottom: 20px;
        }

        .last-transaction-title {
            font-size: 14px;
            color: #666;
            margin-bottom: 8px;
        }

        .last-transaction-desc {
            font-size: 16px;
            font-weight: 600;
            color: #333;
            margin-bottom: 5px;
        }

        .last-transaction-details {
            font-size: 14px;
            color: #666;
        }

        .loading {
            text-align: center;
            padding: 40px;
            color: #666;
        }

        .error {
            background: #fee;
            border-left: 4px solid #f44;
            padding: 15px;
            border-radius: 8px;
            color: #c33;
            margin-bottom: 20px;
        }

        .empty-state {
            text-align: center;
            padding: 40px;
            color: #666;
        }

        .empty-state-emoji {
            font-size: 48px;
            margin-bottom: 10px;
        }

        .refresh-info {
            text-align: center;
            font-size: 12px;
            color: #999;
            margin-top: 20px;
        }

        .refresh-btn {
            background: none;
            border: none;
            color: #667eea;
            cursor: pointer;
            text-decoration: underline;
            font-size: 12px;
            padding: 0;
            margin-left: 5px;
        }

        .refresh-btn:hover {
            color: #764ba2;
        }

        @media (max-width: 768px) {
            .container {
                padding: 20px;
                margin: 10px;
            }

            .total-amount {
                font-size: 36px;
            }

            .header h1 {
                font-size: 24px;
            }

            .categories-grid {
                grid-template-columns: 1fr !important;
            }
        }

        @media (max-width: 480px) {
            .container {
                padding: 15px;
                border-radius: 15px;
            }

            .total-section {
                padding: 20px;
            }

            .category-item {
                padding: 12px;
            }

            .transaction-item {
                padding: 10px 12px;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>💰 Transaction Tracker</h1>
            <p class="cycle-info">Billing Cycle Dashboard</p>
        </div>

        <div id="content">
            <div class="loading">Loading your stats...</div>
        </div>

        <div class="refresh-info">
            Last updated: <span id="lastUpdate">Never</span>
            <button class="refresh-btn" onclick="loadStats()">Refresh</button>
        </div>
    </div>

    <!-- Edit Modal -->
    <div id="editModal" class="modal-overlay">
        <div class="modal">
            <div class="modal-header">Edit Transaction</div>
            <div class="form-group">
                <label for="editDescription">Description</label>
                <input type="text" id="editDescription" placeholder="Transaction description">
            </div>
            <div class="form-group">
                <label for="editAmount">Amount (AED)</label>
                <input type="number" id="editAmount" step="0.01" placeholder="0.00">
            </div>
            <div class="form-group">
                <label for="editDate">Date</label>
                <input type="date" id="editDate">
            </div>
            <div class="form-group">
                <label for="editCategory">Category</label>
                <select id="editCategory">
                    <option value="Food & Dining">🍔 Food & Dining</option>
                    <option value="Transport">🚗 Transport</option>
                    <option value="Shopping">🛍️ Shopping</option>
                    <option value="Bills & Utilities">💳 Bills & Utilities</option>
                    <option value="Entertainment">🎬 Entertainment</option>
                    <option value="Health & Fitness">💪 Health & Fitness</option>
                    <option value="Travel">✈️ Travel</option>
                    <option value="Cash Withdrawal">💵 Cash Withdrawal</option>
                    <option value="Income/Transfer">💰 Income/Transfer</option>
                    <option value="Unknown">❓ Unknown</option>
                </select>
            </div>
            <div class="modal-actions">
                <button class="btn-cancel" onclick="closeEditModal()">Cancel</button>
                <button class="btn-save" onclick="saveTransaction()">Save Changes</button>
            </div>
        </div>
    </div>

    <!-- Delete Confirmation Modal -->
    <div id="deleteModal" class="modal-overlay">
        <div class="modal">
            <div class="modal-header">Confirm Delete</div>
            <div class="delete-warning">
                Are you sure you want to delete this transaction?
                <div class="delete-item" id="deleteItemPreview"></div>
            </div>
            <p style="font-size: 13px; color: #999;">This action cannot be undone.</p>
            <div class="modal-actions">
                <button class="btn-cancel" onclick="closeDeleteModal()">Cancel</button>
                <button class="btn-confirm" onclick="confirmDelete()">Delete</button>
            </div>
        </div>
    </div>

    <script>
        // Global state to store current stats data
        let statsData = null;
        let categoriesData = [];

        // Automatically load stats on page load
        window.addEventListener('DOMContentLoaded', () => {
            loadStats();
        });

        async function loadStats() {
            const content = document.getElementById('content');

            // Show loading
            content.innerHTML = '<div class="loading">Loading your stats...</div>';

            try {
                // Use relative URL - will automatically use the same host
                const response = await fetch('/stats');

                if (!response.ok) {
                    throw new Error('HTTP ' + response.status + ': ' + response.statusText);
                }

                const data = await response.json();

                if (!data.success) {
                    throw new Error(data.message || 'Failed to load stats');
                }

                renderStats(data);
                document.getElementById('lastUpdate').textContent = new Date().toLocaleTimeString();

            } catch (error) {
                content.innerHTML =
                    '<div class="error">' +
                    '<strong>Error loading stats:</strong><br>' +
                    error.message +
                    '<br><br>' +
                    '<small>Make sure your API is running correctly.</small>' +
                    '</div>';
            }
        }

        function renderStats(data) {
            const content = document.getElementById('content');

            // Store data globally for DOM updates
            statsData = data;
            categoriesData = data.categories || [];

            // Handle empty state
            if (data.count === 0) {
                content.innerHTML =
                    '<div class="empty-state">' +
                    '<div class="empty-state-emoji">📊</div>' +
                    '<h3>No Transactions Yet</h3>' +
                    '<p style="margin-top: 10px; color: #999;">' +
                    'Start logging your expenses for ' + escapeHtml(data.cycle) +
                    '</p>' +
                    '</div>';
                return;
            }

            let html =
                '<div class="total-section">' +
                '<div class="total-label">Total Spent</div>' +
                '<div class="total-amount">' + formatCurrency(data.total) + '</div>' +
                '<div class="total-count">' + data.count + ' transaction' + (data.count !== 1 ? 's' : '') + ' in ' + escapeHtml(data.cycle) + '</div>' +
                '</div>';

            // Last transaction
            if (data.lastTransaction) {
                const txDate = new Date(data.lastTransaction.date);
                const today = new Date();
                const isToday = txDate.toDateString() === today.toDateString();
                const dateStr = isToday ? 'today' : formatDate(txDate);

                html +=
                    '<div class="last-transaction">' +
                    '<div class="last-transaction-title">🕐 Last Transaction</div>' +
                    '<div class="last-transaction-desc">' + escapeHtml(data.lastTransaction.description) + '</div>' +
                    '<div class="last-transaction-details">' +
                    formatCurrency(data.lastTransaction.amount) + ' • ' + dateStr +
                    '</div>' +
                    '</div>';
            }

            // Categories
            if (data.categories && data.categories.length > 0) {
                html += '<div class="categories-section"><div class="section-title">By Category</div>';
                html += '<div class="categories-grid">';

                data.categories.forEach(function(cat, index) {
                    html += '<div class="category-wrapper">';

                    // Category header
                    html +=
                        '<div class="category-item" onclick="toggleCategory(' + index + ')">' +
                        '<div class="category-left">' +
                        '<div class="category-emoji">' + cat.emoji + '</div>' +
                        '<div>' +
                        '<div class="category-name">' + escapeHtml(cat.category) + '</div>' +
                        '<div class="category-count">' + cat.count + ' transaction' + (cat.count !== 1 ? 's' : '') + '</div>' +
                        '</div>' +
                        '</div>' +
                        '<div style="display: flex; align-items: center;">' +
                        '<div class="category-amount">' + formatCurrency(cat.total) + '</div>' +
                        '<span class="expand-icon" id="icon-' + index + '">▶</span>' +
                        '</div>' +
                        '</div>';

                    // Transactions list
                    html += '<div class="transactions-list" id="transactions-' + index + '">';

                    if (cat.transactions && cat.transactions.length > 0) {
                        cat.transactions.forEach(function(tx) {
                            const txDate = new Date(tx.date);
                            const dateStr = formatDate(txDate);

                            html +=
                                '<div class="transaction-item">' +
                                '<div class="transaction-desc">' + escapeHtml(tx.description) + '</div>' +
                                '<div class="transaction-details">' +
                                '<div class="transaction-meta">' +
                                '<span>' + dateStr + '</span>' +
                                '<span>' + tx.confidence + '% confidence</span>' +
                                '</div>' +
                                '<div class="transaction-amount">' + formatCurrency(tx.amount) + '</div>' +
                                '</div>' +
                                '<div class="transaction-actions">' +
                                '<button class="btn-edit" onclick=\'openEditModal(' + JSON.stringify(tx) + ')\'>✏️ Edit</button>' +
                                '<button class="btn-delete" onclick=\'openDeleteModal(' + tx.id + ', "' + escapeHtml(tx.description) + '", ' + tx.amount + ')\'>🗑️ Delete</button>' +
                                '</div>' +
                                '</div>';
                        });
                    }

                    html += '</div></div>'; // Close transactions-list and category-wrapper
                });

                html += '</div>'; // Close categories-grid
                html += '</div>'; // Close categories-section
            }

            content.innerHTML = html;
        }

        function toggleCategory(index) {
            const list = document.getElementById('transactions-' + index);
            const icon = document.getElementById('icon-' + index);

            if (list.classList.contains('expanded')) {
                list.classList.remove('expanded');
                icon.classList.remove('expanded');
                icon.textContent = '▶';
            } else {
                list.classList.add('expanded');
                icon.classList.add('expanded');
                icon.textContent = '▼';
            }
        }

        function formatCurrency(amount) {
            return new Intl.NumberFormat('en-AE', {
                style: 'currency',
                currency: 'AED',
                minimumFractionDigits: 2
            }).format(amount);
        }

        function formatDate(date) {
            return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        // Edit/Delete Modal Functions
        let currentTransactionId = null;
        let currentTransaction = null;

        function openEditModal(tx) {
            currentTransactionId = tx.id;
            currentTransaction = tx; // Store original transaction
            document.getElementById('editDescription').value = tx.description;
            document.getElementById('editAmount').value = tx.amount;
            document.getElementById('editDate').value = tx.date;
            document.getElementById('editCategory').value = tx.category;
            document.getElementById('editModal').classList.add('active');
        }

        function closeEditModal() {
            document.getElementById('editModal').classList.remove('active');
            currentTransactionId = null;
        }

        async function saveTransaction() {
            if (!currentTransactionId) return;

            const updatedTransaction = {
                description: document.getElementById('editDescription').value,
                amount: parseFloat(document.getElementById('editAmount').value),
                date: document.getElementById('editDate').value,
                category: document.getElementById('editCategory').value
            };

            try {
                const response = await fetch('/transaction/' + currentTransactionId, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(updatedTransaction)
                });

                if (!response.ok) {
                    throw new Error('Failed to update transaction');
                }

                closeEditModal();
                updateTransactionInDOM(currentTransactionId, currentTransaction, updatedTransaction);
            } catch (error) {
                alert('Error updating transaction: ' + error.message);
            }
        }

        function updateTransactionInDOM(txId, oldTx, newTx) {
            const oldCategory = oldTx.category;
            const newCategory = newTx.category;
            const categoryChanged = oldCategory !== newCategory;

            // Find the transaction in categoriesData
            for (let cat of categoriesData) {
                const txIndex = cat.transactions.findIndex(t => t.id === txId);
                if (txIndex !== -1) {
                    if (categoryChanged) {
                        // Remove from old category
                        const removedTx = cat.transactions.splice(txIndex, 1)[0];
                        cat.total -= oldTx.amount;
                        cat.count--;

                        // Update the transaction data
                        removedTx.description = newTx.description;
                        removedTx.amount = newTx.amount;
                        removedTx.date = newTx.date;
                        removedTx.category = newTx.category;

                        // Add to new category
                        let newCat = categoriesData.find(c => c.category === newCategory);
                        if (!newCat) {
                            // Create new category if it doesn't exist
                            newCat = {
                                category: newCategory,
                                emoji: getCategoryEmojiByName(newCategory),
                                total: 0,
                                count: 0,
                                transactions: []
                            };
                            categoriesData.push(newCat);
                        }
                        newCat.transactions.push(removedTx);
                        newCat.total += newTx.amount;
                        newCat.count++;

                        // Remove old category if empty
                        if (cat.count === 0) {
                            const catIndex = categoriesData.indexOf(cat);
                            if (catIndex > -1) {
                                categoriesData.splice(catIndex, 1);
                            }
                        }
                    } else {
                        // Just update in the same category
                        cat.transactions[txIndex].description = newTx.description;
                        cat.transactions[txIndex].amount = newTx.amount;
                        cat.transactions[txIndex].date = newTx.date;
                        cat.total = cat.total - oldTx.amount + newTx.amount;
                    }
                    break;
                }
            }

            // Update totals
            statsData.total = statsData.total - oldTx.amount + newTx.amount;

            // Update the DOM
            updateTotalsInDOM();
            updateCategoriesInDOM();

            // Show success toast
            showToast('Transaction updated successfully');
        }

        function openDeleteModal(id, description, amount) {
            currentTransactionId = id;
            const preview = description + ' - ' + formatCurrency(amount);
            document.getElementById('deleteItemPreview').textContent = preview;
            document.getElementById('deleteModal').classList.add('active');
        }

        function closeDeleteModal() {
            document.getElementById('deleteModal').classList.remove('active');
            currentTransactionId = null;
        }

        async function confirmDelete() {
            if (!currentTransactionId) return;

            try {
                const response = await fetch('/transaction/' + currentTransactionId, {
                    method: 'DELETE'
                });

                if (!response.ok) {
                    throw new Error('Failed to delete transaction');
                }

                closeDeleteModal();
                deleteTransactionFromDOM(currentTransactionId);
            } catch (error) {
                alert('Error deleting transaction: ' + error.message);
            }
        }

        function deleteTransactionFromDOM(txId) {
            let deletedTx = null;
            let categoryIndex = -1;

            // Find and remove transaction from categoriesData
            for (let i = 0; i < categoriesData.length; i++) {
                const txIndex = categoriesData[i].transactions.findIndex(t => t.id === txId);
                if (txIndex !== -1) {
                    deletedTx = categoriesData[i].transactions[txIndex];
                    categoriesData[i].transactions.splice(txIndex, 1);
                    categoriesData[i].total -= deletedTx.amount;
                    categoriesData[i].count--;
                    categoryIndex = i;
                    break;
                }
            }

            if (!deletedTx) return;

            // Update totals
            statsData.total -= deletedTx.amount;
            statsData.count--;

            // Remove empty categories
            if (categoriesData[categoryIndex].count === 0) {
                categoriesData.splice(categoryIndex, 1);
            }

            // Update the DOM
            updateTotalsInDOM();
            updateCategoriesInDOM();

            // Show success toast
            showToast('Transaction deleted successfully');
        }

        function updateTotalsInDOM() {
            const totalAmountEl = document.querySelector('.total-amount');
            const totalCountEl = document.querySelector('.total-count');

            if (totalAmountEl) {
                totalAmountEl.textContent = formatCurrency(statsData.total);
            }
            if (totalCountEl) {
                totalCountEl.textContent = statsData.count + ' transaction' + (statsData.count !== 1 ? 's' : '') + ' in ' + statsData.cycle;
            }
        }

        function updateCategoriesInDOM() {
            // Sort categories by total descending
            categoriesData.sort((a, b) => b.total - a.total);

            // Get the categories grid container
            const gridContainer = document.querySelector('.categories-grid');
            if (!gridContainer) return;

            // Store currently expanded categories
            const expandedCategories = new Set();
            document.querySelectorAll('.transactions-list.expanded').forEach(list => {
                const index = list.id.replace('transactions-', '');
                const categoryName = categoriesData[parseInt(index)]?.category;
                if (categoryName) expandedCategories.add(categoryName);
            });

            // Rebuild categories
            let html = '';
            categoriesData.forEach(function(cat, index) {
                const isExpanded = expandedCategories.has(cat.category);

                html += '<div class="category-wrapper">';
                html +=
                    '<div class="category-item" onclick="toggleCategory(' + index + ')">' +
                    '<div class="category-left">' +
                    '<div class="category-emoji">' + cat.emoji + '</div>' +
                    '<div>' +
                    '<div class="category-name">' + escapeHtml(cat.category) + '</div>' +
                    '<div class="category-count">' + cat.count + ' transaction' + (cat.count !== 1 ? 's' : '') + '</div>' +
                    '</div>' +
                    '</div>' +
                    '<div style="display: flex; align-items: center;">' +
                    '<div class="category-amount">' + formatCurrency(cat.total) + '</div>' +
                    '<span class="expand-icon' + (isExpanded ? ' expanded' : '') + '" id="icon-' + index + '">' + (isExpanded ? '▼' : '▶') + '</span>' +
                    '</div>' +
                    '</div>';

                html += '<div class="transactions-list' + (isExpanded ? ' expanded' : '') + '" id="transactions-' + index + '">';

                if (cat.transactions && cat.transactions.length > 0) {
                    cat.transactions.forEach(function(tx) {
                        const txDate = new Date(tx.date);
                        const dateStr = formatDate(txDate);

                        html +=
                            '<div class="transaction-item">' +
                            '<div class="transaction-desc">' + escapeHtml(tx.description) + '</div>' +
                            '<div class="transaction-details">' +
                            '<div class="transaction-meta">' +
                            '<span>' + dateStr + '</span>' +
                            '<span>' + tx.confidence + '% confidence</span>' +
                            '</div>' +
                            '<div class="transaction-amount">' + formatCurrency(tx.amount) + '</div>' +
                            '</div>' +
                            '<div class="transaction-actions">' +
                            '<button class="btn-edit" onclick=\'openEditModal(' + JSON.stringify(tx) + ')\'>✏️ Edit</button>' +
                            '<button class="btn-delete" onclick=\'openDeleteModal(' + tx.id + ', "' + escapeHtml(tx.description) + '", ' + tx.amount + ')\'>🗑️ Delete</button>' +
                            '</div>' +
                            '</div>';
                    });
                }

                html += '</div></div>';
            });

            gridContainer.innerHTML = html;
        }

        function getCategoryEmojiByName(category) {
            const emojis = {
                "Food & Dining": "🍔",
                "Transport": "🚗",
                "Shopping": "🛍️",
                "Bills & Utilities": "💳",
                "Entertainment": "🎬",
                "Health & Fitness": "💪",
                "Travel": "✈️",
                "Cash Withdrawal": "💵",
                "Income/Transfer": "💰",
                "Unknown": "❓"
            };
            return emojis[category] || "📌";
        }

        // Toast notification function
        function showToast(message) {
            // Remove existing toast if any
            const existingToast = document.querySelector('.toast');
            if (existingToast) {
                existingToast.remove();
            }

            // Create toast element
            const toast = document.createElement('div');
            toast.className = 'toast';
            toast.textContent = message;
            document.body.appendChild(toast);

            // Trigger animation
            setTimeout(() => {
                toast.classList.add('show');
            }, 10);

            // Auto-hide after 3 seconds
            setTimeout(() => {
                toast.classList.add('hide');
                toast.classList.remove('show');

                // Remove from DOM after animation
                setTimeout(() => {
                    toast.remove();
                }, 300);
            }, 3000);
        }

        // Close modals on overlay click
        document.addEventListener('click', function(e) {
            if (e.target.classList.contains('modal-overlay')) {
                closeEditModal();
                closeDeleteModal();
            }
        });

        // Auto-refresh every 30 seconds
        setInterval(function() {
            loadStats();
        }, 30000);
    </script>
</body>
</html>
`
