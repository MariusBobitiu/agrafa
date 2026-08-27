import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { alertsApi } from "@/data/alerts.ts";
import type {
  AlertHistoryFilters,
  AlertRuleCreateInput,
  AlertRuleUpdateInput,
} from "@/types/alert.ts";

export const alertHistoryKeys = {
  active: (projectId: number) => ["alerts", projectId, "active"] as const,
  history: (projectId: number, filters: AlertHistoryFilters, limit: number) =>
    ["alerts", projectId, "history", filters, limit] as const,
};

export function useAlerts(projectId: number) {
  return useQuery({
    queryKey: ["alerts", projectId],
    queryFn: () => alertsApi.listAlerts(projectId),
    enabled: projectId > 0,
  });
}

export function useActiveAlerts(projectId: number) {
  return useQuery({
    queryKey: alertHistoryKeys.active(projectId),
    queryFn: () => alertsApi.listActive(projectId),
    enabled: projectId > 0,
  });
}

export function useAlertHistory(projectId: number, filters: AlertHistoryFilters, limit = 25) {
  return useInfiniteQuery({
    queryKey: alertHistoryKeys.history(projectId, filters, limit),
    queryFn: ({ pageParam }) =>
      alertsApi.listHistory(projectId, filters, limit, pageParam ?? undefined),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => lastPage.pagination.nextCursor ?? undefined,
    enabled: projectId > 0,
  });
}

export function useAlertRules(projectId: number) {
  return useQuery({
    queryKey: ["alert-rules", projectId],
    queryFn: () => alertsApi.listRules(projectId),
    enabled: projectId > 0,
  });
}

export function useCreateAlertRule(projectId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: AlertRuleCreateInput) => alertsApi.createRule(payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-rules", projectId] }),
  });
}

export function useUpdateAlertRule(projectId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: AlertRuleUpdateInput }) =>
      alertsApi.updateRule(id, payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-rules", projectId] }),
  });
}

export function useDeleteAlertRule(projectId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => alertsApi.deleteRule(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-rules", projectId] }),
  });
}
