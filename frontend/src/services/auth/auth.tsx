import { atom, useAtom } from 'jotai';
import { User } from '../types';
import React from 'react';

// Auth state atoms
export const userAtom = atom<User | null>(null);
export const tokenAtom = atom<string | null>(null);
export const isAuthenticatedAtom = atom((get) => get(userAtom) !== null && get(tokenAtom) !== null);

// Auth service functions
const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

export async function login(email: string, password: string): Promise<{ user: User; token: string }> {
  const response = await fetch(`${API_BASE_URL}/auth/login`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email, password }),
  });

  if (!response.ok) {
    throw new Error('Login failed');
  }

  const data = await response.json();
  return data;
}

export async function signOut(): Promise<void> {
  // Clear token from storage
  localStorage.removeItem('token');
}

export async function getCurrentUser(token: string): Promise<User | null> {
  try {
    const response = await fetch(`${API_BASE_URL}/auth/me`, {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
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

export async function refreshToken(token: string): Promise<string | null> {
  try {
    const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      return null;
    }

    const data = await response.json();
    return data.token;
  } catch {
    return null;
  }
}

export async function updateAccount(formData: FormData): Promise<void> {
  const token = localStorage.getItem('token');
  if (!token) {
    throw new Error('Not authenticated');
  }

  const response = await fetch(`${API_BASE_URL}/auth/account`, {
    method: 'PUT',
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    body: formData,
  });

  if (!response.ok) {
    throw new Error('Failed to update account');
  }
}

export async function updatePassword(formData: FormData): Promise<void> {
  const token = localStorage.getItem('token');
  if (!token) {
    throw new Error('Not authenticated');
  }

  const response = await fetch(`${API_BASE_URL}/auth/password`, {
    method: 'PUT',
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    body: formData,
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to update password');
  }
}

export async function deleteAccount(formData: FormData): Promise<void> {
  const token = localStorage.getItem('token');
  if (!token) {
    throw new Error('Not authenticated');
  }

  const response = await fetch(`${API_BASE_URL}/auth/account`, {
    method: 'DELETE',
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    body: formData,
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to delete account');
  }
}

// Custom hook for auth state
export function useAuth() {
  const [user, setUser] = useAtom(userAtom);
  const [token, setToken] = useAtom(tokenAtom);
  const [isAuthenticated] = useAtom(isAuthenticatedAtom);

  // Initialize token from localStorage on mount
  React.useEffect(() => {
    const storedToken = localStorage.getItem('token');
    if (storedToken) {
      setToken(storedToken);
      getCurrentUser(storedToken).then(setUser);
    }
  }, [setToken, setUser]);

  // Set up token refresh interval
  React.useEffect(() => {
    if (!token) return;

    const interval = setInterval(async () => {
      const newToken = await refreshToken(token);
      if (newToken) {
        setToken(newToken);
        localStorage.setItem('token', newToken);
      } else {
        // Token refresh failed, logout
        setToken(null);
        setUser(null);
        localStorage.removeItem('token');
      }
    }, 5 * 60 * 1000); // Refresh every 5 minutes

    return () => clearInterval(interval);
  }, [token, setToken, setUser]);

  return {
    user,
    token,
    isAuthenticated,
    setUser,
    setToken,
    login: async (email: string, password: string) => {
      const { user, token } = await login(email, password);
      setUser(user);
      setToken(token);
      localStorage.setItem('token', token);
      return user;
    },
    signOut: async () => {
      await signOut();
      setUser(null);
      setToken(null);
      localStorage.removeItem('token');
    },
  };
} 
