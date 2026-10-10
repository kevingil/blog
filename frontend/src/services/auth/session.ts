import { atom, getDefaultStore } from 'jotai';
import type { User } from '../types';

const TOKEN_KEY = 'token';
const USER_KEY = 'user';

export interface Session {
  token: string;
  user: User;
  /** Epoch milliseconds from the token's `iat` and `exp` claims. */
  issuedAt: number;
  expiresAt: number;
}

function tokenClaims(token: string): { iat?: unknown; exp?: unknown } | null {
  const payload = token.split('.')[1];
  if (!payload) return null;
  try {
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/');
    return JSON.parse(atob(base64.padEnd(Math.ceil(base64.length / 4) * 4, '=')));
  } catch {
    return null;
  }
}

/** Null when the token carries no expiry, which no server-issued token lacks. */
export function sessionFromToken(token: string, user: User): Session | null {
  const claims = tokenClaims(token);
  if (typeof claims?.exp !== 'number') return null;
  return {
    token,
    user,
    issuedAt: typeof claims.iat === 'number' ? claims.iat * 1000 : 0,
    expiresAt: claims.exp * 1000,
  };
}

function storedSession(): Session | null {
  if (typeof window === 'undefined') return null;
  const token = localStorage.getItem(TOKEN_KEY);
  const user = localStorage.getItem(USER_KEY);
  if (!token || !user) return null;
  try {
    const session = sessionFromToken(token, JSON.parse(user) as User);
    return session && session.expiresAt > Date.now() ? session : null;
  } catch {
    return null;
  }
}

const store = getDefaultStore();

export const sessionAtom = atom<Session | null>(storedSession());
export const isAuthenticatedAtom = atom((get) => get(sessionAtom) !== null);

/** The signed-in session, or null. Reading an expired session ends it. */
export function currentSession(now = Date.now()): Session | null {
  const session = store.get(sessionAtom);
  if (session && session.expiresAt <= now) {
    endSession();
    return null;
  }
  return session;
}

export function saveSession(token: string, user: User): Session | null {
  const session = sessionFromToken(token, user);
  if (!session) {
    endSession();
    return null;
  }
  localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(USER_KEY, JSON.stringify(user));
  store.set(sessionAtom, session);
  return session;
}

export function endSession() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
  if (store.get(sessionAtom) !== null) {
    store.set(sessionAtom, null);
  }
}

/** Adopts a sign-in, sign-out, or refresh that happened in another tab. */
export function syncSessionFromStorage() {
  const stored = storedSession();
  if (stored?.token !== store.get(sessionAtom)?.token) {
    store.set(sessionAtom, stored);
  }
}
