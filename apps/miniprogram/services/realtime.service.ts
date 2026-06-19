import type { RealtimeEvent } from '../models/realtime-event';
import { getToken } from '../utils/storage';

export class RealtimeService {
  private socket: WechatMiniprogram.SocketTask | null = null;
  private gameSessionId = '';
  private handlers: Array<(event: RealtimeEvent) => void> = [];
  private reconnectTimer: number | null = null;
  private closedByPage = false;

  connect(gameSessionId: string): void {
    this.disconnect();
    this.closedByPage = false;
    this.gameSessionId = gameSessionId;
    const app = getApp<{ globalData: { baseUrl: string } }>();
    const url = app.globalData.baseUrl.replace(/^http/, 'ws');
    const token = encodeURIComponent(getToken());
    this.socket = wx.connectSocket({ url: `${url}/ws/game-sessions/${gameSessionId}?token=${token}` });
    this.socket.onMessage((message) => {
      const event = JSON.parse(String(message.data)) as RealtimeEvent;
      this.handlers.forEach((handler) => handler(event));
    });
    this.socket.onClose(() => this.scheduleReconnect());
    this.socket.onError(() => this.scheduleReconnect());
  }

  disconnect(): void {
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

  onEvent(handler: (event: RealtimeEvent) => void): void {
    this.handlers.push(handler);
  }

  private scheduleReconnect(): void {
    if (this.closedByPage || this.reconnectTimer) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      if (this.gameSessionId) this.connect(this.gameSessionId);
    }, 1500);
  }
}
