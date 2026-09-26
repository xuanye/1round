const assert = require('node:assert/strict');
const { loadSource } = require('./page-test-utils');

const { buildSuggestedNames, FALLBACK_SUGGESTION_NAMES } = loadSource('utils/suggestion-names.ts').exports;

// Mid-Autumn 2026-09-25 (3 days) and National Day 2026-10-01 are both
// within their windows on 9/26: holiday names first, "小聚" leads once
// the festival day has arrived.
const duringHoliday = buildSuggestedNames(new Date(2026, 8, 26));
assert.equal(duringHoliday.length, 3);
assert.equal(duringHoliday[0], '中秋节小聚');
assert.ok(duringHoliday.includes('中秋节前小聚'));
assert.ok(duringHoliday.includes('国庆节前小聚'));

// Window starts 7 days before the festival day.
const windowStart = buildSuggestedNames(new Date(2026, 8, 18));
assert.equal(windowStart[0], '中秋节前小聚');
assert.equal(windowStart.length, 3);

const beforeWindow = buildSuggestedNames(new Date(2026, 8, 17));
assert.ok(!beforeWindow.some((name) => name.includes('中秋') || name.includes('国庆')));

// Non-holiday dates fall back to 3 unique names from the regular pool.
for (const date of [new Date(2026, 2, 10), new Date(2026, 7, 5), new Date(2027, 4, 20)]) {
  const names = buildSuggestedNames(date);
  assert.equal(names.length, 3);
  assert.equal(new Set(names).size, 3);
  for (const name of names) {
    assert.ok(FALLBACK_SUGGESTION_NAMES.includes(name), `unexpected name ${name}`);
  }
}

// Fallback pool itself has no duplicates.
assert.equal(new Set(FALLBACK_SUGGESTION_NAMES).size, FALLBACK_SUGGESTION_NAMES.length);
