import { type Query, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { nodesApi } from "@/data/nodes.ts";
import { useSSE } from "@/hooks/use-sse.ts";
import type { Node, NodeCreateInput, NodeUpdateInput } from "@/types/node.ts";

export function useNodes(projectId: number, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ["nodes", projectId],
    queryFn: () => nodesApi.list(projectId),
    enabled: (options?.enabled ?? true) && projectId > 0,
  });
}

export function useNodeDetailStream(id: number, options?: { enabled?: boolean }) {
  const qc = useQueryClient();

  useSSE<{ node: Node }>({
    enabled: (options?.enabled ?? true) && id > 0,
    path: `/nodes/${id}/stream`,
    onMessage: (payload) => {
      qc.setQueryData(["nodes", "detail", id], payload);
    },
  });
}

export function useNode(
  id: number,
  options?: {
    enabled?: boolean;
    refetchInterval?:
      | number
      | false
      | ((
          query: Query<{ node: Awaited<ReturnType<typeof nodesApi.get>>["node"] }, Error>,
        ) => number | false);
  },
) {
  return useQuery({
    queryKey: ["nodes", "detail", id],
    queryFn: () => nodesApi.get(id),
    enabled: (options?.enabled ?? true) && id > 0,
    refetchInterval: options?.refetchInterval,
  });
}

export function useCreateNode(projectId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: NodeCreateInput) => nodesApi.create(payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["nodes", projectId] });
      void qc.invalidateQueries({ queryKey: ["overview", projectId] });
    },
  });
}

export function useUpdateNode(projectId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: NodeUpdateInput }) =>
      nodesApi.update(id, payload),
    onSuccess: (_, variables) => {
      void qc.invalidateQueries({ queryKey: ["nodes", projectId] });
      void qc.invalidateQueries({ queryKey: ["nodes", "detail", variables.id] });
      void qc.invalidateQueries({ queryKey: ["overview", projectId] });
    },
  });
}

export function useDeleteNode(projectId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => nodesApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["nodes", projectId] }),
  });
}

export function useRegenerateAgentToken() {
  return useMutation({
    mutationFn: (id: number) => nodesApi.regenerateAgentToken(id),
  });
}
