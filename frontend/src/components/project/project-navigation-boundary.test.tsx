// @vitest-environment happy-dom

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, type ReactNode, useState } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter, useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";
import { settingsTabFromSearchParams } from "@/app/settings/settings-tabs.ts";
import { ProjectNavigationBoundary } from "@/components/project/project-navigation-boundary.tsx";
import { projectIdFromSearchParams } from "@/components/project/project-navigation.ts";
import { useUIStore } from "@/stores/ui-store.ts";

const roots: Root[] = [];
useUIStore.persist.setOptions({
  storage: {
    getItem: () => null,
    removeItem: () => undefined,
    setItem: () => undefined,
  },
});
const projects = [
  {
    id: 1,
    slug: "project-a",
    name: "Project A",
    created_at: "2026-08-30T10:00:00Z",
    current_user_role: "admin",
  },
  {
    id: 2,
    slug: "project-b",
    name: "Project B",
    created_at: "2026-08-30T10:00:00Z",
    current_user_role: "admin",
  },
];

function Probe() {
  const activeProjectId = useUIStore((state) => state.activeProjectId);
  const [searchParams] = useSearchParams();
  return (
    <div
      data-active-project={activeProjectId}
      data-project-param={searchParams.get("project_id") ?? ""}
      data-settings-tab={settingsTabFromSearchParams(searchParams, false)}
    />
  );
}

function NavigationProbe() {
  const navigate = useNavigate();
  const location = useLocation();
  const activeProjectId = useUIStore((state) => state.activeProjectId);
  const [resourceFilter, setResourceFilter] = useState("all");

  return (
    <div
      data-active-project={activeProjectId}
      data-path={location.pathname + location.search}
      data-resource-filter={resourceFilter}
    >
      <button type="button" onClick={() => setResourceFilter("node:22")}>
        Filter project B
      </button>
      <button type="button" onClick={() => navigate("/alerts?project_id=1")}>
        Open project A email link
      </button>
      <button type="button" onClick={() => navigate(-1)}>
        Back
      </button>
    </div>
  );
}

async function renderBoundary(path: string, child: ReactNode = <Probe />) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Number.POSITIVE_INFINITY } },
  });
  client.setQueryData(["projects"], { projects });
  const container = document.createElement("div");
  document.body.append(container);
  const root = createRoot(container);
  roots.push(root);

  await act(async () => {
    root.render(
      <QueryClientProvider client={client}>
        <MemoryRouter initialEntries={[path]}>
          <ProjectNavigationBoundary>{child}</ProjectNavigationBoundary>
        </MemoryRouter>
      </QueryClientProvider>,
    );
  });

  return container;
}

afterEach(() => {
  for (const root of roots.splice(0)) {
    act(() => root.unmount());
  }
  document.body.replaceChildren();
  useUIStore.setState({ activeProjectId: null });
});

describe("project-aware email navigation", () => {
  it("accepts only strict positive integer project IDs", () => {
    expect(projectIdFromSearchParams(new URLSearchParams("project_id=12"))).toBe(12);
    expect(projectIdFromSearchParams(new URLSearchParams("project_id=0"))).toBeNull();
    expect(projectIdFromSearchParams(new URLSearchParams("project_id=1.5"))).toBeNull();
    expect(projectIdFromSearchParams(new URLSearchParams("project_id=project-a"))).toBeNull();
  });

  it("selects the accessible email project instead of the persisted project", async () => {
    useUIStore.setState({ activeProjectId: 2 });
    const container = await renderBoundary("/alerts?project_id=1");

    expect(container.firstElementChild?.getAttribute("data-active-project")).toBe("1");
  });

  it("selects the email project while preserving the notifications tab", async () => {
    useUIStore.setState({ activeProjectId: 2 });
    const container = await renderBoundary("/settings?tab=notifications&project_id=1");
    const probe = container.firstElementChild;

    expect(probe?.getAttribute("data-active-project")).toBe("1");
    expect(probe?.getAttribute("data-project-param")).toBe("1");
    expect(probe?.getAttribute("data-settings-tab")).toBe("notifications");
  });

  it("does not overwrite the current project for an invalid project ID", async () => {
    useUIStore.setState({ activeProjectId: 2 });
    const container = await renderBoundary("/alerts?project_id=not-a-number");

    expect(container.firstElementChild?.getAttribute("data-active-project")).toBe("2");
  });

  it("does not activate a project the current user cannot access", async () => {
    useUIStore.setState({ activeProjectId: 2 });
    const container = await renderBoundary("/alerts?project_id=999");

    expect(container.firstElementChild?.getAttribute("data-active-project")).toBe("2");
  });

  it("preserves the persisted project when no project ID is present", async () => {
    useUIStore.setState({ activeProjectId: 2 });
    const container = await renderBoundary("/alerts");

    expect(container.firstElementChild?.getAttribute("data-active-project")).toBe("2");
  });

  it("remounts project-scoped content without stale filters and follows history", async () => {
    useUIStore.setState({ activeProjectId: 2 });
    const container = await renderBoundary("/alerts?project_id=2", <NavigationProbe />);
    const buttons = () => Array.from(container.querySelectorAll("button"));

    await act(async () => buttons()[0]?.click());
    expect(container.firstElementChild?.getAttribute("data-resource-filter")).toBe("node:22");

    await act(async () => buttons()[1]?.click());
    expect(container.firstElementChild?.getAttribute("data-active-project")).toBe("1");
    expect(container.firstElementChild?.getAttribute("data-resource-filter")).toBe("all");

    await act(async () => buttons()[2]?.click());
    expect(container.firstElementChild?.getAttribute("data-path")).toBe("/alerts?project_id=2");
    expect(container.firstElementChild?.getAttribute("data-active-project")).toBe("2");
    expect(container.firstElementChild?.getAttribute("data-resource-filter")).toBe("all");
  });
});
