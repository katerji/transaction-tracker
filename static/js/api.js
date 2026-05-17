// Backend API communication

export async function fetchStats() {
  const res = await fetch('/stats');
  if (!res.ok) throw new Error('HTTP ' + res.status + ': ' + res.statusText);
  const data = await res.json();
  if (!data.success) throw new Error(data.message || 'Failed to load stats');
  return data;
}

export async function createTransaction(data) {
  const res = await fetch('/transaction/manual', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!res.ok) throw new Error('Failed to add transaction');
  const result = await res.json();
  if (!result.success) throw new Error(result.message || 'Failed to add transaction');
  return result;
}

export async function updateTransaction(id, data) {
  const res = await fetch('/transaction/' + id, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!res.ok) throw new Error('Failed to update transaction');
  return res.json();
}

export async function removeTransaction(id) {
  const res = await fetch('/transaction/' + id, { method: 'DELETE' });
  if (!res.ok) throw new Error('Failed to delete transaction');
  return res.json();
}

export async function parseTransaction(text) {
  const res = await fetch('/transaction', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text }),
  });
  if (!res.ok) throw new Error('Failed to parse transaction');
  const result = await res.json();
  if (!result.success) throw new Error(result.message || 'No transactions found');
  return result;
}

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
