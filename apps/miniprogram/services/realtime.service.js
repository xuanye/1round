"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.RealtimeService = void 0;
const storage_1 = require("../utils/storage");
class RealtimeService {
    constructor() {
        this.socket = null;
        this.gameSessionId = '';
        this.handlers = [];
        this.reconnectTimer = null;
        this.closedByPage = false;
    }
    connect(gameSessionId) {
        this.disconnect();
        this.closedByPage = false;
        this.gameSessionId = gameSessionId;
        const app = getApp();
        const url = app.globalData.baseUrl.replace(/^http/, 'ws');
        const token = encodeURIComponent((0, storage_1.getToken)());
        this.socket = wx.connectSocket({ url: `${url}/ws/game-sessions/${gameSessionId}?token=${token}` });
        this.socket.onMessage((message) => {
            const event = JSON.parse(String(message.data));
            this.handlers.forEach((handler) => handler(event));
        });
        this.socket.onClose(() => this.scheduleReconnect());
        this.socket.onError(() => this.scheduleReconnect());
    }
    disconnect() {
        this.closedByPage = true;
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        if (this.socket) {
            this.socket.close({});
            this.socket = null;
        }
    }
    onEvent(handler) {
        this.handlers.push(handler);
    }
    scheduleReconnect() {
        if (this.closedByPage || this.reconnectTimer)
            return;
        this.reconnectTimer = setTimeout(() => {
            this.reconnectTimer = null;
            if (this.gameSessionId)
                this.connect(this.gameSessionId);
        }, 1500);
    }
}
exports.RealtimeService = RealtimeService;
//# sourceMappingURL=realtime.service.js.map