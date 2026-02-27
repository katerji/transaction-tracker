// Transactions tab — search, sort, date-grouping logic

import { formatDateGroupLabel } from '../utils.js';

export function computeSearchedAndSorted(allTx, query, sortBy, dir) {
  let result = [...allTx];

  if (query.trim()) {
    const q = query.trim().toLowerCase();
    result = result.filter(tx =>
      tx.description.toLowerCase().includes(q) ||
      tx.category.toLowerCase().includes(q) ||
      String(tx.amount).includes(q)
    );
  }

  result.sort((a, b) => {
    let cmp = 0;
    if (sortBy === 'date') {
      cmp = a.date.localeCompare(b.date);
    } else if (sortBy === 'amount') {
      cmp = a.amount - b.amount;
    } else if (sortBy === 'category') {
      cmp = a.category.localeCompare(b.category);
    }
    return dir === 'desc' ? -cmp : cmp;
  });

  return result;
}

export function computeGroupedByDate(sortedTx) {
  const groups = {};
  for (const tx of sortedTx) {
    const dateKey = tx.date.split(' ')[0];
    if (!groups[dateKey]) groups[dateKey] = [];
    groups[dateKey].push(tx);
  }
  return Object.keys(groups).map(date => ({
    date,
    label: formatDateGroupLabel(date),
    transactions: groups[date],
  }));
}
