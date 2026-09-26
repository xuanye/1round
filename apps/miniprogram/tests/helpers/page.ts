// Port of the old scripts/page-test-utils.js harness onto Jest.
//
// Differences from the vm version:
// - TS is transformed by ts-jest instead of ts.transpileModule.
// - Module loading goes through Jest's registry (jest.isolateModules), so
//   per-file `jest.mock('../../src/...')` calls replace the old ad-hoc
//   `require(name)` override map.
// - Globals (wx, getApp, setTimeout, ...) are installed on globalThis for the
//   test file, matching how each old test shared one vm context.

export interface LoadResult {
  exports: Record<string, any>;
  page?: Record<string, any>;
  component?: Record<string, any>;
}

type PageLike = Record<string, any>;

/**
 * Loads a module under src/ in an isolated module registry, capturing the
 * Page()/Component() definition when the file registers one.
 *
 * @param srcRelative e.g. 'pages/game-create/index.ts' or 'utils/format.ts'
 * @param globals     values assigned to globalThis for this and later loads
 */
export function loadSource(srcRelative: string, globals: Record<string, unknown> = {}): LoadResult {
  let moduleExports: LoadResult['exports'] = {};
  let page: PageLike | undefined;
  let component: PageLike | undefined;

  const prevPage = (globalThis as any).Page;
  const prevComponent = (globalThis as any).Component;
  (globalThis as any).Page = (value: PageLike) => { page = value; };
  (globalThis as any).Component = (value: PageLike) => { component = value; };
  try {
    for (const [key, value] of Object.entries(globals)) {
      (globalThis as any)[key] = value;
    }
    jest.isolateModules(() => {
      const modulePath = `../../src/${srcRelative.replace(/\.ts$/, '')}`;
      moduleExports = require(modulePath) as Record<string, any>;
    });
  } finally {
    (globalThis as any).Page = prevPage;
    (globalThis as any).Component = prevComponent;
  }
  return { exports: moduleExports, page, component };
}

/**
 * Creates an independent copy of a captured Page definition with its own data
 * bag, mirroring the old `instance()` helper.
 */
export function instance<T extends PageLike>(definition: T): T {
  return {
    ...definition,
    data: structuredClone(definition.data),
    setData(this: PageLike, next: Record<string, unknown>) {
      Object.assign(this.data, next);
    },
  } as T;
}
