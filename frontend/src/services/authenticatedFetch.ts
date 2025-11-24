/**
 * Authenticated API request utility
 * Handles authentication, token management, and automatic error handling across the app
 */

import { VITE_API_BASE_URL } from './constants';

export interface ApiSuccessResponse<T> {
  data: T;
  message?: string;
}

export interface ApiErrorResponse {
  error: string;
  code?: string;
  details?: Record<string, unknown>;
}

export class AuthenticationError extends Error {
  code: string;
  
  constructor(message: string, code: string) {
    super(message);
    this.name = 'AuthenticationError';
    this.code = code;
  }
}

export class ApiError extends Error {
  code: string;
  status: number;
  
  constructor(message: string, code: string, status: number) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
  }
}

interface FetchOptions extends RequestInit {
  skipAuth?: boolean;
}

/**
 * Get authentication headers
 */
function getAuthHeaders(): HeadersInit {
  const token = localStorage.getItem('token');
  if (!token) {
    return {};
  }
  return {
    'Authorization': `Bearer ${token}`,
  };
}

/**
 * Handle authentication errors by clearing tokens
 */
function handleAuthError(errorCode: string, errorMessage: string): void {
  if (errorCode === 'INVALID_TOKEN' || errorCode === 'UNAUTHORIZED') {
    // Clear expired/invalid tokens
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    
    // Redirect to login after a short delay to allow error messages to display
    setTimeout(() => {
      window.location.href = '/login';
    }, 1500);
  }
}

/**
 * Core authenticated fetch function
 * Automatically includes auth headers and handles authentication errors
 */
export async function authenticatedFetch<T>(
  url: string,
  options: FetchOptions = {}
): Promise<T> {
  const { skipAuth = false, headers = {}, ...restOptions } = options;
  
  // Build headers
  const finalHeaders: HeadersInit = {
    'Content-Type': 'application/json',
    ...headers,
    ...(skipAuth ? {} : getAuthHeaders()),
  };
  
  // Make the request
  const response = await fetch(url, {
    ...restOptions,
    headers: finalHeaders,
  });
  
  // Handle errors
  if (!response.ok) {
    let errorMessage = 'An error occurred';
    let errorCode = 'UNKNOWN_ERROR';
    
    try {
      const errorData: ApiErrorResponse = await response.json();
      errorMessage = errorData.error || errorMessage;
      errorCode = errorData.code || errorCode;
    } catch {
      errorMessage = response.statusText || errorMessage;
    }
    
    // Handle authentication errors
    if (response.status === 401) {
      handleAuthError(errorCode, errorMessage);
      throw new AuthenticationError(errorMessage, errorCode);
    }
    
    // Handle other errors
    throw new ApiError(errorMessage, errorCode, response.status);
  }
  
  // Parse and return successful response
  const jsonData = await response.json();
  
  // Check if response is wrapped in { data: ... }
  if (jsonData && typeof jsonData === 'object' && 'data' in jsonData) {
    return (jsonData as ApiSuccessResponse<T>).data;
  }
  
  // Return as-is if not wrapped
  return jsonData as T;
}

/**
 * Convenience methods for common HTTP verbs
 */

export async function apiGet<T>(endpoint: string, options?: FetchOptions): Promise<T> {
  const url = endpoint.startsWith('http') ? endpoint : `${VITE_API_BASE_URL}${endpoint}`;
  return authenticatedFetch<T>(url, {
    method: 'GET',
    ...options,
  });
}

export async function apiPost<T>(
  endpoint: string,
  body?: unknown,
  options?: FetchOptions
): Promise<T> {
  const url = endpoint.startsWith('http') ? endpoint : `${VITE_API_BASE_URL}${endpoint}`;
  return authenticatedFetch<T>(url, {
    method: 'POST',
    body: body ? JSON.stringify(body) : undefined,
    ...options,
  });
}

export async function apiPut<T>(
  endpoint: string,
  body?: unknown,
  options?: FetchOptions
): Promise<T> {
  const url = endpoint.startsWith('http') ? endpoint : `${VITE_API_BASE_URL}${endpoint}`;
  return authenticatedFetch<T>(url, {
    method: 'PUT',
    body: body ? JSON.stringify(body) : undefined,
    ...options,
  });
}

export async function apiDelete<T>(endpoint: string, options?: FetchOptions): Promise<T> {
  const url = endpoint.startsWith('http') ? endpoint : `${VITE_API_BASE_URL}${endpoint}`;
  return authenticatedFetch<T>(url, {
    method: 'DELETE',
    ...options,
  });
}

/**
 * Check if an error is an authentication error
 */
export function isAuthError(error: unknown): error is AuthenticationError {
  return error instanceof AuthenticationError;
}

/**
 * Check if an error is an API error
 */
export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}

