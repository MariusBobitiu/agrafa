import {
  type InfiniteData,
  keepPreviousData,
  type QueryClient,
  type QueryState,
  queryOptions,
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { servicesApi, toHistoryObservation } from "@/data/services.ts";
import { useSSE } from "@/hooks/use-sse.ts";
import type {
  Service,
  ServiceCreateInput,
  ServiceHistoryObservation,
  ServiceHistoryPage,
  ServiceHistoryWindow,
  ServiceUpdateInput,
} from "@/types/service.ts";

export const serviceHistoryKeys = {
  all: ["services", "history"] as const,
  service: (id: number) => [...serviceHistoryKeys.all, id] as const,
  lists: (id: number) => [...serviceHistoryKeys.service(id), "list"] as const,
  list: (id: number, limit: number) => [...serviceHistoryKeys.lists(id), limit] as const,
  window: (id: number, rangeHours: number) =>
    [...serviceHistoryKeys.service(id), "window", rangeHours] as const,
};

export const SERVICE_HISTORY_RANGE_REFETCH_INTERVAL_MS = 60_000;

export function serviceHistoryRangeRefetchInterval(
  state: Pick<QueryState<unknown, Error>, "dataUpdatedAt" | "errorUpdatedAt" | "fetchStatus">,
) {
  if (state.fetchStatus === "fetching") return SERVICE_HISTORY_RANGE_REFETCH_INTERVAL_MS;

  const lastAuthoritativeUpdate = Math.max(state.dataUpdatedAt, state.errorUpdatedAt);
  const elapsed = Date.now() - lastAuthoritativeUpdate;
  return Math.max(1, SERVICE_HISTORY_RANGE_REFETCH_INTERVAL_MS - elapsed);
}

function newestObservationFirst(left: ServiceHistoryObservation, right: ServiceHistoryObservation) {
  const timestampDifference = Date.parse(right.observedAt) - Date.parse(left.observedAt);
  return timestampDifference || right.id - left.id;
}

/**
 * Updates active service history caches with a valid observation that falls within each cached time window.
 *
 * @param id - The service identifier associated with the observation
 * @param observation - The observation to add or update in the caches
 */
export function updateServiceHistoryCaches(
  queryClient: QueryClient,
  id: number,
  observation: ServiceHistoryObservation,
) {
  if (observation.serviceId !== id || !Number.isFinite(Date.parse(observation.observedAt))) return;

  const activeListQueries = queryClient
    .getQueryCache()
    .findAll({ queryKey: serviceHistoryKeys.lists(id) })
    .filter((query) => query.getObserversCount() > 0 && query.state.data != null);

  for (const query of activeListQueries) {
    queryClient.setQueryData<InfiniteData<ServiceHistoryPage, string | null>>(
      query.queryKey,
      (current) => {
        const firstPage = current?.pages[0];
        if (!current || !firstPage) return current;

        const observations = [
          observation,
          ...firstPage.observations.filter((item) => item.id !== observation.id),
        ].sort(newestObservationFirst);

        return {
          ...current,
          pages: [{ ...firstPage, observations }, ...current.pages.slice(1)],
        };
      },
      { updatedAt: query.state.dataUpdatedAt },
    );
  }

  const rangeQueries = queryClient
    .getQueryCache()
    .findAll({ queryKey: [...serviceHistoryKeys.service(id), "window"] })
    .filter((query) => query.state.data != null);

  for (const query of rangeQueries) {
    queryClient.setQueryData<ServiceHistoryWindow>(
      query.queryKey,
      (current) => {
        if (!current) return current;
        const observedAt = Date.parse(observation.observedAt);
        if (observedAt < Date.parse(current.from) || observedAt > Date.parse(current.to)) {
          return current;
        }

        const observations = [
          observation,
          ...current.observations.filter((item) => item.id !== observation.id),
        ].sort(newestObservationFirst);

        return { ...current, observations };
      },
      { updatedAt: query.state.dataUpdatedAt },
    );
  }
}

export function useServices(
  projectId: number,
  options?: {
    enabled?: boolean;
    refetchInterval?: number | false;
  },
) {
  return useQuery({
    queryKey: ["services", projectId],
    queryFn: () => servicesApi.list(projectId),
    enabled: (options?.enabled ?? true) && projectId > 0,
    refetchInterval: options?.refetchInterval ?? 10_000,
  });
}

export function useService(
  id: number,
  options?: {
    enabled?: boolean;
    refetchInterval?: number | false;
  },
) {
  return useQuery({
    queryKey: ["services", "detail", id],
    queryFn: () => servicesApi.get(id),
    enabled: (options?.enabled ?? true) && id > 0,
    refetchInterval: options?.refetchInterval ?? false,
  });
}

export type ServiceDetailStreamPayload = {
  service: Service;
  observation?: unknown;
};

/**
 * Applies a service detail stream payload to the service detail and history caches.
 *
 * @param queryClient - The query client whose caches are updated
 * @param id - The service identifier
 * @param payload - The streamed service detail payload
 */
export async function applyServiceDetailStreamPayload(
  queryClient: QueryClient,
  id: number,
  payload: ServiceDetailStreamPayload,
) {
  queryClient.setQueryData(["services", "detail", id], { service: payload.service });

  const observation = toHistoryObservation(payload.observation);
  if (!observation || observation.serviceId !== id) return;

  const affectedQueries = queryClient
    .getQueryCache()
    .findAll({ queryKey: serviceHistoryKeys.service(id) })
    .filter((query) => {
      if (query.state.data == null) return false;
      if (query.queryKey[3] === "list") return query.getObserversCount() > 0;
      if (query.queryKey[3] !== "window") return false;

      const current = query.state.data as ServiceHistoryWindow;
      const observedAt = Date.parse(observation.observedAt);
      return observedAt >= Date.parse(current.from) && observedAt <= Date.parse(current.to);
    });

  await Promise.all(
    affectedQueries.map((query) =>
      queryClient.cancelQueries({ queryKey: query.queryKey, exact: true }, { silent: true }),
    ),
  );
  updateServiceHistoryCaches(queryClient, id, observation);
}

export function useServiceDetailStream(id: number, options?: { enabled?: boolean }) {
  const qc = useQueryClient();

  useSSE<ServiceDetailStreamPayload>({
    enabled: (options?.enabled ?? true) && id > 0,
    path: `/services/${id}/stream`,
    onMessage: (payload) => void applyServiceDetailStreamPayload(qc, id, payload),
  });
}

export function useServiceHistory(id: number, limit = 20) {
  return useInfiniteQuery({
    queryKey: serviceHistoryKeys.list(id, limit),
    queryFn: ({ pageParam }) => servicesApi.history(id, { limit, before: pageParam ?? undefined }),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => lastPage.pagination.nextCursor ?? undefined,
    enabled: id > 0,
  });
}

export function useServiceHistoryWindow(id: number, rangeHours: number) {
  return useQuery(serviceHistoryWindowQueryOptions(id, rangeHours));
}

/**
 * Configures a query for service observations within a rolling time window.
 *
 * @param rangeHours - The number of hours included in the window
 * @returns Query options for fetching the service history window
 */
export function serviceHistoryWindowQueryOptions(id: number, rangeHours: number) {
  return queryOptions({
    queryKey: serviceHistoryKeys.window(id, rangeHours),
    queryFn: () => {
      const to = new Date();
      const from = new Date(to.getTime() - rangeHours * 60 * 60 * 1_000);
      return servicesApi.historyWindow(id, from, to);
    },
    enabled: id > 0,
    refetchInterval: (query) => serviceHistoryRangeRefetchInterval(query.state),
    placeholderData: keepPreviousData,
  });
}

export function useCreateService(projectId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: ServiceCreateInput) => servicesApi.create(payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["services", projectId] });
      void qc.invalidateQueries({ queryKey: ["overview", projectId] });
    },
  });
}

export function useUpdateService(projectId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: ServiceUpdateInput }) =>
      servicesApi.update(id, payload),
    onSuccess: (_, variables) => {
      void qc.invalidateQueries({ queryKey: ["services", projectId] });
      void qc.invalidateQueries({ queryKey: ["services", "detail", variables.id] });
      void qc.invalidateQueries({ queryKey: ["overview", projectId] });
    },
  });
}

export function useDeleteService(projectId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => servicesApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["services", projectId] }),
  });
}
