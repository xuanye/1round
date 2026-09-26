import { getSystemFontClass } from '../utils/system-font';

Component({
  options: { styleIsolation: 'apply-shared' },
  data: {
    fontClass: 'font-system',
    selected: 0,
    tabs: [
      { text: '牌局', path: '/pages/home/index', icon: '/images/tabbar/game.png', activeIcon: '/images/tabbar/game-active.png' },
      { text: '战绩', path: '/pages/ranking/index', icon: '/images/tabbar/records.png', activeIcon: '/images/tabbar/records-active.png' },
      { text: '我的', path: '/pages/mine/index', icon: '/images/tabbar/mine.png', activeIcon: '/images/tabbar/mine-active.png' },
    ],
  },
  lifetimes: {
    attached() { this.setData({ fontClass: getSystemFontClass() }); },
  },
  methods: {
    selectTab(event: WechatMiniprogram.TouchEvent) {
      const index = Number(event.currentTarget.dataset.index);
      const tab = this.data.tabs[index];
      if (!tab || index === this.data.selected) return;
      wx.switchTab({ url: tab.path });
    },
  },
});
