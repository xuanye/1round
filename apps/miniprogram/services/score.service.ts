import { request } from './http';

export function submitScoreTransfer(
  gameSessionId: string,
  receiverPlayerIds: string[],
  amount: number,
  idempotencyKey: string
): Promise<{ id: string; sequenceNo: number; version: number }> {
  return request({
    url: `/api/game-sessions/${gameSessionId}/score-transfers`,
    method: 'POST',
    data: { receiverPlayerIds, amount, idempotencyKey },
  });
}
