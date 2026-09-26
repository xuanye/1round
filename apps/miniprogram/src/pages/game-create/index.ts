import { requireLogin } from '../../services/auth.service';
import { createGame } from '../../services/game.service';
import { saveRecentSession } from '../../utils/storage';
import { getSystemFontClass } from '../../utils/system-font';
import { DEFAULT_PRESET_SCORES, MAX_PRESET_SCORES, buildPresetOptions, selectedPresetScores } from '../../utils/preset-scores';
import { buildSuggestedNames } from '../../utils/suggestion-names';

Page({
  data: {
    fontClass: getSystemFontClass(),
    name: '周六朋友局',
    nameLength: 5,
    suggestedNames: buildSuggestedNames(new Date()),
    nameFocused: false,
    submitting: false,
    creationError: '',
    presetOptions: buildPresetOptions(DEFAULT_PRESET_SCORES),
  },
  async onLoad() {
    try {
      await requireLogin();
    } catch (err) {
      wx.showToast({ title: (err as any).message || '登录失败', icon: 'none' });
      wx.switchTab({ url: '/pages/home/index' });
    }
  },
  onNameInput(event: WechatMiniprogram.Input) {
    const name = event.detail.value;
    this.setData({ name, nameLength: name.length, creationError: '' });
  },
  onNameFocus() {
    this.setData({ nameFocused: true });
  },
  onNameBlur() {
    this.setData({ nameFocused: false });
  },
  clearName() {
    if (this.data.submitting) return;
    this.setData({ name: '', nameLength: 0, creationError: '', nameFocused: true });
  },
  selectSuggestedName(event: WechatMiniprogram.TouchEvent) {
    if (this.data.submitting) return;
    const name = String(event.currentTarget.dataset.name);
    this.setData({ name, nameLength: name.length, creationError: '' });
  },
  togglePresetScore(event: WechatMiniprogram.TouchEvent) {
    if (this.data.submitting) return;
    const value = Number(event.currentTarget.dataset.value);
    const presetOptions = this.data.presetOptions.map((option) => ({ ...option }));
    const target = presetOptions.find((option) => option.value === value);
    if (!target) return;
    const selectedCount = presetOptions.filter((option) => option.selected).length;
    if (target.selected) {
      if (selectedCount <= 1) {
        wx.showToast({ title: '至少保留 1 个分值', icon: 'none' });
        return;
      }
      target.selected = false;
    } else {
      if (selectedCount >= MAX_PRESET_SCORES) {
        wx.showToast({ title: `最多选 ${MAX_PRESET_SCORES} 个分值`, icon: 'none' });
        return;
      }
      target.selected = true;
    }
    this.setData({ presetOptions, creationError: '' });
  },
  async submit() {
    if (this.data.submitting) return;
    const name = String(this.data.name).trim();
    if (!name) return wx.showToast({ title: '请输入牌局名称', icon: 'none' });

    const presetScores = selectedPresetScores(this.data.presetOptions);
    this.setData({ submitting: true, creationError: '' });
    try {
      await requireLogin();
      const game = await createGame(name, null, presetScores);
      saveRecentSession(game.id);
      wx.switchTab({ url: '/pages/home/index' });
    } catch (err) {
      this.setData({ creationError: (err as any).message || '创建失败，请重试' });
    } finally {
      this.setData({ submitting: false });
    }
  },
});
