import { useState } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import {
  BellIcon as LucideBellIcon,
  ChevronRightIcon,
  CpuIcon,
  HardDriveIcon,
  MemoryStickIcon,
  ServerIcon,
  TrashIcon,
} from "lucide-react";
import { toast } from "sonner";
import { PageHeader } from "@/components/ui/page-header.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { StatusBadge } from "@/components/ui/status-badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import { useNodes, useDeleteNode } from "@/hooks/use-nodes.ts";
import { useUIStore } from "@/stores/ui-store.ts";
import { formatRelativeTime } from "@/lib/utils.ts";
import { cn } from "@/lib/utils.ts";
import { AnimateIcon, PlusIcon } from "@/components/animate-ui/icons";
import { CreateNodeDialog } from "./components/create-node-dialog.tsx";
import { ConfirmDialog } from "@/components/ui/confirm-dialog.tsx";
import type { Node } from "@/types/node.ts";
import { InlineMetric } from "@/components/inline-metric.tsx";
// ─── Node row card ────────────────────────────────────────────────────────────

function NodeRowCard({ node, onDelete }: { node: Node; onDelete: (id: number) => void }) {
  const navigate = useNavigate();
  const [deleteOpen, setDeleteOpen] = useState(false);
  const cpu = node.latest_cpu?.value ?? null;
  const mem = node.latest_memory?.value ?? null;
  const disk = node.latest_disk?.value ?? null;
  const hasMetrics = cpu !== null || mem !== null || disk !== null;
  const isOnline = node.current_state === "online";
  const isOffline = node.current_state === "offline";

  return (
    <>
      <div
        role="button"
        tabIndex={0}
        onClick={() => navigate(`/nodes/${node.id}`)}
        onKeyDown={(e) => e.key === "Enter" && navigate(`/nodes/${node.id}`)}
        className={cn(
          "group flex items-center gap-6 rounded-xl border px-5 py-3.5 cursor-pointer transition-colors",
          "hover:bg-muted/30",
          isOffline ? "border-destructive/25 bg-destructive/2" : "border-border bg-card",
        )}
      >
        {/* ── Left: identity ── */}
        <div className="flex items-center gap-3 w-56 shrink-0 min-w-0">
          <div
            className={cn(
              "flex size-9 items-center justify-center rounded-lg shrink-0",
              isOnline ? "bg-primary/10" : isOffline ? "bg-destructive/10" : "bg-muted",
            )}
          >
            <ServerIcon
              size={16}
              className={cn(
                isOnline
                  ? "text-primary"
                  : isOffline
                    ? "text-destructive"
                    : "text-muted-foreground",
              )}
            />
          </div>
          <div className="min-w-0">
            <p className="text-sm font-semibold text-foreground truncate leading-snug">
              {node.name}
            </p>
            <p className="text-xs text-muted-foreground/70 mt-0.5 truncate">
              {node.last_seen_at
                ? isOnline
                  ? `Active · ${formatRelativeTime(node.last_seen_at)}`
                  : `Last seen ${formatRelativeTime(node.last_seen_at)}`
                : "Never seen"}
            </p>
          </div>
        </div>

        {/* ── Middle: compact metrics ── */}
        <div className="flex items-center gap-4 flex-1 min-w-0">
          {hasMetrics ? (
            <>
              {cpu !== null && (
                <InlineMetric icon={<CpuIcon size={10} />} label="CPU" value={cpu} variant="cpu" />
              )}
              {mem !== null && (
                <InlineMetric
                  icon={<MemoryStickIcon size={10} />}
                  label="Mem"
                  value={mem}
                  variant="mem"
                />
              )}
              {disk !== null && (
                <InlineMetric
                  icon={<HardDriveIcon size={10} />}
                  label="Disk"
                  value={disk}
                  variant="disk"
                />
              )}
            </>
          ) : (
            <p className="text-xs text-muted-foreground/40 italic">
              {isOnline ? "Waiting for metrics…" : "No metrics"}
            </p>
          )}
        </div>

        {/* ── Right: counts + badge + delete + chevron ── */}
        <div className="flex items-center gap-4 shrink-0">
          <div className="flex items-center gap-3 text-xs text-muted-foreground">
            <span className="flex items-center gap-1.5">
              <ServerIcon size={11} className="text-muted-foreground/50" />
              <span>{node.service_count}</span>
            </span>
            <span
              className={cn(
                "flex items-center gap-1.5",
                node.active_alert_count > 0 ? "text-destructive font-medium" : "opacity-30",
              )}
            >
              <LucideBellIcon size={11} />
              <span>{node.active_alert_count}</span>
            </span>
          </div>
          <StatusBadge status={node.current_state} />
          <Button
            variant="ghost"
            size="icon-sm"
            className="text-muted-foreground/30 hover:bg-destructive hover:text-foreground opacity-0 group-hover:opacity-100 transition-opacity"
            onClick={(e) => {
              e.stopPropagation();
              setDeleteOpen(true);
            }}
          >
            <TrashIcon size={13} />
          </Button>
          <ChevronRightIcon
            size={13}
            className="text-muted-foreground/20 group-hover:text-muted-foreground/60 transition-colors ml-1"
          />
        </div>
      </div>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete node"
        description="This will permanently delete the node. The node must have no services assigned before it can be deleted."
        onConfirm={() => {
          onDelete(node.id);
          setDeleteOpen(false);
        }}
      />
    </>
  );
}

