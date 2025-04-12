import { atom, useAtom } from 'jotai';
import { User } from '../types';

// Auth state atoms
export const userAtom = atom<User | null>(null);
export const isAuthenticatedAtom = atom((get) => get(userAtom) !== null);

// Auth service functions
const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

export async function login(email: string, password: string): Promise<User> {
  const response = await fetch(`${API_BASE_URL}/auth/login`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email, password }),
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error('Login failed');
  }

  const data = await response.json();
  return data.user;
}

export async function logout(): Promise<void> {
  await fetch(`${API_BASE_URL}/auth/logout`, {
    method: 'POST',
    credentials: 'include',
  });
}

export async function getCurrentUser(): Promise<User | null> {
  try {
    const response = await fetch(`${API_BASE_URL}/auth/me`, {
      credentials: 'include',
    });

    if (!response.ok) {
      return null;
    }

    const data = await response.json();
    return data.user;
  } catch {
    return null;
  }
}

// Custom hook for auth state
export function useAuth() {
  const [user, setUser] = useAtom(userAtom);
  const [isAuthenticated] = useAtom(isAuthenticatedAtom);

  return {
    user,
    isAuthenticated,
    setUser,
    login: async (email: string, password: string) => {
      const user = await login(email, password);
      setUser(user);
      return user;
    },
    logout: async () => {
      await logout();
      setUser(null);
    },
  };
} 
