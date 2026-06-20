import type {
  GameSession,
  GameSummary,
  JoinPreview,
  SettlementDetail,
  PublicSettlement,
  RankingItem,
  HistoryPage,
  PendingFinishRequest,
} from '../models/game-session';
import type { Player } from '../models/player';
import type { ScoreTransfer } from '../models/score-transfer';
import { request, requestBinary } from './http';

export function createGame(name: string, maxParticipants: number | null): Promise<GameSession> {
  return request({ url: '/api/game-sessions', method: 'POST', data: { name, maxParticipants } });
}

export function getCurrentGame(): Promise<GameSession | null> {
  return request<GameSession | null>({ url: '/api/game-sessions/current' });
}

export function joinGame(inviteCode: string, displayName: string): Promise<{ gameSessionId: string }> {
  return request({ url: '/api/game-sessions/join', method: 'POST', data: { inviteCode, displayName } });
}

export function joinPreview(inviteCode: string): Promise<JoinPreview> {
  return request({ url: '/api/game-sessions/join-preview', method: 'POST', data: { inviteCode } });
}

export function getSummary(gameSessionId: string): Promise<GameSummary> {
  return request({ url: `/api/game-sessions/${gameSessionId}/summary` });
}

export function getRanking(gameSessionId: string): Promise<RankingItem[]> {
  return request({ url: `/api/game-sessions/${gameSessionId}/ranking` });
}

export function getScoreTransfers(gameSessionId: string, beforeSequenceNo?: number, limit?: number): Promise<ScoreTransfer[]> {
  const query: string[] = [];
  if (beforeSequenceNo !== undefined) query.push(`beforeSequenceNo=${beforeSequenceNo}`);
  if (limit) query.push(`limit=${limit}`);
  const queryString = query.length > 0 ? `?${query.join('&')}` : '';
  return request({ url: `/api/game-sessions/${gameSessionId}/score-transfers${queryString}` });
}

export function finishGameDirect(gameSessionId: string): Promise<GameSession> {
  return request({ url: `/api/game-sessions/${gameSessionId}/finish`, method: 'POST' });
}

export function requestFinish(gameSessionId: string): Promise<PendingFinishRequest> {
  return request({ url: `/api/game-sessions/${gameSessionId}/finish-requests`, method: 'POST' });
}

export function approveFinishRequest(gameSessionId: string, requestId: string): Promise<GameSession> {
  return request({ url: `/api/game-sessions/${gameSessionId}/finish-requests/${requestId}/approve`, method: 'POST' });
}

export function rejectFinishRequest(gameSessionId: string, requestId: string): Promise<PendingFinishRequest> {
  return request({ url: `/api/game-sessions/${gameSessionId}/finish-requests/${requestId}/reject`, method: 'POST' });
}

export function leaveGame(gameSessionId: string): Promise<void> {
  return request({ url: `/api/game-sessions/${gameSessionId}/leave`, method: 'POST' });
}

export function updateMyProfile(gameSessionId: string, displayName: string): Promise<Player> {
  return request({ url: `/api/game-sessions/${gameSessionId}/my-profile`, method: 'PATCH', data: { displayName } });
}

export function getHistory(beforeSettledAt?: string, limit?: number): Promise<HistoryPage> {
  const query: string[] = [];
  if (beforeSettledAt) query.push(`beforeSettledAt=${encodeURIComponent(beforeSettledAt)}`);
  if (limit) query.push(`limit=${limit}`);
  const queryString = query.length > 0 ? `?${query.join('&')}` : '';
  return request({ url: `/api/history/game-sessions${queryString}` });
}

export function getSettlementDetail(gameSessionId: string, beforeSequenceNo?: number, limit?: number): Promise<SettlementDetail> {
  const query: string[] = [];
  if (beforeSequenceNo !== undefined) query.push(`beforeSequenceNo=${beforeSequenceNo}`);
  if (limit) query.push(`limit=${limit}`);
  const queryString = query.length > 0 ? `?${query.join('&')}` : '';
  return request({ url: `/api/history/game-sessions/${gameSessionId}${queryString}` });
}

export function getPublicSettlement(shareToken: string): Promise<PublicSettlement> {
  return request({ url: `/api/public/settlements/${shareToken}`, auth: false });
}

export function getHistoryStats(): Promise<{ totalGames: number; maxScore: number }> {
  return request({ url: '/api/history/stats' });
}

export async function getJoinMiniProgramCode(gameSessionId: string): Promise<string> {
  const imageData = await requestBinary({ url: `/api/game-sessions/${gameSessionId}/join-mini-program-code` });
  const fileSystemManager = wx.getFileSystemManager();
  const filePath = `${wx.env.USER_DATA_PATH}/oneround-join-${gameSessionId}.png`;
  const base64 = wx.arrayBufferToBase64(imageData);
  fileSystemManager.writeFileSync(filePath, base64, 'base64');
  return filePath;
}
