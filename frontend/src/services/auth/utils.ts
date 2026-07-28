/**
 * Auth utility functions for making authenticated API requests
 */

/**
 * Get authentication headers for API requests
 * Retrieves the JWT token from localStorage and formats it for the Authorization header
 * @returns Headers object with Authorization token
 * @throws Error if user is not authenticated (no token found)
 */
export function getAuthHeaders(): HeadersInit {
  const token = localStorage.getItem('token');
  if (!token) {
    throw new Error('Not authenticated');
  }
  return {
    'Authorization': `Bearer ${token}`,
  };
}

/**
 * Get authentication headers with Content-Type for JSON requests
 * @returns Headers object with Authorization token and Content-Type
 * @throws Error if user is not authenticated (no token found)
 */
export function getAuthHeadersWithContentType(): HeadersInit {
  return {
    ...getAuthHeaders(),
    'Content-Type': 'application/json',
  };
}

/**
 * Check if user is currently authenticated
 * @returns true if token exists in localStorage
 */
export function isAuthenticated(): boolean {
  return !!localStorage.getItem('token');
}

