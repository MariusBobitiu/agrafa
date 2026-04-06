import { api } from "@/lib/fetch-client.ts";
import type {
  ProjectInvitation,
  ProjectInvitationAcceptInput,
  ProjectInvitationAcceptResponse,
  ProjectInvitationCreateBatchInput,
  ProjectInvitationCreateBatchResponse,
  ProjectInvitationLookupResponse,
} from "@/types/project-invitation.ts";

export const projectInvitationsApi = {
  createBatch: (
    payload: ProjectInvitationCreateBatchInput,
  ): Promise<ProjectInvitationCreateBatchResponse> =>
    api.post("/project-invitations", payload),

  list: (projectId: number): Promise<{ project_invitations: ProjectInvitation[] }> =>
    api.get(`/project-invitations?project_id=${projectId}`),

  getByToken: (token: string): Promise<ProjectInvitationLookupResponse> =>
    api.get(`/project-invitations/by-token?token=${encodeURIComponent(token)}`),

  accept: (
    payload: ProjectInvitationAcceptInput,
  ): Promise<ProjectInvitationAcceptResponse> =>
    api.post("/project-invitations/accept", payload),

  delete: (id: string): Promise<void> =>
    api.del(`/project-invitations/${id}`),
};
