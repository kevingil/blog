import { User } from '../types';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

export interface SessionData {
  user: User;
  expires: string;
}

export async function getSession(): Promise<SessionData | null> {
  try {
    const response = await fetch(`${API_BASE_URL}/auth/session`, {
      credentials: 'include',
    });

    if (!response.ok) {
      return null;
    }

    return await response.json();
  } catch {
    return null;
  }
}

export async function refreshSession(): Promise<SessionData | null> {
  try {
    const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
      method: 'POST',
      credentials: 'include',
    });

    if (!response.ok) {
      return null;
    }

    return await response.json();
  } catch {
    return null;
  }
}
