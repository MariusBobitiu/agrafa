import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Button } from "@/components/ui/button.tsx";
import { EmptyState } from "@/components/ui/empty-state.tsx";
import { PageHeader } from "@/components/ui/page-header.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { StatusBadge } from "@/components/ui/status-badge.tsx";
import { useServices } from "@/hooks/use-services.ts";
import { formatRelativeTime } from "@/lib/utils.ts";
import { useServiceCreationStore } from "@/stores/service-creation-store.ts";
import { useUIStore } from "@/stores/ui-store.ts";
import { CreateNodeDialog } from "../nodes/components/create-node-dialog.tsx";
import { CreateServiceDialog } from "./components/create-service-dialog.tsx";
import { AnimateIcon, PlusIcon, ActivityIcon } from "@/components/animate-ui/icons";

function getExecutionLocationLabel(executionMode: "managed" | "agent") {
  return executionMode === "managed"
    ? {
        title: "This instance",
        description: "Current Agrafa server",
      }
    : {
        title: "My own server",
        description: "A server you control",
      };
}

export function ServicesPage() {
  const activeProjectId = useUIStore((s) => s.activeProjectId);
  const markServiceDraftResume = useServiceCreationStore((s) => s.markResume);
  const { data, isLoading, error } = useServices(activeProjectId ?? 0);
  const [searchParams, setSearchParams] = useSearchParams();
  const createOpen = searchParams.get("create") === "1";
  const [nodeSetupOpen, setNodeSetupOpen] = useState(false);
  const [, setNow] = useState(() => Date.now());

  useEffect(() => {
    const timer = window.setInterval(() => {
      setNow(Date.now());
    }, 1000);

    return () => {
      window.clearInterval(timer);
    };
  }, []);

  function setCreateOpen(open: boolean) {
    const nextParams = new URLSearchParams(searchParams);
    if (open) {
      nextParams.set("create", "1");
    } else {
      nextParams.delete("create");
    }

    setSearchParams(nextParams, { replace: true });
  }

  function handleRequestNodeSetup() {
    setCreateOpen(false);
    setNodeSetupOpen(true);
  }

  function handleNodeSetupOpenChange(open: boolean) {
    setNodeSetupOpen(open);
    if (!open) {
      markServiceDraftResume();
      setCreateOpen(true);
    }
  }

  function handleNodeSetupComplete(nodeId: number) {
    markServiceDraftResume(nodeId.toString());
    setNodeSetupOpen(false);
    setCreateOpen(true);
  }

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        title="Services"
        description="HTTP and TCP service health checks"
        actions={
          <AnimateIcon asChild animateOnHover>
            <Button size="sm" onClick={() => setCreateOpen(true)} disabled={!activeProjectId}>
              <PlusIcon size={14} className="mr-1.5" />
              Add service
            </Button>
          </AnimateIcon>
        }
      />

      {error && <p className="text-sm text-destructive">Failed to load services.</p>}

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : !data?.services?.length ? (
        <EmptyState
          icon={ActivityIcon}
          title="No services yet"
          description="Monitor a service from this instance or your own server."
          action={{ label: "Monitor a service", onClick: () => setCreateOpen(true) }}
        />
      ) : (
        <div className="rounded-md border border-border overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/40">
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Name</th>
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">
                  Runs from
                </th>
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Type</th>
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Target</th>
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Status</th>
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">
                  Last checked
                </th>
              </tr>
            </thead>
            <tbody>
              {data.services.map((svc) => (
                <tr key={svc.id} className="border-b border-border last:border-0 hover:bg-muted/20">
                  <td className="px-4 py-2.5 font-medium">{svc.name}</td>
                  <td className="px-4 py-2.5">
                    {(() => {
                      const location = getExecutionLocationLabel(svc.execution_mode);
                      return (
                        <div>
                          <p className="font-medium text-foreground">{location.title}</p>
                          <p className="text-xs text-muted-foreground">{location.description}</p>
                        </div>
                      );
                    })()}
                  </td>
                  <td className="px-4 py-2.5 text-muted-foreground uppercase text-xs">
                    {svc.check_type}
                  </td>
                  <td className="px-4 py-2.5 text-muted-foreground font-mono text-xs truncate max-w-48">
                    {svc.check_target}
                  </td>
                  <td className="px-4 py-2.5">
                    <StatusBadge status={svc.status} />
                  </td>
                  <td className="px-4 py-2.5 text-muted-foreground">
                    {svc.last_checked_at ? formatRelativeTime(svc.last_checked_at) : "Never"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {activeProjectId && (
        <>
          <CreateServiceDialog
            projectId={activeProjectId}
            open={createOpen}
            onOpenChange={setCreateOpen}
            onRequestNodeSetup={handleRequestNodeSetup}
          />
          <CreateNodeDialog
            projectId={activeProjectId}
            open={nodeSetupOpen}
            onOpenChange={handleNodeSetupOpenChange}
            launchedFromService
            onComplete={handleNodeSetupComplete}
          />
        </>
      )}
    </div>
  );
}
