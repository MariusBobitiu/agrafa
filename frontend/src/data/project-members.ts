import { api } from "@/lib/fetch-client.ts";
import type { ProjectMember, ProjectMemberRole } from "@/types/project-member.ts";

export const projectMembersApi = {
  list: (projectId: number): Promise<{ project_members: ProjectMember[] }> =>
    api.get(`/project-members?project_id=${projectId}`),

  updateRole: (id: string, role: ProjectMemberRole): Promise<{ project_member: ProjectMember }> =>
    api.patch(`/project-members/${id}`, { role }),

  delete: (id: string): Promise<void> =>
    api.del(`/project-members/${id}`),
};
