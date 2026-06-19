import { request } from './http';

export type ScoreInput = {
  playerId: string;
  score: number;
};

export function submitRound(gameSessionId: string, scores: ScoreInput[], note?: string): Promise<{ roundId: string; roundNo: number; version: number }> {
  return request({
    url: `/api/game-sessions/${gameSessionId}/rounds`,
    method: 'POST',
    data: { scores, note },
  });
}
