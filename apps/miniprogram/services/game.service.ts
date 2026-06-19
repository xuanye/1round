import type { GameSession, GameSummary } from '../models/game-session';
import type { Player } from '../models/player';
import { request } from './http';

export function createGame(name: string, zeroSumRequired: boolean): Promise<GameSession> {
  return request({ url: '/api/game-sessions', method: 'POST', data: { name, zeroSumRequired } });
}

export function joinGame(inviteCode: string): Promise<{ gameSessionId: string }> {
  return request({ url: '/api/game-sessions/join', method: 'POST', data: { inviteCode } });
}

export function getSummary(gameSessionId: string): Promise<GameSummary> {
  return request({ url: `/api/game-sessions/${gameSessionId}/summary` });
}

export function addPlayer(gameSessionId: string, displayName: string): Promise<Player> {
  return request({ url: `/api/game-sessions/${gameSessionId}/players`, method: 'POST', data: { displayName } });
}

export function finishGame(gameSessionId: string): Promise<{ id: string; status: string }> {
  return request({ url: `/api/game-sessions/${gameSessionId}/finish`, method: 'POST' });
}
