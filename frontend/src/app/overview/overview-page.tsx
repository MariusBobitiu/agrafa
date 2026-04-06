import { useQuery } from "@tanstack/react-query";
import {
  ActivityIcon,
  BellIcon,
  UnplugIcon,
  WifiIcon,
} from "@/components/animate-ui/icons/index.ts";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button.tsx";
import { EmptyState } from "@/components/ui/empty-state.tsx";
import { PageHeader } from "@/components/ui/page-header.tsx";
import { StatCard } from "@/components/ui/stat-card.tsx";
import { StatusBadge } from "@/components/ui/status-badge.tsx";
import { overviewApi } from "@/data/overview.ts";
import { useUIStore } from "@/stores/ui-store.ts";
import { formatRelativeTime } from "@/lib/utils.ts";
import { Skeleton } from "@/components/ui/skeleton.tsx";

export function OverviewPage() {
  const activeProjectId = useUIStore((s) => s.activeProjectId);
  const navigate = useNavigate();

  const { data, isLoading, error } = useQuery({
    queryKey: ["overview", activeProjectId],
    queryFn: () => overviewApi.get(activeProjectId!),
    enabled: !!activeProjectId,
    refetchInterval: 15_000,
  });
  const visibleNodes = data?.node_summaries ?? [];
  const visibleOnlineNodes = visibleNodes.filter((node) => node.current_state === "online").length;
  const visibleOfflineNodes = visibleNodes.filter(
    (node) => node.current_state === "offline",
  ).length;

  if (!activeProjectId) {
    return (
      <div className="p-6">
        <EmptyState
          icon={UnplugIcon}
          title="No project selected"
          description="Select a project from the sidebar to view your overview."
        />
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      <PageHeader title="Overview" description="Live infrastructure status" />

      {error && <p className="text-sm text-destructive">Failed to load overview data.</p>}

      {/* Stats grid */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <StatCard
          title="Total Nodes"
          value={visibleNodes.length}
          icon={UnplugIcon}
          loading={isLoading}
        />
        <StatCard
          title="Online"
          value={visibleOnlineNodes}
          icon={WifiIcon}
          description={`${visibleOfflineNodes} offline`}
          loading={isLoading}
        />
        <StatCard
          title="Services"
          value={data?.total_services ?? 0}
          icon={ActivityIcon}
          description={`${data?.services_unhealthy ?? 0} unhealthy, ${data?.services_degraded ?? 0} degraded`}
          loading={isLoading}
        />
        <StatCard
          title="Active Alerts"
          value={data?.active_alerts ?? 0}
          icon={BellIcon}
          loading={isLoading}
        />
      </div>

      <div>
        <h2 className="mb-3 text-sm font-semibold text-foreground">Services</h2>
        {isLoading ? (
          <Skeleton className="h-28 w-full" />
        ) : (data?.total_services ?? 0) === 0 ? (
          <EmptyState
            icon={ActivityIcon}
            title="No services yet"
            description="Monitor your first endpoint from this instance or a server you control."
            action={{ label: "Monitor a service", onClick: () => navigate("/services?create=1") }}
          />
        ) : (
          <div className="flex flex-col gap-4 rounded-md border border-border bg-card p-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-1">
              <p className="text-sm font-medium text-foreground">
                Monitoring {data?.total_services ?? 0} services
              </p>
              <p className="text-sm text-muted-foreground">
                {data?.services_healthy ?? 0} healthy, {data?.services_degraded ?? 0} degraded,{" "}
                {data?.services_unhealthy ?? 0} unhealthy
              </p>
            </div>
            <Button variant="outline" size="sm" onClick={() => navigate("/services")}>
              View services
            </Button>
          </div>
        )}
      </div>

      {/* Node list */}
      <div>
        <h2 className="mb-3 text-sm font-semibold text-foreground">Nodes</h2>
        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : !data?.node_summaries?.length ? (
          <EmptyState
            icon={UnplugIcon}
            title="No nodes yet"
            description="Set up your first node so Agrafa can run checks from infrastructure you control."
            action={{ label: "Set up a node", onClick: () => navigate("/nodes?setup=1") }}
          />
        ) : (
          <div className="rounded-md border border-border overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-muted/40">
                  <th className="px-4 py-2 text-left font-medium text-muted-foreground">Name</th>
                  <th className="px-4 py-2 text-left font-medium text-muted-foreground">Status</th>
                  <th className="px-4 py-2 text-left font-medium text-muted-foreground">
                    Last seen
                  </th>
                  <th className="px-4 py-2 text-left font-medium text-muted-foreground">
                    Services
                  </th>
                  <th className="px-4 py-2 text-left font-medium text-muted-foreground">Alerts</th>
                </tr>
              </thead>
              <tbody>
                {data.node_summaries.map((node) => (
                  <tr key={node.id} className="border-b border-border last:border-0">
                    <td className="px-4 py-2 font-medium">{node.name}</td>
                    <td className="px-4 py-2">
                      <StatusBadge status={node.current_state} />
                    </td>
                    <td className="px-4 py-2 text-muted-foreground">
                      {node.last_seen_at ? formatRelativeTime(node.last_seen_at) : "Never"}
                    </td>
                    <td className="px-4 py-2 text-muted-foreground">{node.service_count}</td>
                    <td className="px-4 py-2 text-muted-foreground">{node.active_alert_count}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
