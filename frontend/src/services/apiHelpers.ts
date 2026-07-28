// API response helpers for handling the new standardized backend response format

export interface ApiSuccessResponse<T> {
  data: T;
  message?: string;
}

export interface ApiErrorResponse {
  error: string;
  code?: string;
  details?: Record<string, unknown>;
}

/**
 * Handles API responses with the new standardized format
 * Success responses are wrapped in { data: ... }
 * Error responses have { error, code, details }
 */
export async function handleApiResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    let errorMessage = 'An error occurred';
    let errorCode = 'UNKNOWN_ERROR';
    
    try {
      const errorData: ApiErrorResponse = await response.json();
      errorMessage = errorData.error || errorMessage;
      errorCode = errorData.code || errorCode;
    } catch {
      // If JSON parsing fails, use status text
      errorMessage = response.statusText || errorMessage;
    }
    
    const error = new Error(errorMessage) as Error & { code?: string };
    error.code = errorCode;
    throw error;
  }

  const jsonData = await response.json();
  
  // Check if response is wrapped in { data: ... }
  if (jsonData && typeof jsonData === 'object' && 'data' in jsonData) {
    return (jsonData as ApiSuccessResponse<T>).data;
  }
  
  // Return as-is if not wrapped (for backwards compatibility)
  return jsonData as T;
}

/**
 * Gets the authorization header with the stored token
 */
export function getAuthHeaders(): HeadersInit {
  const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null;
  return {
    ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
  };
}

/**
 * Gets the authorization header with Content-Type
 */
export function getAuthHeadersWithContentType(): HeadersInit {
  return {
    ...getAuthHeaders(),
    'Content-Type': 'application/json',
  };
}

