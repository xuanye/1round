import { requireLogin } from '../../services/auth.service';
import { joinPreview, joinGame, getCurrentGame, getSummary, leaveGame } from '../../services/game.service';
import { getUser, saveRecentSession } from '../../utils/storage';
import { isSystemDisplayName } from '../../utils/user-profile';

type ParticipantPreview = {
  id: string;
  displayName: string;
};

Page({
  data: {
    icons: {
      dice: '\uf522',
      close: '\uf00d',
      users: '\uf0c0',
      edit: '\uf304',
      info: '\uf05a',
      check: '\uf058',
      arrowRight: '\uf061',
    },
    inviteCode: '',
    game: null as {
      id: string;
      name: string;
      creatorName: string;
      creatorInitial: string;
      participantText: string;
    } | null,
    participants: [] as ParticipantPreview[],
    participantSummary: '',
    displayName: '',
    joining: false,
    joined: false,

    // Conflict game state
    conflictGame: null as { id: string; name: string; canLeave: boolean } | null,
  },

  async onLoad(query: Record<string, string>) {
    let inviteCode = query.inviteCode || '';
    if (query.scene) {
      const scene = decodeURIComponent(query.scene);
      const match = scene.match(/(?:code=)?([A-Za-z0-9]+)/);
      if (match) {
        inviteCode = match[1];
      }
    }
    if (!inviteCode) {
      wx.showToast({ title: '邀请码缺失', icon: 'none' });
      setTimeout(() => wx.redirectTo({ url: '/pages/home/index' }), 1500);
      return;
    }
    this.setData({ inviteCode });

    wx.showLoading({ title: '正在获取预览...' });
    try {
      await requireLogin();

      // Check join preview
      const preview = await joinPreview(inviteCode);
      if (preview.alreadyJoined) {
        // Redirection for users already joined
        wx.redirectTo({ url: `/pages/game-detail/index?id=${preview.gameSessionId}&inviteCode=${inviteCode}` });
        return;
      }

      // Check if user has another active game
      const current = await getCurrentGame();
      if (current?.id && current.id !== preview.gameSessionId) {
        const summary = await getSummary(current.id);
        const user = getUser();
        let myScore = 0;
        const myPlayer = summary.players.find(p => p.userId === user?.id || (p.displayName === user?.displayName && p.userId === user?.id));
        if (myPlayer) myScore = myPlayer.totalScore;
        this.setData({
          conflictGame: {
            id: current.id,
            name: current.name,
            canLeave: myScore === 0,
          },
        });
      }

      const ownerInit = preview.ownerDisplayName ? preview.ownerDisplayName.slice(0, 1) : '创';
      const preferredDisplayName = !isSystemDisplayName(preview.currentUserDisplayName)
        ? preview.currentUserDisplayName
        : (getUser()?.displayName || preview.currentUserDisplayName);

      this.setData({
        game: {
          id: preview.gameSessionId,
          name: preview.name,
          creatorName: preview.ownerDisplayName,
          creatorInitial: ownerInit,
          participantText: `${preview.participantCount}${preview.maxParticipants ? ' / ' + preview.maxParticipants : ''}`,
        },
        participants: preview.participants.map(p => ({
          id: p.id,
          displayName: p.displayName,
        })),
        participantSummary: preview.participants.map((p) => p.displayName).slice(0, 4).join('、'),
        displayName: preferredDisplayName,
      });
    } catch (err) {
      wx.showModal({
        title: '加入失败',
        content: (err as any).message || '获取牌局预览失败',
        showCancel: false,
        success: () => wx.redirectTo({ url: '/pages/home/index' }),
      });
    } finally {
      wx.hideLoading();
    }
  },

  onNameInput(event: WechatMiniprogram.Input) {
    this.setData({ displayName: String(event.detail.value) });
  },

  cancel() {
    wx.redirectTo({ url: '/pages/home/index' });
  },

  async submit() {
    const displayName = String(this.data.displayName).trim();
    if (!displayName) {
      wx.showToast({ title: '请输入显示名称', icon: 'none' });
      return;
    }

    this.setData({ joining: true });
    wx.showLoading({ title: '正在加入牌局...' });

    try {
      await requireLogin();

      // If there is a conflict game, we must leave it first (and we only get here if canLeave is true)
      if (this.data.conflictGame) {
        if (!this.data.conflictGame.canLeave) {
          throw new Error('原当前牌局分值不为 0，不能退出并加入新牌局');
        }
        wx.showLoading({ title: '正在退出原牌局...' });
        await leaveGame(this.data.conflictGame.id);
        wx.showLoading({ title: '正在加入新牌局...' });
      }

      const res = await joinGame(this.data.inviteCode, displayName);
      saveRecentSession(res.gameSessionId); // Save recent session!
      this.setData({ joining: false, joined: true });
      wx.showToast({ title: '已加入牌局', icon: 'success', duration: 500 });
      wx.redirectTo({ url: `/pages/game-detail/index?id=${res.gameSessionId}&inviteCode=${this.data.inviteCode}` });
    } catch (err) {
      this.setData({ joining: false });
      wx.showToast({ title: (err as any).message || '加入失败', icon: 'none' });
    } finally {
      wx.hideLoading();
    }
  },
});
