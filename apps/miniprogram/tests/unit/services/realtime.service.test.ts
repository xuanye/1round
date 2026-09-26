import { loadSource } from '../../helpers/page';

jest.mock('../../../src/utils/storage', () => ({
  getToken: () => 'test-token',
}));

describe('realtime.service', () => {
  it('retains event notifications after an automatic reconnect', () => {
    const sockets: Array<Record<string, any>> = [];
    const wx = {
      connectSocket() {
        const socket = {
          message: undefined as ((value: unknown) => void) | undefined,
          closed: undefined as (() => void) | undefined,
          error: undefined as (() => void) | undefined,
          onMessage(callback: (value: unknown) => void) { this.message = callback; },
          onClose(callback: () => void) { this.closed = callback; },
          onError(callback: () => void) { this.error = callback; },
          close() {},
        };
        sockets.push(socket as Record<string, any>);
        return socket;
      },
    };
    const { exports } = loadSource('services/realtime.service.ts', {
      wx,
      getApp: () => ({ globalData: { baseUrl: 'https://example.test' } }),
    });
    const service = new exports.RealtimeService();
    let notifications = 0;

    jest.useFakeTimers();
    try {
      service.connect('game-1');
      service.onEvent(() => { notifications++; });
      sockets[0].message({ data: '{"type":"score_transfer.submitted"}' });
      sockets[0].closed();
      // Reconnect is scheduled on a timer; fire it instead of waiting 1500ms.
      jest.runOnlyPendingTimers();
      sockets[1].message({ data: '{"type":"score_transfer.submitted"}' });
      expect(notifications).toBe(2);
    } finally {
      jest.useRealTimers();
    }
  });
});
