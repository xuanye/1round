import { createWxMock } from '../helpers/wx-mock';

// Required (not imported) on purpose: miniprogram-simulate's .d.ts chain pulls
// in its own nested miniprogram-api-typings copy (3.x), whose global Component
// declarations conflict with the root 4.x typings and break `this` typing in
// src components. The untyped require (see tests/globals.d.ts) severs that.
const simulate = require('miniprogram-simulate');
const path = require('node:path');

// Component tests load the compiled dist output: miniprogram-simulate reads
// index.js/index.json/index.wxml, which only exist after `pnpm run build`
// (pnpm test runs the build first).
//
// Load once per test file: under Jest the component JS is only executed the
// first time it is required, so a second simulate.load() would mint a fresh
// id that never gets registered with j-component (render returns undefined).
const projectRoot = path.resolve(__dirname, '../..');
const componentPath = path.resolve(projectRoot, 'dist/components/empty-state/index');
const componentId = simulate.load(componentPath);

describe('empty-state component', () => {
  beforeEach(() => {
    (globalThis as Record<string, unknown>).wx = createWxMock();
  });

  it('renders the default text', () => {
    const component = simulate.render(componentId);
    component.attach(document.createElement('parent-wrapper'));

    expect(component.data.text).toBe('暂无数据');
    expect(component.querySelector('.empty')).toBeDefined();
    expect(component.toJSON()).toMatchSnapshot();
  });

  it('renders custom text passed as a property', () => {
    const component = simulate.render(componentId, { text: '还没有战绩' });
    component.attach(document.createElement('parent-wrapper'));

    expect(component.data.text).toBe('还没有战绩');
    expect(component.toJSON()).toMatchSnapshot();
  });
});
