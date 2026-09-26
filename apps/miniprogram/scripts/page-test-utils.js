const fs = require('fs');
const path = require('path');
const vm = require('vm');
const ts = require('typescript');

function loadSource(file, globals = {}) {
  const source = fs.readFileSync(path.join(__dirname, '..', 'src', file), 'utf8');
  const compiled = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2019 },
  }).outputText;
  const exports = {};
  let page;
  const mockRequire = globals.require || (() => ({}));
  vm.runInNewContext(compiled, {
    console,
    exports,
    Page(value) { page = value; },
    ...globals,
    require(name) {
      if (name === './page' || name === '../game-detail/page') {
        return loadSource('pages/game-detail/page.ts', globals).exports;
      }
      if (name.endsWith('/system-font')) return loadSource('utils/system-font.ts', globals).exports;
      return mockRequire(name);
    },
  });
  return { exports, page };
}

function instance(page) {
  return { ...page, data: structuredClone(page.data), setData(next) { Object.assign(this.data, next); } };
}

module.exports = { loadSource, instance };
