import type { Player } from './player';
import type { ScoreTransfer } from './score-transfer';

export type PendingFinishRequest = {
  id: string;
  requestedByPlayerId: string;
  requestedByName: string;
  createdAt: string;
};

export type GameSession = {
  id: string;
  name: string;
  inviteCode: string;
  ownerUserId: string;
  status: 'active' | 'finished';
  maxParticipants: number | null;
  scoreTransferCount: number;
  version: number;
};

export type GameSummary = {
  id: string;
  name: string;
  inviteCode: string;
  ownerUserId: string;
  status: 'active' | 'finished';
  scoreTransferCount: number;
  players: Player[];
  scoreTransfers?: ScoreTransfer[];
  updatedAt: string;
  version: number;
  pendingFinishRequest?: PendingFinishRequest;
  publicShareToken?: string;
};

export type PlayerPreview = {
  id: string;
  displayName: string;
};

export type JoinPreview = {
  gameSessionId: string;
  name: string;
  ownerDisplayName: string;
  participantCount: number;
  maxParticipants: number | null;
  participants: PlayerPreview[];
  currentUserDisplayName: string;
  alreadyJoined: boolean;
};

export type SettlementParticipant = {
  id: string;
  displayName: string;
  avatarUrl?: string;
  finalScore: number;
};

export type SettlementDetail = {
  id: string;
  name: string;
  settledAt: string;
  participants: SettlementParticipant[];
  scoreTransfers: ScoreTransfer[];
  nextCursor?: number;
  publicShareToken?: string;
};

export type PublicSettlement = {
  gameSessionId: string;
  name: string;
  settledAt: string;
  participants: SettlementParticipant[];
  scoreTransfers: ScoreTransfer[];
};

export type HistoryItem = {
  id: string;
  name: string;
  settledAt: string;
  scoreTransferCount: number;
  myFinalScore: number;
};

export type HistoryPage = {
  items: HistoryItem[];
  nextCursor?: string;
};

export type RankingItem = {
  rank: number;
  playerId: string;
  displayName: string;
  totalScore: number;
  scoreTransferCount: number;
  averageScore: number;
};
