import type { Performance } from '../models/game-session';
import { formatScore } from './format';

export const periods = [
  { label: '最近半年', months: 6 },
  { label: '最近三个月', months: 3 },
  { label: '最近一年', months: 12 },
];

export function performanceRange(months: number, now = new Date()) {
  const start = new Date(now.getFullYear(), now.getMonth() - months + 1, 1);
  const end = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1);
  const date = (d: Date) => `${d.getFullYear()}.${String(d.getMonth() + 1).padStart(2, '0')}.${String(d.getDate()).padStart(2, '0')}`;
  return { start: start.toISOString(), end: end.toISOString(), label: `${date(start)} — ${date(now)}` };
}

export function scoreTone(score: number) {
  return score > 0 ? 'positive' : score < 0 ? 'negative' : 'zero';
}

// Use timestamps for the horizontal axis; sparse games retain their true spacing.
export function chartGeometry(trend: Performance['trend'], start: string, end: string, width: number, height: number) {
  const values = [0, ...trend.map(point => point.score)];
  const extent = Math.max(100, Math.max(...values) - Math.min(...values));
  const step = Math.pow(10, Math.floor(Math.log10(extent)));
  const top = Math.max(step, Math.ceil(Math.max(...values) / step) * step);
  const bottom = Math.min(-step, Math.floor(Math.min(...values) / step) * step);
  const labelWidth = Math.max(formatScore(top).length, formatScore(bottom).length) * 7 + 8;
  const left = Math.max(48, Math.min(width * .32, labelWidth)), right = width - 14, yTop = 24, yBottom = height - 30;
  const timeStart = new Date(start).getTime(), timeEnd = new Date(end).getTime();
  const x = (date: string) => left + (new Date(date).getTime() - timeStart) / (timeEnd - timeStart) * (right - left);
  const y = (score: number) => yBottom - (score - bottom) / (top - bottom) * (yBottom - yTop);
  const points = [{ x: left, y: y(0) }, ...trend.map(point => ({ x: x(point.settledAt), y: y(point.score) }))];
  const months = [] as { label: string; x: number }[];
  const date = new Date(start);
  const last = new Date(end);
  while (date < last) {
    months.push({ label: `${date.getMonth() + 1}月`, x: x(date.toISOString()) });
    date.setMonth(date.getMonth() + 1);
  }
  const labels = months.filter((_, i) => months.length <= 6 || i % 2 === 0);
  return { top, bottom, left, right, yTop, yBottom, points, labels, y, scoreLabel: formatScore(trend.length ? trend[trend.length - 1].score : 0) };
}
