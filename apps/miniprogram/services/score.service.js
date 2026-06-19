"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.submitRound = submitRound;
const http_1 = require("./http");
function submitRound(gameSessionId, scores, note) {
    return (0, http_1.request)({
        url: `/api/game-sessions/${gameSessionId}/rounds`,
        method: 'POST',
        data: { scores, note },
    });
}
//# sourceMappingURL=score.service.js.map