Component({
  properties: {
    players: { type: Array, value: [] },
  },
  data: {
    displayPlayers: [] as Array<Record<string, unknown>>,
  },
  observers: {
    players(players: Array<{ displayName?: string; totalScore?: number; averageScore?: number }>) {
      this.setData({
        displayPlayers: (players || []).map((player) => ({
          ...player,
          initial: (player.displayName || '?').slice(0, 1),
          scoreLabel: `${(player.totalScore || 0) >= 0 ? '+' : ''}${player.totalScore || 0}`,
          scoreTone: (player.totalScore || 0) >= 0 ? 'positive' : 'negative',
          averageLabel: String(player.averageScore || 0),
        })),
      });
    },
  },
});
