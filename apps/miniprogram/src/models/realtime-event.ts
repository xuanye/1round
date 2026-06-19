export type RealtimeEvent = {
  type: string;
  gameSessionId: string;
  version: number;
  payload?: unknown;
  sentAt: string;
};
