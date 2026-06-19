"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.submitScoreTransfer = submitScoreTransfer;
const http_1 = require("./http");
function submitScoreTransfer(gameSessionId, receiverPlayerIds, amount, idempotencyKey) {
    return (0, http_1.request)({
        url: `/api/game-sessions/${gameSessionId}/score-transfers`,
        method: 'POST',
        data: { receiverPlayerIds, amount, idempotencyKey },
    });
}
//# sourceMappingURL=score.service.js.map