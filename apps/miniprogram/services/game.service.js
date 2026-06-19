"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createGame = createGame;
exports.getCurrentGame = getCurrentGame;
exports.joinGame = joinGame;
exports.joinPreview = joinPreview;
exports.getSummary = getSummary;
exports.getRanking = getRanking;
exports.getScoreTransfers = getScoreTransfers;
exports.finishGameDirect = finishGameDirect;
exports.requestFinish = requestFinish;
exports.approveFinishRequest = approveFinishRequest;
exports.rejectFinishRequest = rejectFinishRequest;
exports.leaveGame = leaveGame;
exports.updateMyProfile = updateMyProfile;
exports.getHistory = getHistory;
exports.getSettlementDetail = getSettlementDetail;
exports.getPublicSettlement = getPublicSettlement;
exports.getHistoryStats = getHistoryStats;
const http_1 = require("./http");
function createGame(name, maxParticipants) {
    return (0, http_1.request)({ url: '/api/game-sessions', method: 'POST', data: { name, maxParticipants } });
}
function getCurrentGame() {
    return (0, http_1.request)({ url: '/api/game-sessions/current' });
}
function joinGame(inviteCode, displayName) {
    return (0, http_1.request)({ url: '/api/game-sessions/join', method: 'POST', data: { inviteCode, displayName } });
}
function joinPreview(inviteCode) {
    return (0, http_1.request)({ url: '/api/game-sessions/join-preview', method: 'POST', data: { inviteCode } });
}
function getSummary(gameSessionId) {
    return (0, http_1.request)({ url: `/api/game-sessions/${gameSessionId}/summary` });
}
function getRanking(gameSessionId) {
    return (0, http_1.request)({ url: `/api/game-sessions/${gameSessionId}/ranking` });
}
function getScoreTransfers(gameSessionId, beforeSequenceNo, limit) {
    const query = [];
    if (beforeSequenceNo !== undefined)
        query.push(`beforeSequenceNo=${beforeSequenceNo}`);
    if (limit)
        query.push(`limit=${limit}`);
    const queryString = query.length > 0 ? `?${query.join('&')}` : '';
    return (0, http_1.request)({ url: `/api/game-sessions/${gameSessionId}/score-transfers${queryString}` });
}
function finishGameDirect(gameSessionId) {
    return (0, http_1.request)({ url: `/api/game-sessions/${gameSessionId}/finish`, method: 'POST' });
}
function requestFinish(gameSessionId) {
    return (0, http_1.request)({ url: `/api/game-sessions/${gameSessionId}/finish-requests`, method: 'POST' });
}
function approveFinishRequest(gameSessionId, requestId) {
    return (0, http_1.request)({ url: `/api/game-sessions/${gameSessionId}/finish-requests/${requestId}/approve`, method: 'POST' });
}
function rejectFinishRequest(gameSessionId, requestId) {
    return (0, http_1.request)({ url: `/api/game-sessions/${gameSessionId}/finish-requests/${requestId}/reject`, method: 'POST' });
}
function leaveGame(gameSessionId) {
    return (0, http_1.request)({ url: `/api/game-sessions/${gameSessionId}/leave`, method: 'POST' });
}
function updateMyProfile(gameSessionId, displayName) {
    return (0, http_1.request)({ url: `/api/game-sessions/${gameSessionId}/my-profile`, method: 'PATCH', data: { displayName } });
}
function getHistory(beforeSettledAt, limit) {
    const query = [];
    if (beforeSettledAt)
        query.push(`beforeSettledAt=${encodeURIComponent(beforeSettledAt)}`);
    if (limit)
        query.push(`limit=${limit}`);
    const queryString = query.length > 0 ? `?${query.join('&')}` : '';
    return (0, http_1.request)({ url: `/api/history/game-sessions${queryString}` });
}
function getSettlementDetail(gameSessionId, beforeSequenceNo, limit) {
    const query = [];
    if (beforeSequenceNo !== undefined)
        query.push(`beforeSequenceNo=${beforeSequenceNo}`);
    if (limit)
        query.push(`limit=${limit}`);
    const queryString = query.length > 0 ? `?${query.join('&')}` : '';
    return (0, http_1.request)({ url: `/api/history/game-sessions/${gameSessionId}${queryString}` });
}
function getPublicSettlement(shareToken) {
    return (0, http_1.request)({ url: `/api/public/settlements/${shareToken}`, auth: false });
}
function getHistoryStats() {
    return (0, http_1.request)({ url: '/api/history/stats' });
}
//# sourceMappingURL=game.service.js.map