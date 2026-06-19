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
        const baseUrl = app.globalData.baseUrl;
        const wsProto = baseUrl.startsWith('https:') ? 'wss:' : 'ws:';
        const cleanUrl = baseUrl.replace(/^https?:\/\//, '');
        const socketUrl = `${wsProto}//${cleanUrl}/ws/game-sessions/${gameSessionId}?token=${encodeURIComponent((0, storage_1.getToken)())}`;
        this.socket = wx.connectSocket({ url: socketUrl });
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
        this.handlers = []; // Fix listener leak by clearing handlers on disconnect
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