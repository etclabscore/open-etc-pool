// Typed shapes of the open-etc-pool HTTP API (api/server.go). Money fields are
// integers in Shannon (Gwei = 1e9 Wei) unless noted; block reward is a decimal
// string in Wei. Timestamps from Redis are unix SECONDS; the top-level `now` is
// unix MILLISECONDS. Hashrate fields are hashes/second. Many top-level keys are
// optional: they are absent until the first background stats snapshot loads.

export interface NodeState {
  name: string;
  height: string; // decimal string
  difficulty: string; // decimal string, can be very large
  lastBeat: string; // unix seconds, as string
}

export interface Block {
  height: number;
  timestamp: number; // unix seconds
  difficulty: number;
  shares: number;
  uncle: boolean;
  uncleHeight: number; // 0 when not an uncle
  orphan: boolean;
  hash: string; // 0x… ; "" for candidates
  reward: string; // decimal Wei ; "" for candidates
}

export interface PoolPayment {
  timestamp: number; // unix seconds
  tx: string;
  address: string;
  amount: number; // Shannon
}

export interface AccountPayment {
  timestamp: number; // unix seconds
  tx: string;
  amount: number; // Shannon
}

export interface Miner {
  lastBeat: number; // unix seconds
  hr: number; // H/s, small window
  offline: boolean;
}

export interface Worker {
  lastBeat: number; // unix seconds
  hr: number; // H/s, small window (30m)
  hr2: number; // H/s, large window (3h)
  offline: boolean;
}

export type PoolStats = Record<string, number | string>;
export type Luck = Record<string, { luck: number; uncleRate: number; orphanRate: number }>;

export interface StatsResponse {
  nodes: NodeState[];
  now?: number;
  stats?: PoolStats; // includes lastBlockFound (unix seconds) and roundShares
  hashrate?: number;
  minersTotal?: number;
  maturedTotal?: number;
  immatureTotal?: number;
  candidatesTotal?: number;
}

export interface MinersResponse {
  now?: number;
  miners?: Record<string, Miner>;
  hashrate?: number;
  minersTotal?: number;
}

export interface BlocksResponse {
  matured?: Block[];
  maturedTotal?: number;
  immature?: Block[];
  immatureTotal?: number;
  candidates?: Block[];
  candidatesTotal?: number;
  luck?: Luck;
}

export interface PaymentsResponse {
  payments?: PoolPayment[];
  paymentsTotal?: number;
}

export type AccountStats = Record<string, number | string>;

export interface AccountResponse {
  stats: AccountStats; // balance, immature, paid, pending (Shannon), blocksFound, lastShare (unix seconds)
  payments: AccountPayment[];
  paymentsTotal: number;
  roundShares: number;
  workers: Record<string, Worker>;
  workersTotal: number;
  workersOnline: number;
  workersOffline: number;
  hashrate: number; // Σ hr2 (3h)
  currentHashrate: number; // Σ hr (30m)
  pageSize: number;
}
