export type RoundScore = {
  playerId: string;
  score: number;
};

export type RecentRound = {
  id: string;
  roundNo: number;
  createdAt: string;
  scores: RoundScore[];
};
