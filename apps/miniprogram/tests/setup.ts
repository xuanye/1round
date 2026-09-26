// Shared test environment defaults. Individual tests override `wx`/`getApp`
// via the loadSource helper, matching the old vm-harness behavior where each
// test supplied its own globals.

// The jsdom test environment does not expose Node's structuredClone, which
// the page-instance helper needs for copying page data bags. v8.serialize
// gives the same plain-data clone semantics (throws on functions, like
// structuredClone does).
if (typeof (globalThis as { structuredClone?: unknown }).structuredClone !== 'function') {
  const v8 = require('node:v8');
  (globalThis as Record<string, unknown>).structuredClone = (value: unknown) =>
    v8.deserialize(v8.serialize(value));
}

(globalThis as Record<string, unknown>).wx = {};
(globalThis as Record<string, unknown>).getApp = () => ({
  globalData: { baseUrl: 'https://example.test' },
});
