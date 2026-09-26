import { requireLogin } from '../../services/auth.service';
import { getSystemFontClass } from '../../utils/system-font';

Page({
  data: {
    fontClass: getSystemFontClass(),
    displayName: '', initial: '', avatarUrl: '', avatarFailed: false,
    loading: true, error: '', version: '',
  },
  onLoad() {
    const version = wx.getAccountInfoSync().miniProgram.version;
    this.setData({ version: version ? `v${version}` : '开发版' });
  },
  async onShow() {
    this.getTabBar?.()?.setData({ selected: 2 });
    this.setData({ loading: true, error: '' });
    try {
      const user = await requireLogin();
      const displayName = user.displayName || '老书记';
      this.setData({ displayName, initial: Array.from(displayName)[0], avatarUrl: user.avatarUrl || '', avatarFailed: false });
    } catch (err) {
      this.setData({ error: (err as Error).message || '资料加载失败，请重试' });
    } finally {
      this.setData({ loading: false });
    }
  },
  onAvatarError() { this.setData({ avatarFailed: true }); },
  openHistory() { wx.navigateTo({ url: '/pages/history/index' }); },
  showHelp() {
    wx.showModal({
      title: '使用说明',
      content: '1. 扫码加入或创建牌局，和朋友一起记分。\n2. 点击“记一笔”，选中牌友并输入正整数：牌友加分，自己扣除对应总分。常用分值在创建牌局时设置，由牌友共享。\n3. 记错可在计分明细中撤销；自己的分值为 0 时可退出。\n4. 创建者可结束牌局，其他牌友可申请结束。结算后可在历史牌局和战绩中查看。\n5. 牌局内点击自己的名字可修改展示名称。',
      showCancel: false,
    });
  },
  showAbout() {
    wx.showModal({
      title: '关于一局一分',
      content: `一局一分 · ${this.data.version}\n简单记分，尽兴相聚。\n为家庭聚会和朋友牌局记录积分。`,
      showCancel: false,
    });
  },
});
