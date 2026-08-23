import { api } from "@/lib/fetch-client.ts";
import type {
  Service,
  ServiceCreateInput,
  ServiceHistoryObservation,
  ServiceHistoryPage,
  ServiceHistorySummary,
  ServiceHistoryWindow,
  ServiceUpdateInput,
} from "@/types/service.ts";

type ServiceHistoryResponse = {
  history: unknown[];
  pagination: {
    limit: number;
    has_more: boolean;
    next_cursor: string | null;
  };
};

export type ServiceHistoryParams = {
  limit?: number;
  before?: string;
  from?: Date;
  to?: Date;
};

const HISTORY_WINDOW_PAGE_SIZE = 500;
const MAX_HISTORY_WINDOW_OBSERVATIONS = 2_000;
const MAX_HISTORY_WINDOW_PAGES = MAX_HISTORY_WINDOW_OBSERVATIONS / HISTORY_WINDOW_PAGE_SIZE;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value != null && !Array.isArray(value);
}

export function toHistoryObservation(entry: unknown): ServiceHistoryObservation | null {
  if (!isRecord(entry)) return null;

  const rawCheckType = entry["check_type"];
  const checkType = typeof rawCheckType === "string" ? rawCheckType.trim().toLowerCase() : null;
  const observedAt = entry["observed_at"];
  const statusCode = entry["status_code"];
  const responseTimeMs = entry["response_time_ms"];
  const message = entry["message"];
  if (
    typeof entry["id"] !== "number" ||
    !Number.isInteger(entry["id"]) ||
    entry["id"] <= 0 ||
    typeof entry["service_id"] !== "number" ||
    typeof entry["node_id"] !== "number" ||
    (checkType !== "http" && checkType !== "tcp") ||
    typeof entry["source"] !== "string" ||
    typeof observedAt !== "string" ||
    !Number.isFinite(Date.parse(observedAt)) ||
    typeof entry["is_success"] !== "boolean" ||
    (statusCode !== null && typeof statusCode !== "number") ||
    (responseTimeMs !== null && typeof responseTimeMs !== "number") ||
    (message !== null && typeof message !== "string")
  ) {
    return null;
  }

  const metadata = entry["metadata"];
  return {
    id: entry["id"],
    serviceId: entry["service_id"],
    nodeId: entry["node_id"],
    checkType,
    source: entry["source"],
    observedAt,
    isSuccess: entry["is_success"],
    statusCode,
    latencyMs: responseTimeMs,
    message,
    metadata: isRecord(metadata) ? metadata : {},
  };
}

/**
 * Removes duplicate history observations by ID, retaining the last occurrence of each ID.
 *
 * @param observations - The history observations to deduplicate
 * @returns The observations with duplicate IDs removed
 */
function deduplicateObservations(
  observations: ServiceHistoryObservation[],
): ServiceHistoryObservation[] {
  return Array.from(
    new Map(observations.map((observation) => [observation.id, observation])).values(),
  );
}

/**
 * Fetches service history observations within a date range subject to pagination and observation limits.
 *
 * @param id - The service ID
 * @param from - The start of the observation date range
 * @param to - The end of the observation date range
 * @returns The observations in the range and whether additional history pages remain
 */
async function fetchBoundedHistoryObservations(
  id: number,
  from: Date,
  to: Date,
): Promise<Pick<ServiceHistoryWindow, "observations" | "isTruncated">> {
  const fromMs = from.getTime();
  const toMs = to.getTime();
  let before: string | undefined;
  let hasMore = true;
  let pagesFetched = 0;
  let observations: ServiceHistoryObservation[] = [];

  while (
    hasMore &&
    pagesFetched < MAX_HISTORY_WINDOW_PAGES &&
    observations.length < MAX_HISTORY_WINDOW_OBSERVATIONS
  ) {
    const page = await servicesApi.history(id, {
      limit: HISTORY_WINDOW_PAGE_SIZE,
      before,
      from,
      to,
    });
    pagesFetched += 1;
    observations = deduplicateObservations([...observations, ...page.observations]);
    hasMore = page.pagination.hasMore;
    before = page.pagination.nextCursor ?? undefined;
    if (page.observations.length === 0 || !before) break;
  }

  return {
    observations: observations.filter((observation) => {
      const observedAt = Date.parse(observation.observedAt);
      return observedAt >= fromMs && observedAt <= toMs;
    }),
    isTruncated: hasMore,
  };
}

