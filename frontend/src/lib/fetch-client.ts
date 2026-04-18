const BASE_URL = import.meta.env["VITE_API_URL"] ?? "";

export function getApiBaseUrl() {
  if (typeof window === "undefined") {
    return BASE_URL || "";
  }

  if (!BASE_URL) {
    return window.location.origin;
  }

  return new URL(BASE_URL, window.location.origin).toString().replace(/\/$/, "");
}

export function getAgentApiBaseUrl() {
  const baseUrl = getApiBaseUrl();

  if (!baseUrl) {
    return "/v1";
  }

  return baseUrl.endsWith("/v1") ? baseUrl : `${baseUrl}/v1`;
}

export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

export class NetworkError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "NetworkError";
  }
}

function getServerErrorMessage(status: number, fallback: string) {
  if (status === 502 || status === 503 || status === 504) {
    return "Agrafa can't reach the server right now. Try again in a moment.";
  }

  return fallback;
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const baseUrl = getApiBaseUrl();
  const url = `${baseUrl}/v1${path}`;

  let res: Response;
  try {
    res = await fetch(url, {
      ...init,
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        ...init.headers,
      },
    });
  } catch {
    throw new NetworkError(
      "Agrafa can't reach the server right now. Check your connection and try again.",
    );
  }

  if (!res.ok) {
    let message = res.statusText;
    try {
      const body = (await res.json()) as { error?: string };
      if (body.error) message = body.error;
    } catch {
      // ignore parse error
    }
    throw new ApiError(res.status, getServerErrorMessage(res.status, message));
  }

  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

export const api = {
  get: <T>(path: string) => request<T>(path, { method: "GET" }),

  post: <T>(path: string, body?: unknown) =>
    request<T>(path, {
      method: "POST",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }),

  patch: <T>(path: string, body?: unknown) =>
    request<T>(path, {
      method: "PATCH",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }),

  del: <T = void>(path: string) => request<T>(path, { method: "DELETE" }),
};
