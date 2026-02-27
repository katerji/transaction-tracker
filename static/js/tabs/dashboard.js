// Dashboard tab — pure computation functions

import { getDaysElapsedInCycle } from '../utils.js';

export function computeTodaySpend(allTransactions) {
  const now = new Date();
  const todayStr = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
  return allTransactions
    .filter(tx => tx.date.startsWith(todayStr) && tx.category !== 'Income/Transfer')
    .reduce((sum, tx) => sum + tx.amount, 0);
}

export function computeBiggestExpense(allTransactions) {
  const spending = allTransactions.filter(tx => tx.category !== 'Income/Transfer');
  if (spending.length === 0) return { amount: 0, description: '-' };
  return spending.reduce((max, tx) => tx.amount > max.amount ? tx : max, spending[0]);
}

export function computeDailyAverage(total, cycleStr) {
  const days = getDaysElapsedInCycle(cycleStr);
  return days > 0 ? total / days : 0;
}

export function computeTopCategory(categories) {
  if (!categories || categories.length === 0) return { name: '-', emoji: '', total: 0 };
  const filtered = categories.filter(c => c.category !== 'Income/Transfer');
  if (filtered.length === 0) return { name: '-', emoji: '', total: 0 };
  const top = filtered.reduce((max, c) => c.total > max.total ? c : max, filtered[0]);
  return { name: top.category, emoji: top.emoji, total: top.total };
}
