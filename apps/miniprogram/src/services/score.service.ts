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

export function reverseScoreTransfer(
  gameSessionId: string,
  transferId: string,
  idempotencyKey: string,
  reason: string
): Promise<{ id: string; sequenceNo: number; version: number }> {
  return request({
    url: `/api/game-sessions/${gameSessionId}/score-transfers/${transferId}/reversal`,
    method: 'POST',
    data: { idempotencyKey, reason },
  });
}
