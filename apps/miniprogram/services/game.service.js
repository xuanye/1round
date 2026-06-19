"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createGame = createGame;
exports.joinGame = joinGame;
exports.getSummary = getSummary;
exports.addPlayer = addPlayer;
exports.finishGame = finishGame;
const http_1 = require("./http");
function createGame(name, zeroSumRequired) {
    return (0, http_1.request)({ url: '/api/game-sessions', method: 'POST', data: { name, zeroSumRequired } });
}
function joinGame(inviteCode) {
    return (0, http_1.request)({ url: '/api/game-sessions/join', method: 'POST', data: { inviteCode } });
}
function getSummary(gameSessionId) {
    return (0, http_1.request)({ url: `/api/game-sessions/${gameSessionId}/summary` });
}
function addPlayer(gameSessionId, displayName) {
    return (0, http_1.request)({ url: `/api/game-sessions/${gameSessionId}/players`, method: 'POST', data: { displayName } });
}
function finishGame(gameSessionId) {
    return (0, http_1.request)({ url: `/api/game-sessions/${gameSessionId}/finish`, method: 'POST' });
}
//# sourceMappingURL=game.service.js.map