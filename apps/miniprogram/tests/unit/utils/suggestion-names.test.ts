import {
  buildSuggestedNames,
  FALLBACK_SUGGESTION_NAMES,
} from '../../../src/utils/suggestion-names';

describe('buildSuggestedNames', () => {
  // Mid-Autumn 2026-09-25 (3 days) and National Day 2026-10-01 are both
  // within their windows on 9/26: holiday names first, "小聚" leads once
  // the festival day has arrived.
  it('ranks holiday names during festival windows', () => {
    const duringHoliday = buildSuggestedNames(new Date(2026, 8, 26));
    expect(duringHoliday).toHaveLength(3);
    expect(duringHoliday[0]).toBe('中秋节小聚');
    expect(duringHoliday).toContain('中秋节前小聚');
    expect(duringHoliday).toContain('国庆节前小聚');
  });

  // Window starts 7 days before the festival day.
  it('starts the window 7 days before the festival day', () => {
    const windowStart = buildSuggestedNames(new Date(2026, 8, 18));
    expect(windowStart[0]).toBe('中秋节前小聚');
    expect(windowStart).toHaveLength(3);
  });

  it('excludes holiday names before the window opens', () => {
    const beforeWindow = buildSuggestedNames(new Date(2026, 8, 17));
    expect(beforeWindow.some((name) => name.includes('中秋') || name.includes('国庆'))).toBe(false);
  });

  // Non-holiday dates fall back to 3 unique names from the regular pool.
  it('falls back to 3 unique names from the regular pool', () => {
    for (const date of [new Date(2026, 2, 10), new Date(2026, 7, 5), new Date(2027, 4, 20)]) {
      const names = buildSuggestedNames(date);
      expect(names).toHaveLength(3);
      expect(new Set(names).size).toBe(3);
      for (const name of names) {
        expect(FALLBACK_SUGGESTION_NAMES).toContain(name);
      }
    }
  });

  // Fallback pool itself has no duplicates.
  it('has no duplicates in the fallback pool', () => {
    expect(new Set(FALLBACK_SUGGESTION_NAMES).size).toBe(FALLBACK_SUGGESTION_NAMES.length);
  });
});
