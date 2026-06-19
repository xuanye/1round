"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getToken = getToken;
exports.setToken = setToken;
exports.clearToken = clearToken;
exports.saveRecentSession = saveRecentSession;
exports.getRecentSessions = getRecentSessions;
const TOKEN_KEY = 'one_round_token';
const RECENT_KEY = 'one_round_recent_sessions';
function getToken() {
    return wx.getStorageSync(TOKEN_KEY) || '';
}
function setToken(token) {
    wx.setStorageSync(TOKEN_KEY, token);
}
function clearToken() {
    wx.removeStorageSync(TOKEN_KEY);
}
function saveRecentSession(gameSessionId) {
    const list = getRecentSessions().filter((id) => id !== gameSessionId);
    wx.setStorageSync(RECENT_KEY, [gameSessionId, ...list].slice(0, 20));
}
function getRecentSessions() {
    return wx.getStorageSync(RECENT_KEY) || [];
}
//# sourceMappingURL=storage.js.map