// ─── Skeleton row ─────────────────────────────────────────────────────────────

function NodeRowSkeleton() {
  return (
    <div className="flex items-center gap-4 rounded-xl border border-border bg-card px-5 py-4">
      <div className="flex items-center gap-3 w-52 shrink-0">
        <Skeleton className="h-8 w-8 rounded-lg" />
        <div className="space-y-1.5">
          <Skeleton className="h-3.5 w-28" />
          <Skeleton className="h-3 w-20" />
        </div>
      </div>
      <div className="flex-1 space-y-1.5">
        <Skeleton className="h-2 w-full" />
        <Skeleton className="h-2 w-full" />
        <Skeleton className="h-2 w-4/5" />
      </div>
      <div className="flex items-center gap-3 shrink-0">
        <Skeleton className="h-3 w-10" />
        <Skeleton className="h-5 w-16 rounded-full" />
      </div>
    </div>
  );
}

// ─── Empty state ──────────────────────────────────────────────────────────────

function NodesEmptyState({ onSetup }: { onSetup: () => void }) {
  return (
    <div className="rounded-xl border border-dashed border-border bg-card/50 px-8 py-14 text-center">
      <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-muted">
        <ServerIcon size={20} className="text-muted-foreground" />
      </div>
      <h3 className="text-sm font-semibold text-foreground">No nodes yet</h3>
      <p className="mt-1.5 text-sm text-muted-foreground max-w-xs mx-auto">
        Set up your first server so Agrafa can run checks from infrastructure you control.
      </p>
      <Button size="sm" className="mt-5 gap-1.5" onClick={onSetup}>
        <PlusIcon size={13} />
        Set up a node
      </Button>
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export function NodesPage() {
  const activeProjectId = useUIStore((s) => s.activeProjectId);
  const { data, isLoading, error } = useNodes(activeProjectId ?? 0);
  const deleteNode = useDeleteNode(activeProjectId ?? 0);
  const [searchParams, setSearchParams] = useSearchParams();
  const setupOpen = searchParams.get("setup") === "1";

  function handleDelete(id: number) {
    deleteNode.mutate(id, {
      onSuccess: () => toast.success("Node deleted"),
      onError: () =>
        toast.error("Failed to delete node. Remove all services from this node first."),
    });
  }

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
    <div className="p-6 space-y-6 max-w-6xl mx-auto">
      <PageHeader
        title="Nodes"
        description="Servers you control that can run checks for this project"
        actions={
          <AnimateIcon asChild animateOnHover>
            <Button
              size="sm"
              variant={"secondary"}
              onClick={() => setSetupOpen(true)}
              disabled={!activeProjectId}
            >
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
            <NodeRowSkeleton key={i} />
          ))}
        </div>
      ) : !data?.nodes?.length ? (
        <NodesEmptyState onSetup={() => setSetupOpen(true)} />
      ) : (
        <div className="space-y-2">
          {data.nodes.map((node) => (
            <NodeRowCard key={node.id} node={node} onDelete={handleDelete} />
          ))}
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
