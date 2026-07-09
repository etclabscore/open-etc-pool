import { get } from 'svelte/store';
import { config } from './config';
import type {
  StatsResponse,
  MinersResponse,
  BlocksResponse,
  PaymentsResponse,
  AccountResponse,
} from './types';

export class NotFoundError extends Error {
  constructor() {
    super('not found');
    this.name = 'NotFoundError';
  }
}

function base(): string {
  let u = get(config).apiUrl || '/';
  if (!u.endsWith('/')) u += '/';
  return u;
}

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(base() + path, { headers: { Accept: 'application/json' } });
  if (res.status === 404) throw new NotFoundError();
  if (!res.ok) throw new Error('HTTP ' + res.status);
  return (await res.json()) as T;
}

export const getStats = () => getJSON<StatsResponse>('api/stats');
export const getMiners = () => getJSON<MinersResponse>('api/miners');
export const getBlocks = () => getJSON<BlocksResponse>('api/blocks');
export const getPayments = () => getJSON<PaymentsResponse>('api/payments');
export const getAccount = (login: string) => getJSON<AccountResponse>('api/accounts/' + login);
