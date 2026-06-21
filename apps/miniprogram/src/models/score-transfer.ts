export type ScoreTransfer = {
  id: string;
  sequenceNo: number;
  fromPlayerId: string;
  receiverPlayerIds: string[];
  amount: number;
  createdAt: string;
  text: string;
  transferKind?: string;
  reversalOfTransferId?: string;
  reversedAt?: string;
};
