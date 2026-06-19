import type { Player } from '../../models/player';
import { getSummary } from '../../services/game.service';

type RankedPlayer = Player & {
  initial: string;
  scoreLabel: string;
  scoreTone: 'positive' | 'negative';
  averageLabel: string;
};

Page({
  data: { players: [] as RankedPlayer[] },
  async onLoad(query: Record<string, string>) {
    const summary = await getSummary(query.id);
    this.setData({
      players: summary.players.map((player) => ({
        ...player,
        initial: player.displayName.slice(0, 1),
        scoreLabel: `${player.totalScore >= 0 ? '+' : ''}${player.totalScore}`,
        scoreTone: player.totalScore >= 0 ? 'positive' : 'negative',
        averageLabel: String(player.averageScore || 0),
      })),
    });
  },
});
