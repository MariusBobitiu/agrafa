import { useSearchParams, useNavigate } from "react-router-dom";
import { EmptyState } from "@/components/ui/empty-state.tsx";
import { PageHeader } from "@/components/ui/page-header.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { StatusBadge } from "@/components/ui/status-badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import { useNodes } from "@/hooks/use-nodes.ts";
import { useUIStore } from "@/stores/ui-store.ts";
import { formatRelativeTime } from "@/lib/utils.ts";
import { AnimateIcon, PlusIcon, UnplugIcon } from "@/components/animate-ui/icons";
import { CreateNodeDialog } from "./components/create-node-dialog.tsx";

export function NodesPage() {
  const activeProjectId = useUIStore((s) => s.activeProjectId);
  const { data, isLoading, error } = useNodes(activeProjectId ?? 0);
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const setupOpen = searchParams.get("setup") === "1";

  function setSetupOpen(open: boolean) {
    const nextParams = new URLSearchParams(searchParams);
    if (open) {
      nextParams.set("setup", "1");
    } else {
      nextParams.delete("setup");
    }

    setSearchParams(nextParams, { replace: true });
  }

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        title="Nodes"
        description="Servers you control that can run checks for this project"
        actions={
          <AnimateIcon asChild animateOnHover>
            <Button size="sm" onClick={() => setSetupOpen(true)} disabled={!activeProjectId}>
              <PlusIcon size={14} className="mr-1.5" />
              Add node
            </Button>
          </AnimateIcon>
        }
      />

      {error && <p className="text-sm text-destructive">Failed to load nodes.</p>}

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : !data?.nodes?.length ? (
        <EmptyState
          icon={UnplugIcon}
          title="No nodes yet"
          description="Set up your first server so Agrafa can run checks from infrastructure you control."
          action={{ label: "Set up a node", onClick: () => setSetupOpen(true) }}
        />
      ) : (
        <div className="rounded-md border border-border overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/40">
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Name</th>
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">
                  Identifier
                </th>
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Status</th>
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">CPU</th>
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Memory</th>
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">
                  Last seen
                </th>
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Alerts</th>
              </tr>
            </thead>
            <tbody>
              {data.nodes.map((node) => (
                <tr
                  key={node.id}
                  className="border-b border-border last:border-0 hover:bg-muted/20 cursor-pointer"
                  onClick={() => navigate(`/nodes/${node.id}`)}
                >
                  <td className="px-4 py-2.5 font-medium">{node.name}</td>
                  <td className="px-4 py-2.5 text-muted-foreground font-mono text-xs">
                    {node.identifier}
                  </td>
                  <td className="px-4 py-2.5">
                    <StatusBadge status={node.current_state} />
                  </td>
                  <td className="px-4 py-2.5 text-muted-foreground">
                    {node.latest_cpu ? `${node.latest_cpu.value.toFixed(1)}%` : "—"}
                  </td>
                  <td className="px-4 py-2.5 text-muted-foreground">
                    {node.latest_memory ? `${node.latest_memory.value.toFixed(1)}%` : "—"}
                  </td>
                  <td className="px-4 py-2.5 text-muted-foreground">
                    {node.last_seen_at ? formatRelativeTime(node.last_seen_at) : "Never"}
                  </td>
                  <td className="px-4 py-2.5 text-muted-foreground">{node.active_alert_count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {activeProjectId && (
        <CreateNodeDialog
          projectId={activeProjectId}
          open={setupOpen}
          onOpenChange={setSetupOpen}
        />
      )}
    </div>
  );
}
