import { api } from "@/lib/fetch-client.ts";
import type {
  Service,
  ServiceCreateInput,
  ServiceHistoryObservation,
  ServiceHistoryPage,
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

function deduplicateObservations(
  observations: ServiceHistoryObservation[],
): ServiceHistoryObservation[] {
  return Array.from(
    new Map(observations.map((observation) => [observation.id, observation])).values(),
  );
}

export const servicesApi = {
  list: (projectId: number): Promise<{ services: Service[] }> =>
    api.get(`/services?project_id=${projectId}`),

  get: (id: number): Promise<{ service: Service }> => api.get(`/services/${id}`),

  async history(id: number, params: ServiceHistoryParams = {}): Promise<ServiceHistoryPage> {
    const search = new URLSearchParams();
    if (params.limit != null) search.set("limit", String(params.limit));
    if (params.before) search.set("before", params.before);

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

  async historyWindow(id: number, since: Date): Promise<ServiceHistoryWindow> {
    const sinceMs = since.getTime();
    let before: string | undefined;
    let hasMore = true;
    let crossedCutoff = false;
    let pagesFetched = 0;
    let observations: ServiceHistoryObservation[] = [];

    // TODO(api): replace this bounded cursor walk with a server-side `since` filter when available.
    while (
      hasMore &&
      pagesFetched < MAX_HISTORY_WINDOW_PAGES &&
      observations.length < MAX_HISTORY_WINDOW_OBSERVATIONS
    ) {
      const page = await servicesApi.history(id, {
        limit: HISTORY_WINDOW_PAGE_SIZE,
        before,
      });
      pagesFetched += 1;
      observations = deduplicateObservations([...observations, ...page.observations]);
      crossedCutoff = page.observations.some(
        (observation) => new Date(observation.observedAt).getTime() < sinceMs,
      );
      hasMore = page.pagination.hasMore;
      before = page.pagination.nextCursor ?? undefined;

      if (crossedCutoff || page.observations.length === 0 || !before) break;
    }

    return {
      observations: observations.filter(
        (observation) => new Date(observation.observedAt).getTime() >= sinceMs,
      ),
      isTruncated: hasMore && !crossedCutoff,
    };
  },

  create: (payload: ServiceCreateInput): Promise<{ service: Service }> =>
    api.post("/services", payload),

  update: (id: number, payload: ServiceUpdateInput): Promise<{ service: Service }> =>
    api.patch(`/services/${id}`, payload),

  delete: (id: number): Promise<void> => api.del(`/services/${id}`),
};
