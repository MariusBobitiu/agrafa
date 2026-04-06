import { api } from "@/lib/fetch-client.ts";
import type { Node, NodeCreateInput, NodeResponse, NodeUpdateInput } from "@/types/node.ts";

export const nodesApi = {
  list: (projectId: number): Promise<{ nodes: Node[] }> =>
    api.get(`/nodes?project_id=${projectId}`),

  get: (id: number): Promise<{ node: Node }> =>
    api.get(`/nodes/${id}`),

  create: (payload: NodeCreateInput): Promise<{ node: NodeResponse }> =>
    api.post("/nodes", payload),

  update: (id: number, payload: NodeUpdateInput): Promise<{ node: NodeResponse }> =>
    api.patch(`/nodes/${id}`, payload),

  delete: (id: number): Promise<void> =>
    api.del(`/nodes/${id}`),

  regenerateAgentToken: (id: number): Promise<{ node_id: number; agent_token: string }> =>
    api.post(`/nodes/${id}/regenerate-agent-token`),
};
