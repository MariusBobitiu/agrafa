import type { ProjectMemberRole } from "@/types/project-member.ts";
import { useProjectDetail } from "@/hooks/use-projects.ts";

/**
 * Returns the current user's role in the given project.
 * Reads from the cached project detail query (current_user_role).
 */
export function useProjectRole(projectId: number): ProjectMemberRole | null {
  const { data } = useProjectDetail(projectId);
  const role = data?.project.current_user_role;
  if (role === "owner" || role === "admin" || role === "viewer") return role;
  return null;
}

export function useCanManageMembers(projectId: number): boolean {
  const role = useProjectRole(projectId);
  return role === "owner";
}

export function useCanWrite(projectId: number): boolean {
  const role = useProjectRole(projectId);
  return role === "owner" || role === "admin";
}

export function useCanDeleteProject(projectId: number): boolean {
  const role = useProjectRole(projectId);
  return role === "owner";
}
