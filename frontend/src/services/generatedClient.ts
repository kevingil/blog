import { client } from "@/client/client.gen";
import { VITE_API_BASE_URL } from "./constants";
import {
  ApiError,
  AuthenticationError,
  type ApiErrorResponse,
} from "./authenticatedFetch";
import { currentSession, endSession } from "./auth/session";

client.setConfig({
  baseUrl: VITE_API_BASE_URL,
  auth: () => currentSession()?.token,
});

type GeneratedResult = {
  data?: unknown;
  error?: unknown;
  request?: Request;
  response: Response;
};

export type RequestOptions = {
  /**
   * The endpoint checks a password the user typed and answers 401 when it is
   * wrong. That 401 says nothing about the session, so it must not end it.
   */
  credentialCheck?: boolean;
};

function errorEnvelope(error: unknown): ApiErrorResponse {
  if (error && typeof error === "object" && "error" in error) {
    return error as ApiErrorResponse;
  }

  return {
    error: typeof error === "string" && error ? error : "An error occurred",
  };
}

/** Ends the session only if the server rejected the token it still holds. */
function endRejectedSession(request: Request | undefined) {
  const session = currentSession();
  if (session && request?.headers.get("Authorization") === `Bearer ${session.token}`) {
    endSession();
  }
}

/**
 * Preserve the service layer's existing behavior while using the generated
 * request definitions. The Rust API keeps the Go-compatible `{ data: ... }`
 * success envelope, so the generated transport result needs one final unwrap.
 */
export async function generatedData<T>(
  request: Promise<GeneratedResult>,
  options: RequestOptions = {},
): Promise<T> {
  const result = await request;

  if (result.error !== undefined) {
    const envelope = errorEnvelope(result.error);
    const code = envelope.code ?? "UNKNOWN_ERROR";
    const status = result.response?.status ?? 0;

    if (status === 401) {
      if (!options.credentialCheck) {
        endRejectedSession(result.request);
      }
      throw new AuthenticationError(envelope.error, code);
    }

    throw new ApiError(envelope.error, code, status);
  }

  const payload = result.data;
  if (payload && typeof payload === "object" && "data" in payload) {
    return (payload as { data: T }).data;
  }

  return payload as T;
}
