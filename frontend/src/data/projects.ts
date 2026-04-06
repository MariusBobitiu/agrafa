import { api } from "@/lib/fetch-client.ts";
import type { Project, ProjectDetail, ProjectSummary } from "@/types/project.ts";

export const projectsApi = {
  list: (): Promise<{ projects: ProjectSummary[] }> =>
    api.get("/projects"),

  get: (id: number): Promise<{ project: ProjectDetail }> =>
    api.get(`/projects/${id}`),

  create: (payload: { name: string }): Promise<{ project: Project }> =>
    api.post("/projects", payload),

  update: (id: number, payload: { name?: string }): Promise<{ project: Project }> =>
    api.patch(`/projects/${id}`, payload),

  delete: (id: number): Promise<void> =>
    api.del(`/projects/${id}`),
};
