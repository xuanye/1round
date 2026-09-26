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
  presetScores?: number[];
  scoreTransferCount: number;
  version: number;
  createdAt?: string;
};

export type RoundStatus = {
  roundNo: number;
  status: string;
  pendingPlayerIds: string[];
  pendingPlayerNames: string[];
  canStartNextRound: boolean;
};

export type GameSummary = {
  id: string;
  name: string;
  inviteCode: string;
  ownerUserId: string;
  status: 'active' | 'finished';
  presetScores?: number[];
  scoreTransferCount: number;
  players: Player[];
  scoreTransfers?: ScoreTransfer[];
  updatedAt: string;
  version: number;
  pendingFinishRequest?: PendingFinishRequest;
  publicShareToken?: string;
  roundStatus?: RoundStatus;
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
};

export type HistoryItem = {
  id: string;
  name: string;
  settledAt: string;
  scoreTransferCount: number;
  myFinalScore: number;
  participantCount?: number;
  winnerName?: string;
  winnerScore?: number;
  createdAt?: string;
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

export type Performance = {
  totalScore: number;
  totalGames: number;
  wins: number;
  maxScore: number;
  trend: { settledAt: string; score: number }[];
  players: { id: string; displayName: string; avatarUrl: string; totalScore: number; isMe: boolean }[];
  recentGames: { id: string; name: string; settledAt: string; participantCount: number; myFinalScore: number }[];
};
