export function formatScore(score: number): string {
  return score > 0 ? `+${score}` : `${score}`;
}

export function toInteger(value: string): number | null {
  if (!/^-?\d+$/.test(value.trim())) return null;
  return Number(value);
}

export function formatFriendlyTime(dateInput: Date | string | number): string {
  const target = new Date(dateInput);
  if (isNaN(target.getTime())) return '';
  const now = new Date();
  
  // Clear time for day-based comparison
  const targetDate = new Date(target.getFullYear(), target.getMonth(), target.getDate());
  const nowDate = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const diffTime = nowDate.getTime() - targetDate.getTime();
  const diffDays = Math.floor(diffTime / (24 * 3600 * 1000));

  if (diffDays === 0) {
    return '今天';
  }
  if (diffDays === 1) {
    return '昨天';
  }
  if (diffDays === 2) {
    return '前天';
  }
  if (diffDays > 2 && diffDays < 30) {
    return `${diffDays} 天前`;
  }

  const monthsDiff = (now.getFullYear() - target.getFullYear()) * 12 + now.getMonth() - target.getMonth();
  if (monthsDiff <= 1) {
    return '上个月';
  }
  if (monthsDiff < 6) {
    return `${monthsDiff} 个月前`;
  }
  if (monthsDiff >= 6 && monthsDiff < 12) {
    return '半年前';
  }
  if (monthsDiff >= 12 && monthsDiff < 24) {
    return '1 年前';
  }
  return `${Math.floor(monthsDiff / 12)} 年前`;
}

export function formatTimeOnly(dateInput: Date | string | number): string {
  const d = new Date(dateInput);
  if (isNaN(d.getTime())) return '';
  const hours = String(d.getHours()).padStart(2, '0');
  const minutes = String(d.getMinutes()).padStart(2, '0');
  return `${hours}:${minutes}`;
}
