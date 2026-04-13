import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { servicesApi } from "@/data/services.ts";
import type { ServiceCreateInput, ServiceUpdateInput } from "@/types/service.ts";

export function useServices(projectId: number) {
  return useQuery({
    queryKey: ["services", projectId],
    queryFn: () => servicesApi.list(projectId),
    enabled: projectId > 0,
    refetchInterval: 10_000,
  });
}

export function useService(id: number) {
  return useQuery({
    queryKey: ["services", "detail", id],
    queryFn: () => servicesApi.get(id),
    enabled: id > 0,
    refetchInterval: 10_000,
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
