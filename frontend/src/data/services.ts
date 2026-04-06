import { api } from "@/lib/fetch-client.ts";
import type {
  Service,
  ServiceCreateInput,
  ServiceUpdateInput,
} from "@/types/service.ts";

export const servicesApi = {
  list: (projectId: number): Promise<{ services: Service[] }> =>
    api.get(`/services?project_id=${projectId}`),

  get: (id: number): Promise<{ service: Service }> =>
    api.get(`/services/${id}`),

  create: (payload: ServiceCreateInput): Promise<{ service: Service }> =>
    api.post("/services", payload),

  update: (id: number, payload: ServiceUpdateInput): Promise<{ service: Service }> =>
    api.patch(`/services/${id}`, payload),

  delete: (id: number): Promise<void> =>
    api.del(`/services/${id}`),
};
