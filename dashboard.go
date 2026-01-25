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
            max-width: 600px;
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

        @media (max-width: 600px) {
            .container {
                padding: 20px;
            }

            .total-amount {
                font-size: 36px;
            }

            .header h1 {
                font-size: 24px;
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

    <script>
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
                                '</div>';
                        });
                    }

                    html += '</div></div>'; // Close transactions-list and category-wrapper
                });

                html += '</div>';
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

        // Auto-refresh every 30 seconds
        setInterval(function() {
            loadStats();
        }, 30000);
    </script>
</body>
</html>
`
