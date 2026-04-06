import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { alertsApi } from "@/data/alerts.ts";
import type { AlertRuleCreateInput, AlertRuleUpdateInput } from "@/types/alert.ts";

export function useAlerts(projectId: number) {
  return useQuery({
    queryKey: ["alerts", projectId],
    queryFn: () => alertsApi.listAlerts(projectId),
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
