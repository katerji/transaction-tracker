// Formatting and utility functions

export function formatCurrency(amount) {
  return new Intl.NumberFormat('en-AE', {
    style: 'currency',
    currency: 'AED',
    minimumFractionDigits: 2,
  }).format(amount);
}

export function formatDateTime(dateTimeStr) {
  let date;
  if (dateTimeStr.includes(' ')) {
    const [datePart, timePart] = dateTimeStr.split(' ');
    const [year, month, day] = datePart.split('-');
    const [hours, minutes] = timePart.split(':');
    date = new Date(year, month - 1, day, hours, minutes);
  } else {
    const [year, month, day] = dateTimeStr.split('-');
    date = new Date(year, month - 1, day);
  }
  return date.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  });
}

export function getCategoryEmoji(category) {
  const emojis = {
    'Food & Dining': '🍔',
    Transport: '🚗',
    Shopping: '🛍️',
    'Bills & Utilities': '💳',
    Entertainment: '🎬',
    'Health & Fitness': '💪',
    Travel: '✈️',
    'Cash Withdrawal': '💵',
    'Income/Transfer': '💰',
    Unknown: '❓',
  };
  return emojis[category] || '📌';
}

export function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

export function hapticFeedback(type = 'light') {
  if (!navigator.vibrate) return;
  const patterns = {
    light: 10,
    medium: 25,
    heavy: 50,
    success: [10, 50, 10],
    error: [50, 100, 50],
  };
  navigator.vibrate(patterns[type] || 10);
}

export function dateToInputValue(dateStr) {
  if (dateStr.includes(' ')) {
    return dateStr.substring(0, 16).replace(' ', 'T');
  }
  return dateStr + 'T00:00';
}

export function inputValueToDate(val) {
  if (val.includes('T')) {
    return val.replace('T', ' ') + ':00';
  }
  return val;
}

export function nowLocalInput() {
  const now = new Date();
  const y = now.getFullYear();
  const m = String(now.getMonth() + 1).padStart(2, '0');
  const d = String(now.getDate()).padStart(2, '0');
  const h = String(now.getHours()).padStart(2, '0');
  const min = String(now.getMinutes()).padStart(2, '0');
  return `${y}-${m}-${d}T${h}:${min}`;
}

export function parseTxDate(dateStr) {
  if (dateStr.includes(' ')) {
    const [datePart] = dateStr.split(' ');
    const [year, month, day] = datePart.split('-');
    return new Date(year, month - 1, day);
  }
  const [year, month, day] = dateStr.split('-');
  return new Date(year, month - 1, day);
}

export function formatDateGroupLabel(dateStr) {
  const [year, month, day] = dateStr.split('-').map(Number);
  const date = new Date(year, month - 1, day);
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const yesterday = new Date(today);
  yesterday.setDate(yesterday.getDate() - 1);

  if (date.getTime() === today.getTime()) return 'Today';
  if (date.getTime() === yesterday.getTime()) return 'Yesterday';
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

export function getDaysElapsedInCycle(cycleStr) {
  if (!cycleStr) return 1;
  const months = { Jan: 0, Feb: 1, Mar: 2, Apr: 3, May: 4, Jun: 5, Jul: 6, Aug: 7, Sep: 8, Oct: 9, Nov: 10, Dec: 11 };
  const parts = cycleStr.split(' ');
  if (parts.length !== 2) return 1;
  const month = months[parts[0]];
  const year = parseInt(parts[1]);
  if (month === undefined || isNaN(year)) return 1;

  const cycleStart = new Date(year, month, 23);
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const diffDays = Math.floor((today - cycleStart) / (1000 * 60 * 60 * 24)) + 1;
  return Math.max(1, diffDays);
}
