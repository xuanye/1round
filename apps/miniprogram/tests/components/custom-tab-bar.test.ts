import { createWxMock } from '../helpers/wx-mock';

// Required (not imported) on purpose — see empty-state.test.ts for why the
// miniprogram-simulate type chain must stay out of the test program.
const simulate = require('miniprogram-simulate');
const path = require('node:path');

// See empty-state.test.ts: load once per file because Jest's module registry
// only executes the component JS on the first require.
const projectRoot = path.resolve(__dirname, '../..');
const componentPath = path.resolve(projectRoot, 'dist/custom-tab-bar/index');
const componentId = simulate.load(componentPath);

function renderTabBar(wx: Record<string, unknown>) {
  (globalThis as Record<string, unknown>).wx = wx;
  const component = simulate.render(componentId);
  component.attach(document.createElement('parent-wrapper'));
  return component;
}

describe('custom-tab-bar component', () => {
  it('initializes tabs and font class on attach', () => {
    const component = renderTabBar(createWxMock({
      getDeviceInfo: () => ({ platform: 'ios', system: 'iOS 18' }),
    }));

    expect(component.data.tabs).toHaveLength(3);
    expect(component.data.tabs.map((tab: { text: string }) => tab.text)).toEqual(['牌局', '战绩', '我的']);
    expect(component.data.selected).toBe(0);
    expect(component.data.fontClass).toBe('font-ios');
    expect(component.toJSON()).toMatchSnapshot();
  });

  it('routes a tab tap through wx.switchTab', async () => {
    const switchTab = jest.fn();
    const component = renderTabBar(createWxMock({ switchTab }));

    component.querySelectorAll('.tab-item')[1].dispatchEvent('tap');
    // j-component dispatches custom events on a microtask.
    await Promise.resolve();

    expect(switchTab).toHaveBeenCalledWith({ url: '/pages/ranking/index' });
  });

  it('does not navigate when tapping the already-selected tab', async () => {
    const switchTab = jest.fn();
    const component = renderTabBar(createWxMock({ switchTab }));

    component.querySelectorAll('.tab-item')[0].dispatchEvent('tap');
    await Promise.resolve();

    expect(switchTab).not.toHaveBeenCalled();
  });

  it('ignores invalid tab indexes', () => {
    const switchTab = jest.fn();
    const component = renderTabBar(createWxMock({ switchTab }));

    component.instance.selectTab({ currentTarget: { dataset: { index: 99 } } });

    expect(switchTab).not.toHaveBeenCalled();
  });
});
