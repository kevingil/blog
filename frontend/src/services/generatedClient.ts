import { client } from "@/client/client.gen";
import { VITE_API_BASE_URL } from "./constants";
import {
  ApiError,
  AuthenticationError,
  type ApiErrorResponse,
} from "./authenticatedFetch";

client.setConfig({
  baseUrl: VITE_API_BASE_URL,
  auth: () =>
    typeof window === "undefined" ? undefined : localStorage.getItem("token") ?? undefined,
});

type GeneratedResult = {
  data?: unknown;
  error?: unknown;
  response: Response;
};

function errorEnvelope(error: unknown): ApiErrorResponse {
  if (error && typeof error === "object" && "error" in error) {
    return error as ApiErrorResponse;
  }

  return {
    error: typeof error === "string" && error ? error : "An error occurred",
  };
}

/**
 * Preserve the service layer's existing behavior while using the generated
 * request definitions. The Rust API keeps the Go-compatible `{ data: ... }`
 * success envelope, so the generated transport result needs one final unwrap.
 */
export async function generatedData<T>(
  request: Promise<GeneratedResult>,
): Promise<T> {
  const result = await request;

  if (result.error !== undefined) {
    const envelope = errorEnvelope(result.error);
    const code = envelope.code ?? "UNKNOWN_ERROR";
    const status = result.response?.status ?? 0;

    if (status === 401) {
      if (typeof window !== "undefined") {
        localStorage.removeItem("token");
        localStorage.removeItem("user");
        setTimeout(() => {
          window.location.href = "/login";
        }, 1500);
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
