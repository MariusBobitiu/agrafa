import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { projectMembersApi } from "@/data/project-members.ts";
import { projectInvitationsApi } from "@/data/project-invitations.ts";
import type { ProjectMemberRole } from "@/types/project-member.ts";
import type { ProjectInvitationCreateBatchInput } from "@/types/project-invitation.ts";

export function useProjectMembers(projectId: number) {
  return useQuery({
    queryKey: ["project-members", projectId],
    queryFn: () => projectMembersApi.list(projectId),
    enabled: projectId > 0,
  });
}

export function useUpdateMemberRole(projectId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, role }: { id: string; role: ProjectMemberRole }) =>
      projectMembersApi.updateRole(id, role),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["project-members", projectId] }),
  });
}

export function useRemoveMember(projectId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => projectMembersApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["project-members", projectId] }),
  });
}

export function useInviteMembers(projectId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: ProjectInvitationCreateBatchInput) =>
      projectInvitationsApi.createBatch(payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["project-invitations", projectId] });
    },
  });
}

export function useProjectInvitations(projectId: number) {
  return useQuery({
    queryKey: ["project-invitations", projectId],
    queryFn: () => projectInvitationsApi.list(projectId),
    enabled: projectId > 0,
  });
}

export function useRevokeInvitation(projectId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => projectInvitationsApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["project-invitations", projectId] }),
  });
}
