import { atom, useAtomValue, useSetAtom } from 'jotai';
import { User } from '../types';
import { createContext, useContext, ReactNode, useEffect } from 'react';
import { Auth } from '@/client';
import { generatedData } from '../generatedClient';

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
  return generatedData<{ user: User; token: string }>(
    Auth.authLogin({ body: { email, password } }),
  );
}

export async function performLogout(): Promise<void> {
  const token = localStorage.getItem('token');
  if (token) {
    try {
      await generatedData<{ message: string }>(Auth.authLogout());
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

export async function updateAccount(formData: FormData): Promise<void> {
  await generatedData<{ message: string }>(
    Auth.authUpdateAccount({
      body: {
        name: requiredFormValue(formData, 'name'),
        email: requiredFormValue(formData, 'email'),
      },
    }),
  );
}

export async function updatePassword(formData: FormData): Promise<void> {
  await generatedData<{ message: string }>(
    Auth.authUpdatePassword({
      body: {
        currentPassword: requiredFormValue(formData, 'currentPassword'),
        newPassword: requiredFormValue(formData, 'newPassword'),
      },
    }),
  );
}

export async function deleteAccount(formData: FormData): Promise<void> {
  await generatedData<{ message: string }>(
    Auth.authDeleteAccount({
      body: { password: requiredFormValue(formData, 'password') },
    }),
  );
}

function requiredFormValue(formData: FormData, key: string): string {
  const value = formData.get(key);
  if (typeof value !== 'string') {
    throw new Error(`Missing ${key}`);
  }
  return value;
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
