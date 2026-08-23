import {
  type InfiniteData,
  keepPreviousData,
  type QueryClient,
  queryOptions,
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import {
  MAX_HISTORY_WINDOW_OBSERVATIONS,
  servicesApi,
  toHistoryObservation,
} from "@/data/services.ts";
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

function newestObservationFirst(left: ServiceHistoryObservation, right: ServiceHistoryObservation) {
  const timestampDifference = Date.parse(right.observedAt) - Date.parse(left.observedAt);
  return timestampDifference || right.id - left.id;
}

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
    const limit = query.queryKey.at(-1);
    if (typeof limit !== "number" || limit <= 0) continue;

    queryClient.setQueryData<InfiniteData<ServiceHistoryPage, string | null>>(
      query.queryKey,
      (current) => {
        const firstPage = current?.pages[0];
        if (!current || !firstPage) return current;

        const observations = [
          observation,
          ...firstPage.observations.filter((item) => item.id !== observation.id),
        ]
          .sort(newestObservationFirst)
          .slice(0, limit);

        return {
          ...current,
          pages: [{ ...firstPage, observations }, ...current.pages.slice(1)],
        };
      },
    );
  }

  const rangeQueries = queryClient
    .getQueryCache()
    .findAll({ queryKey: [...serviceHistoryKeys.service(id), "window"] })
    .filter((query) => query.state.data != null);

  for (const query of rangeQueries) {
    const rangeHours = query.queryKey.at(-1);
    if (typeof rangeHours !== "number" || rangeHours <= 0) continue;
    if (Date.parse(observation.observedAt) < Date.now() - rangeHours * 60 * 60 * 1_000) continue;

    queryClient.setQueryData<ServiceHistoryWindow>(query.queryKey, (current) => {
      if (!current) return current;

      const observations = [
        observation,
        ...current.observations.filter((item) => item.id !== observation.id),
      ]
        .sort(newestObservationFirst)
        .slice(0, MAX_HISTORY_WINDOW_OBSERVATIONS);

      return { ...current, observations };
    });
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

export function applyServiceDetailStreamPayload(
  queryClient: QueryClient,
  id: number,
  payload: ServiceDetailStreamPayload,
) {
  queryClient.setQueryData(["services", "detail", id], { service: payload.service });

  const observation = toHistoryObservation(payload.observation);
  if (observation) updateServiceHistoryCaches(queryClient, id, observation);
}

export function useServiceDetailStream(id: number, options?: { enabled?: boolean }) {
  const qc = useQueryClient();

  useSSE<ServiceDetailStreamPayload>({
    enabled: (options?.enabled ?? true) && id > 0,
    path: `/services/${id}/stream`,
    onMessage: (payload) => applyServiceDetailStreamPayload(qc, id, payload),
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

export function serviceHistoryWindowQueryOptions(id: number, rangeHours: number) {
  return queryOptions({
    queryKey: serviceHistoryKeys.window(id, rangeHours),
    queryFn: () =>
      servicesApi.historyWindow(id, new Date(Date.now() - rangeHours * 60 * 60 * 1_000)),
    enabled: id > 0,
    refetchInterval: 60_000,
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
