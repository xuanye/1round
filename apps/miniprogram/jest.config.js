/** @type {import('jest').Config} */
module.exports = {
  testEnvironment: 'jsdom',
  snapshotSerializers: ['miniprogram-simulate/jest-snapshot-plugin'],
  testMatch: ['<rootDir>/tests/**/*.test.ts'],
  setupFilesAfterEnv: ['<rootDir>/tests/setup.ts'],
  transform: {
    '^.+\\.ts$': ['ts-jest', { tsconfig: '<rootDir>/tsconfig.test.json' }],
  },
  // Smoke tests run real builds; component tests load compiled dist output.
  // Serial execution keeps dist stable: smoke tests rebuild it, and the old
  // script chain ran them sequentially too.
  maxWorkers: 1,
  testTimeout: 30000,
};
