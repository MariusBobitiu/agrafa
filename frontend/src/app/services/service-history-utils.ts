import type { ServiceHistoryObservation } from "@/types/service.ts";

export const SERVICE_HISTORY_RANGES = [
  { label: "1H", hours: 1 },
  { label: "6H", hours: 6 },
  { label: "24H", hours: 24 },
  { label: "7D", hours: 24 * 7 },
] as const;

export type ServiceHistoryRangeHours = (typeof SERVICE_HISTORY_RANGES)[number]["hours"];

export type ServiceHistoryMetrics = {
  uptimePercent: number | null;
  averageLatencyMs: number | null;
  lastCheckedAt: string | null;
};

export type HistoryRowPresentation = {
  result: string;
  latency: string;
};

const HTTP_STATUS_TEXT: Record<number, string> = {
  200: "OK",
  201: "Created",
  202: "Accepted",
  204: "No Content",
  301: "Moved Permanently",
  302: "Found",
  304: "Not Modified",
  400: "Bad Request",
  401: "Unauthorized",
  403: "Forbidden",
  404: "Not Found",
  408: "Request Timeout",
  429: "Too Many Requests",
  500: "Internal Server Error",
  502: "Bad Gateway",
  503: "Service Unavailable",
  504: "Gateway Timeout",
};

export function deduplicateHistory(
  observations: ServiceHistoryObservation[],
): ServiceHistoryObservation[] {
  return Array.from(new Map(observations.map((item) => [item.id, item])).values());
}

export function calculateHistoryMetrics(
  observations: ServiceHistoryObservation[],
): ServiceHistoryMetrics {
  if (observations.length === 0) {
    return { uptimePercent: null, averageLatencyMs: null, lastCheckedAt: null };
  }

  const successful = observations.filter((observation) => observation.isSuccess);
  const measuredSuccessful = successful.filter(
    (observation): observation is ServiceHistoryObservation & { latencyMs: number } =>
      observation.latencyMs != null,
  );
  const latencyTotal = measuredSuccessful.reduce(
    (total, observation) => total + observation.latencyMs,
    0,
  );
  const latest = observations.reduce((current, observation) =>
    Date.parse(observation.observedAt) > Date.parse(current.observedAt) ? observation : current,
  );

  return {
    uptimePercent: (successful.length / observations.length) * 100,
    averageLatencyMs:
      measuredSuccessful.length > 0 ? latencyTotal / measuredSuccessful.length : null,
    lastCheckedAt: latest.observedAt,
  };
}

export function formatUptime(value: number | null): string {
  return value == null ? "—" : `${value.toFixed(2)}%`;
}

export function formatLatency(value: number | null): string {
  return value == null ? "—" : `${Math.round(value)} ms`;
}

function conciseMessage(message: string | null): string | null {
  const trimmed = message?.trim();
  if (!trimmed) return null;
  return trimmed.length > 140 ? `${trimmed.slice(0, 137)}…` : trimmed;
}

function normalizedFailureMessage(observation: ServiceHistoryObservation): string {
  const message = conciseMessage(observation.message);
  const normalized = message?.toLowerCase() ?? "";

  if (normalized.includes("no such host") || normalized.includes("name resolution")) {
    return "Host not found";
  }
  if (normalized.includes("connection refused") || normalized.includes("connect: refused")) {
    return "Connection refused";
  }
  if (
    normalized.includes("timeout") ||
    normalized.includes("timed out") ||
    normalized.includes("deadline exceeded")
  ) {
    return observation.checkType === "tcp" ? "Connection timed out" : "Request timed out";
  }

  return message ?? (observation.checkType === "tcp" ? "Connection failed" : "Request failed");
}

export function getHistoryRowPresentation(
  observation: ServiceHistoryObservation,
): HistoryRowPresentation {
  if (observation.checkType === "tcp") {
    return {
      result: observation.isSuccess ? "Connected" : normalizedFailureMessage(observation),
      latency: formatLatency(observation.latencyMs),
    };
  }

  if (observation.statusCode != null) {
    const statusText = HTTP_STATUS_TEXT[observation.statusCode];
    return {
      result: statusText
        ? `${observation.statusCode} ${statusText}`
        : String(observation.statusCode),
      latency: formatLatency(observation.latencyMs),
    };
  }

  return {
    result: observation.isSuccess
      ? (conciseMessage(observation.message) ?? "Request succeeded")
      : normalizedFailureMessage(observation),
    latency: formatLatency(observation.latencyMs),
  };
}
