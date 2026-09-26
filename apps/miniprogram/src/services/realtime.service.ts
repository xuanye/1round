import type { RealtimeEvent } from '../models/realtime-event';
import { getToken } from '../utils/storage';

export class RealtimeService {
  private socket: WechatMiniprogram.SocketTask | null = null;
  private gameSessionId = '';
  private handlers: Array<(event: RealtimeEvent) => void> = [];
  private reconnectTimer: number | null = null;
  private closedByPage = false;

  connect(gameSessionId: string): void {
    this.closeConnection();
    this.closedByPage = false;
    this.gameSessionId = gameSessionId;
    const app = getApp<{ globalData: { baseUrl: string } }>();
    const baseUrl = app.globalData.baseUrl;
    const wsProto = baseUrl.startsWith('https:') ? 'wss:' : 'ws:';
    const cleanUrl = baseUrl.replace(/^https?:\/\//, '');
    const socketUrl = `${wsProto}//${cleanUrl}/ws/game-sessions/${gameSessionId}?token=${encodeURIComponent(getToken())}`;
    const socket = wx.connectSocket({ url: socketUrl });
    this.socket = socket;
    socket.onMessage((message) => {
      if (this.socket !== socket) return;
      const event = JSON.parse(String(message.data)) as RealtimeEvent;
      this.handlers.forEach((handler) => handler(event));
    });
    socket.onClose(() => {
      if (this.socket !== socket) return;
      this.socket = null;
      this.scheduleReconnect();
    });
    socket.onError(() => {
      if (this.socket === socket) this.scheduleReconnect();
    });
  }

  disconnect(): void {
    this.closeConnection();
    this.handlers = [];
  }

  private closeConnection(): void {
    this.closedByPage = true;
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    const socket = this.socket;
    this.socket = null;
    if (socket) socket.close({});
  }

  onEvent(handler: (event: RealtimeEvent) => void): void {
    this.handlers.push(handler);
  }

  private scheduleReconnect(): void {
    if (this.closedByPage || this.reconnectTimer !== null) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      if (this.gameSessionId) this.connect(this.gameSessionId);
    }, 1500);
  }
}
