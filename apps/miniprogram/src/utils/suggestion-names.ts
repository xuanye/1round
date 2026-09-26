export const FALLBACK_SUGGESTION_NAMES = ['周末牌局', '朋友小聚', '家庭聚会', '牌友小聚', '老友局', '欢乐时光'];

export interface FestivalDate {
  start: string;
  days: number;
}

export interface Festival {
  name: string;
  dates: FestivalDate[];
}

// Gregorian dates per year; lunar holidays are pinned per year.
// Extend `dates` each December with the new State Council schedule.
const FESTIVALS: Festival[] = [
  { name: '元旦', dates: [{ start: '2026-01-01', days: 1 }, { start: '2027-01-01', days: 1 }] },
  { name: '春节', dates: [{ start: '2026-02-17', days: 8 }, { start: '2027-02-06', days: 8 }] },
  { name: '五一', dates: [{ start: '2026-05-01', days: 5 }, { start: '2027-05-01', days: 5 }] },
  { name: '端午', dates: [{ start: '2026-06-19', days: 3 }, { start: '2027-06-09', days: 3 }] },
  { name: '中秋节', dates: [{ start: '2026-09-25', days: 3 }, { start: '2027-09-15', days: 3 }] },
  { name: '国庆节', dates: [{ start: '2026-10-01', days: 7 }, { start: '2027-10-01', days: 7 }] },
];

const DAYS_BEFORE_FESTIVAL = 7;
const SUGGESTION_COUNT = 3;

function dayNumber(year: number, month: number, day: number): number {
  return Math.floor(Date.UTC(year, month - 1, day) / 86400000);
}

function parseDayNumber(date: string): number {
  const [year, month, day] = date.split('-').map(Number);
  return dayNumber(year, month, day);
}

function todayDayNumber(today: Date): number {
  return dayNumber(today.getFullYear(), today.getMonth() + 1, today.getDate());
}

function pickFallback(count: number): string[] {
  const shuffled = [...FALLBACK_SUGGESTION_NAMES];
  for (let i = shuffled.length - 1; i > 0; i -= 1) {
    const j = Math.floor(Math.random() * (i + 1));
    [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
  }
  return shuffled.slice(0, count);
}

// Returns 3 suggested game names: holiday-themed names while a festival
// is within its pre-holiday/holiday window, otherwise random regular names.
export function buildSuggestedNames(today: Date): string[] {
  const todayNumber = todayDayNumber(today);
  const suggestions: string[] = [];
  for (const festival of FESTIVALS) {
    for (const date of festival.dates) {
      const start = parseDayNumber(date.start);
      const end = start + date.days - 1;
      if (todayNumber < start - DAYS_BEFORE_FESTIVAL || todayNumber > end) continue;
      const names = todayNumber < start
        ? [`${festival.name}前小聚`, `${festival.name}小聚`]
        : [`${festival.name}小聚`, `${festival.name}前小聚`];
      for (const name of names) {
        if (!suggestions.includes(name)) suggestions.push(name);
      }
    }
  }
  if (suggestions.length >= SUGGESTION_COUNT) return suggestions.slice(0, SUGGESTION_COUNT);
  return [...suggestions, ...pickFallback(SUGGESTION_COUNT - suggestions.length)];
}
