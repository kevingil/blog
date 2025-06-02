import { atom, useAtomValue, useSetAtom } from 'jotai';
import { User } from '../types';
import { createContext, useContext, ReactNode, useEffect } from 'react';
import { VITE_API_BASE_URL } from '../constants';

const initialToken: string | null = typeof window !== 'undefined' ? localStorage.getItem('token') : null;
const storedUser = typeof window !== 'undefined' ? localStorage.getItem('user') : null;
let initialUser: User | null = null;

if (storedUser) {
  try {
    initialUser = JSON.parse(storedUser);
  } catch (error) {
    console.error('Error parsing stored user from localStorage:', error);
    if (typeof window !== 'undefined') {
      localStorage.removeItem('user');
    }
  }
}

export const tokenAtom = atom<string | null>(initialToken);
export const userAtom = atom<User | null>(initialUser);
export const isAuthenticatedAtom = atom((get) => get(tokenAtom) !== null);

// Auth Context type definition
export interface AuthContext {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<User>;
  logout: () => Promise<void>;
  signOut: () => Promise<void>;
}

// Create the Auth Context
const AuthContext = createContext<AuthContext | null>(null);

// Auth Provider Props
interface AuthProviderProps {
  children: ReactNode;
}

// Auth Provider component
export function AuthProvider({ children }: AuthProviderProps) {
  const auth = useAuth();
  
  return (
    <AuthContext.Provider value={auth}>
      {children}
    </AuthContext.Provider>
  );
}

// Hook to use the auth context
export function useAuthContext(): AuthContext {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuthContext must be used within an AuthProvider');
  }
  return context;
}

export async function performLogin(email: string, password: string): Promise<{ user: User; token: string }> {
  const response = await fetch(`${VITE_API_BASE_URL}/auth/login`, {
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

export async function performLogout(): Promise<void> {
  const token = localStorage.getItem('token');
  if (token) {
    try {
      await fetch(`${VITE_API_BASE_URL}/auth/logout`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      });
    } catch (error) {
      console.error('Logout failed:', error);
    }
  }
  localStorage.removeItem('token');
  localStorage.removeItem('user');
}

export async function signOut(): Promise<void> {
  await performLogout();
}

export async function getCurrentUser(token: string): Promise<User | null> {
  try {
    const response = await fetch(`${VITE_API_BASE_URL}/auth/me`, {
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
    const response = await fetch(`${VITE_API_BASE_URL}/auth/refresh`, {
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

  const response = await fetch(`${VITE_API_BASE_URL}/auth/account`, {
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

  const response = await fetch(`${VITE_API_BASE_URL}/auth/password`, {
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

  const response = await fetch(`${VITE_API_BASE_URL}/auth/account`, {
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
  const user = useAtomValue(userAtom);
  const token = useAtomValue(tokenAtom);
  const setUser = useSetAtom(userAtom);
  const setToken = useSetAtom(tokenAtom);
  const isAuthenticated = useAtomValue(isAuthenticatedAtom);

  // Initialize auth state from localStorage on mount
  useEffect(() => {
    const storedToken = localStorage.getItem('token');
    const storedUser = localStorage.getItem('user');
    
    if (storedToken && !token) {
      setToken(storedToken);
    }
    
    if (storedUser && !user) {
      try {
        setUser(JSON.parse(storedUser));
      } catch (error) {
        console.error('Error parsing stored user:', error);
        localStorage.removeItem('user');
      }
    }
  }, []);

  // Update localStorage when auth state changes
  useEffect(() => {
    if (user && token) {
      localStorage.setItem('user', JSON.stringify(user));
      localStorage.setItem('token', token);
    } else if (!user && !token) {
      localStorage.removeItem('user');
      localStorage.removeItem('token');
    }
  }, [user, token]);

  // // Set up token refresh interval
  // React.useEffect(() => {
  //   if (!token) return;

  //   console.log('token', token);
  //   const interval = setInterval(async () => {
  //     const newToken = await refreshToken(token);
  //     if (newToken) {
  //       if (newToken !== token) {
  //         setToken(newToken);
  //         localStorage.setItem('token', newToken);
  //       }
  //     } else {
  //       // Token refresh failed, logout
  //       setToken(null);
  //       setUser(null);
  //       localStorage.removeItem('token');
  //       localStorage.removeItem('user');
  //     }
  //   }, 5 * 60 * 1000); // Refresh every 5 minutes

  //   return () => clearInterval(interval);
  // }, [token, setToken, setUser]);

  return {
    user,
    token,
    isAuthenticated,
    login: async (email: string, password: string) => {
      const { user, token } = await performLogin(email, password);
      setUser(user);
      setToken(token);
      console.log('login token', token);
      localStorage.setItem('token', token);
      localStorage.setItem('user', JSON.stringify(user));
      return user;
    },
    logout: async () => {
      await performLogout();
      setUser(null);
      setToken(null);
    },
    signOut: async () => {
      await performLogout();
      setUser(null);
      setToken(null);
    },
  };
}
