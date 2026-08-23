import { api } from "@/lib/fetch-client.ts";
import type {
  CheckType,
  Service,
  ServiceCreateInput,
  ServiceHistoryObservation,
  ServiceHistoryPage,
  ServiceHistoryWindow,
  ServiceUpdateInput,
} from "@/types/service.ts";

type ServiceHistoryEntryResponse = {
  id: number;
  service_id: number;
  node_id: number;
  check_type: string;
  source: string;
  observed_at: string;
  is_success: boolean;
  status_code: number | null;
  response_time_ms: number | null;
  message: string | null;
  metadata: Record<string, unknown> | null;
};

type ServiceHistoryResponse = {
  history: ServiceHistoryEntryResponse[];
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

function toHistoryObservation(entry: ServiceHistoryEntryResponse): ServiceHistoryObservation {
  return {
    id: entry.id,
    serviceId: entry.service_id,
    nodeId: entry.node_id,
    checkType: entry.check_type as CheckType,
    source: entry.source,
    observedAt: entry.observed_at,
    isSuccess: entry.is_success,
    statusCode: entry.status_code,
    latencyMs: entry.response_time_ms,
    message: entry.message,
    metadata: entry.metadata ?? {},
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
      observations: response.history.map(toHistoryObservation),
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
