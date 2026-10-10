import { useAtomValue } from 'jotai';
import { createContext, useContext, useEffect, type ReactNode } from 'react';
import { Auth } from '@/client';
import type { User } from '../types';
import { generatedData } from '../generatedClient';
import {
  currentSession,
  endSession,
  saveSession,
  sessionAtom,
  syncSessionFromStorage,
  type Session,
} from './session';

export { isAuthenticatedAtom } from './session';

/** Tokens last 30 days. Renewing a day-old token keeps an active author signed in. */
const RENEW_AFTER_MS = 24 * 60 * 60 * 1000;
const EXPIRY_CHECK_MS = 60 * 1000;
const CONFIRM_TIMEOUT_MS = 5_000;

export interface AuthContext {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<User>;
  logout: () => Promise<void>;
  signOut: () => Promise<void>;
}

const AuthContext = createContext<AuthContext | null>(null);

/** The token the server accepted during this page load. */
let confirmedToken: string | null = null;
let refreshing: Promise<Session | null> | null = null;

function adopt(token: string, user: User): Session | null {
  const session = saveSession(token, user);
  confirmedToken = session?.token ?? null;
  return session;
}

/**
 * Trades the current token for a fresh one. A rejected token ends the session;
 * an unreachable server leaves it in place.
 */
export function refreshSession(signal?: AbortSignal): Promise<Session | null> {
  refreshing ??= generatedData<{ token: string; user: User }>(Auth.authRefresh({ signal }))
    .then(({ token, user }) => adopt(token, user))
    .catch(() => currentSession())
    .finally(() => {
      refreshing = null;
    });
  return refreshing;
}

/**
 * Confirms the stored session with the server once per page load, so the
 * dashboard never renders for a revoked token. Expired tokens are dropped
 * before any request.
 */
export async function ensureSession(): Promise<Session | null> {
  const session = currentSession();
  if (!session || session.token === confirmedToken) {
    return session;
  }
  // AbortSignal.timeout is missing before Safari 16; there the request has no deadline.
  const deadline =
    typeof AbortSignal.timeout === 'function' ? AbortSignal.timeout(CONFIRM_TIMEOUT_MS) : undefined;
  return refreshSession(deadline);
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const auth = useAuth();
  useSessionUpkeep();

  return (
    <AuthContext.Provider value={auth}>
      {children}
    </AuthContext.Provider>
  );
}

/** Ends a session the moment it expires and renews it while the author is active. */
function useSessionUpkeep() {
  useEffect(() => {
    const renewIfStale = () => {
      const session = currentSession();
      if (
        session &&
        document.visibilityState === 'visible' &&
        Date.now() - session.issuedAt > RENEW_AFTER_MS
      ) {
        void refreshSession();
      }
    };
    const expiry = window.setInterval(() => currentSession(), EXPIRY_CHECK_MS);
    renewIfStale();
    window.addEventListener('focus', renewIfStale);
    document.addEventListener('visibilitychange', renewIfStale);
    window.addEventListener('storage', syncSessionFromStorage);
    return () => {
      window.clearInterval(expiry);
      window.removeEventListener('focus', renewIfStale);
      document.removeEventListener('visibilitychange', renewIfStale);
      window.removeEventListener('storage', syncSessionFromStorage);
    };
  }, []);
}

export function useAuthContext(): AuthContext {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuthContext must be used within an AuthProvider');
  }
  return context;
}

export async function signOut(): Promise<void> {
  endSession();
  confirmedToken = null;
  try {
    await generatedData<{ message: string }>(Auth.authLogout());
  } catch (error) {
    console.error('Logout failed:', error);
  }
}

export async function updateAccount(formData: FormData): Promise<void> {
  const name = requiredFormValue(formData, 'name');
  const email = requiredFormValue(formData, 'email');
  await generatedData<{ message: string }>(Auth.authUpdateAccount({ body: { name, email } }));
  const session = currentSession();
  if (session) {
    saveSession(session.token, { ...session.user, name, email });
  }
}

/** A password change signs out every other session and returns this one's new token. */
export async function updatePassword(formData: FormData): Promise<void> {
  const { token } = await generatedData<{ message: string; token?: string }>(
    Auth.authUpdatePassword({
      body: {
        currentPassword: requiredFormValue(formData, 'currentPassword'),
        newPassword: requiredFormValue(formData, 'newPassword'),
      },
    }),
    { credentialCheck: true },
  );
  const session = currentSession();
  if (session && token) {
    adopt(token, session.user);
  }
}

export async function deleteAccount(formData: FormData): Promise<void> {
  await generatedData<{ message: string }>(
    Auth.authDeleteAccount({
      body: { password: requiredFormValue(formData, 'password') },
    }),
    { credentialCheck: true },
  );
  endSession();
  confirmedToken = null;
}

function requiredFormValue(formData: FormData, key: string): string {
  const value = formData.get(key);
  if (typeof value !== 'string') {
    throw new Error(`Missing ${key}`);
  }
  return value;
}

export function useAuth(): AuthContext {
  const session = useAtomValue(sessionAtom);

  return {
    user: session?.user ?? null,
    token: session?.token ?? null,
    isAuthenticated: session !== null,
    login: async (email: string, password: string) => {
      const { user, token } = await generatedData<{ user: User; token: string }>(
        Auth.authLogin({ body: { email, password } }),
        { credentialCheck: true },
      );
      if (!adopt(token, user)) {
        throw new Error('Sign-in returned an unusable token');
      }
      return user;
    },
    logout: signOut,
    signOut,
  };
}
