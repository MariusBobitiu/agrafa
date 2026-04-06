import { api } from "@/lib/fetch-client.ts";
import type { Overview } from "@/types/overview.ts";

export const overviewApi = {
  get: (projectId: number): Promise<Overview> =>
    api.get(`/overview?project_id=${projectId}`),
};
