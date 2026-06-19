import type { Player } from './player';
import type { RecentRound } from './round';

export type GameSession = {
  id: string;
  name: string;
  inviteCode: string;
  status: 'active' | 'finished';
  zeroSumRequired: boolean;
  roundCount: number;
  version: number;
};

export type GameSummary = {
  id: string;
  name: string;
  status: 'active' | 'finished';
  roundCount: number;
  zeroSumRequired: boolean;
  players: Player[];
  recentRounds: RecentRound[];
  updatedAt: string;
  version: number;
};
