const API_BASE = "/api/v1";

/** Error raised for every non-2xx API response or network failure. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

interface ErrorEnvelope {
  error?: { code?: string; message?: string };
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${API_BASE}${path}`, {
      ...init,
      headers: { "Content-Type": "application/json", ...init?.headers },
    });
  } catch {
    throw new ApiError(0, "network_error", "Cannot reach the server.");
  }

  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as ErrorEnvelope;
    throw new ApiError(
      response.status,
      body.error?.code ?? "unknown_error",
      body.error?.message ?? `Request failed (${response.status}).`,
    );
  }

  return (await response.json()) as T;
}
