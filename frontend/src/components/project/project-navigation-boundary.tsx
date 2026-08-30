import { Fragment, type ReactNode, useEffect } from "react";
import { useSearchParams } from "react-router-dom";
import { useProjects } from "@/hooks/use-projects.ts";
import { useUIStore } from "@/stores/ui-store.ts";
import { projectIdFromSearchParams } from "./project-navigation.ts";

export function ProjectNavigationBoundary({ children }: { children: ReactNode }) {
  const [searchParams] = useSearchParams();
  const requestedProjectId = projectIdFromSearchParams(searchParams);
  const activeProjectId = useUIStore((state) => state.activeProjectId);
  const setActiveProjectId = useUIStore((state) => state.setActiveProjectId);
  const projectsQuery = useProjects();
  const requestedProjectIsAccessible =
    requestedProjectId !== null &&
    projectsQuery.data?.projects.some((project) => project.id === requestedProjectId) === true;

  useEffect(() => {
    if (requestedProjectIsAccessible && activeProjectId !== requestedProjectId) {
      setActiveProjectId(requestedProjectId);
    }
  }, [activeProjectId, requestedProjectId, requestedProjectIsAccessible, setActiveProjectId]);

  // A well-formed email project ID must be checked against the current user's
  // project list before project-scoped page hooks are allowed to mount.
  if (requestedProjectId !== null && projectsQuery.data === undefined && !projectsQuery.isError) {
    return null;
  }

  if (requestedProjectIsAccessible && activeProjectId !== requestedProjectId) {
    return null;
  }

  return <Fragment key={activeProjectId}>{children}</Fragment>;
}
