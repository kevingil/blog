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

export function isAuthError(error: unknown): error is AuthenticationError {
  return error instanceof AuthenticationError;
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}
