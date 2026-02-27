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
