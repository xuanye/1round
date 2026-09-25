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
  initiatedByName?: string;
  scoreChanges?: ScoreChange[];
};

export type ScoreChange = {
  playerId: string;
  playerName: string;
  before: number;
  after: number;
  delta: number;
};
