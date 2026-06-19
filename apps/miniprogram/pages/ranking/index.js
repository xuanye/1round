"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const game_service_1 = require("../../services/game.service");
Page({
    data: { players: [] },
    async onLoad(query) {
        const summary = await (0, game_service_1.getSummary)(query.id);
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
//# sourceMappingURL=index.js.map