export const servicesApi = {
  list: (projectId: number): Promise<{ services: Service[] }> =>
    api.get(`/services?project_id=${projectId}`),

  get: (id: number): Promise<{ service: Service }> => api.get(`/services/${id}`),

  async history(id: number, params: ServiceHistoryParams = {}): Promise<ServiceHistoryPage> {
    const search = new URLSearchParams();
    if (params.limit != null) search.set("limit", String(params.limit));
    if (params.before) search.set("before", params.before);
    if (params.from) search.set("from", params.from.toISOString());
    if (params.to) search.set("to", params.to.toISOString());

    const query = search.size > 0 ? `?${search.toString()}` : "";
    const response = await api.get<ServiceHistoryResponse>(`/services/${id}/history${query}`);

    return {
      observations: response.history
        .map(toHistoryObservation)
        .filter((observation): observation is ServiceHistoryObservation => observation != null),
      pagination: {
        limit: response.pagination.limit,
        hasMore: response.pagination.has_more,
        nextCursor: response.pagination.next_cursor,
      },
    };
  },

  async historySummary(id: number, from: Date, to: Date): Promise<ServiceHistorySummary> {
    const search = new URLSearchParams({ from: from.toISOString(), to: to.toISOString() });
    const response = await api.get<Record<string, unknown>>(
      `/services/${id}/history/summary?${search.toString()}`,
    );
    const totalChecks = response["total_checks"];
    const successfulChecks = response["successful_checks"];
    const uptimePercent = response["uptime_percent"];
    const averageLatencyMs = response["average_latency_ms"];
    const lastCheckedAt = response["last_checked_at"];
    if (
      typeof response["from"] !== "string" ||
      !Number.isFinite(Date.parse(response["from"])) ||
      typeof response["to"] !== "string" ||
      !Number.isFinite(Date.parse(response["to"])) ||
      typeof totalChecks !== "number" ||
      !Number.isInteger(totalChecks) ||
      totalChecks < 0 ||
      typeof successfulChecks !== "number" ||
      !Number.isInteger(successfulChecks) ||
      successfulChecks < 0 ||
      successfulChecks > totalChecks ||
      (uptimePercent !== null &&
        (typeof uptimePercent !== "number" || !Number.isFinite(uptimePercent))) ||
      (averageLatencyMs !== null &&
        (typeof averageLatencyMs !== "number" || !Number.isFinite(averageLatencyMs))) ||
      (lastCheckedAt !== null &&
        (typeof lastCheckedAt !== "string" || !Number.isFinite(Date.parse(lastCheckedAt))))
    ) {
      throw new Error("Invalid service history summary response");
    }
    return {
      from: response["from"],
      to: response["to"],
      totalChecks,
      successfulChecks,
      uptimePercent,
      averageLatencyMs,
      lastCheckedAt,
    };
  },

  async historyWindow(id: number, since: Date, to = new Date()): Promise<ServiceHistoryWindow> {
    const [boundedHistory, summary] = await Promise.all([
      fetchBoundedHistoryObservations(id, since, to),
      servicesApi.historySummary(id, since, to),
    ]);
    const effectiveFromMs = Date.parse(summary.from);
    const effectiveToMs = Date.parse(summary.to);

    return {
      ...boundedHistory,
      observations: boundedHistory.observations.filter((observation) => {
        const observedAt = Date.parse(observation.observedAt);
        return observedAt >= effectiveFromMs && observedAt <= effectiveToMs;
      }),
      from: summary.from,
      to: summary.to,
      summary,
    };
  },

  create: (payload: ServiceCreateInput): Promise<{ service: Service }> =>
    api.post("/services", payload),

  update: (id: number, payload: ServiceUpdateInput): Promise<{ service: Service }> =>
    api.patch(`/services/${id}`, payload),

  delete: (id: number): Promise<void> => api.del(`/services/${id}`),
};
