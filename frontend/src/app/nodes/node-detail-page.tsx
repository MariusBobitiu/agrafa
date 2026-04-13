import { useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import {
  AlertTriangleIcon,
  ArrowLeftIcon,
  ChevronRightIcon,
  CpuIcon,
  HardDriveIcon,
  MemoryStickIcon,
  PencilIcon,
  ServerIcon,
  TrashIcon,
} from "lucide-react";
import { toast } from "sonner";
import { ActivityIcon, UnplugIcon, WifiIcon } from "@/components/animate-ui/icons/index.ts";
import { Button } from "@/components/ui/button.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { StatusBadge } from "@/components/ui/status-badge.tsx";
import { useNode, useDeleteNode } from "@/hooks/use-nodes.ts";
import { useServices } from "@/hooks/use-services.ts";
import { useAlerts } from "@/hooks/use-alerts.ts";
import { useUIStore } from "@/stores/ui-store.ts";
import { formatRelativeTime, formatDate } from "@/lib/utils.ts";
import { cn } from "@/lib/utils.ts";
import { MetaItem } from "@/components/meta-item.tsx";
import { SectionHeading } from "@/components/section-heading.tsx";
import { ConfirmDialog } from "@/components/ui/confirm-dialog.tsx";
import { CreateNodeDialog } from "./components/create-node-dialog.tsx";
import type { Node } from "@/types/node.ts";
import type { Service } from "@/types/service.ts";
import type { Alert } from "@/types/alert.ts";
import { useMeta } from "@/hooks/use-meta.ts";

// ─── Gauge ────────────────────────────────────────────────────────────────────

type GaugeProps = {
  label: string;
  value: number | null;
  icon: React.ReactNode;
  loading?: boolean;
};

function gaugeAccent(value: number) {
  if (value >= 90) return { stroke: "#ef4444", text: "text-destructive" };
  if (value >= 70) return { stroke: "#eab308", text: "text-warning" };
  return { stroke: "#84cc16", text: "text-primary" };
}

function MetricGauge({ label, value, icon, loading }: GaugeProps) {
  const size = 140;
  const sw = 9; // strokeWidth
  const r = (size - sw) / 2;
  const cx = size / 2;
  const cy = size / 2;

  // 220° arc, opening downward. Start angle in SVG coords (y-down).
  // We want the gap (40°) centered at the bottom, so arc runs from 160° to 20°
  // going clockwise (which in SVG is positive direction).
  // In SVG coords: 0° = 3 o'clock, clockwise positive.
  const startDeg = 160; // bottom-left
  const endDeg = 20; // bottom-right (= 160 + 220 mod 360)

  function polarToXY(deg: number) {
    const rad = (deg * Math.PI) / 180;
    return {
      x: cx + r * Math.cos(rad),
      y: cy + r * Math.sin(rad),
    };
  }

  function describeArc(fromDeg: number, sweepDeg: number) {
    const start = polarToXY(fromDeg);
    const end = polarToXY(fromDeg + sweepDeg);
    const large = sweepDeg > 180 ? 1 : 0;
    return `M ${start.x} ${start.y} A ${r} ${r} 0 ${large} 1 ${end.x} ${end.y}`;
  }

  const totalSweep = 220;
  const pct = value != null ? Math.min(Math.max(value, 0), 100) : 0;
  const fillSweep = (pct / 100) * totalSweep;
  const accent = value != null ? gaugeAccent(value) : null;

  void endDeg; // used only for documentation

  return (
    <div className="flex flex-col items-center gap-2">
      <div className="relative" style={{ width: size, height: size }}>
        <svg width={size} height={size}>
          {/* Track */}
          <path
            d={describeArc(startDeg, totalSweep)}
            fill="none"
            stroke="currentColor"
            className="text-muted"
            strokeWidth={sw}
            strokeLinecap="round"
          />
          {/* Fill */}
          {value != null && fillSweep > 0 && (
            <path
              d={describeArc(startDeg, fillSweep)}
              fill="none"
              stroke={accent!.stroke}
              strokeWidth={sw}
              strokeLinecap="round"
            />
          )}
        </svg>

        {/* Center value */}
        <div className="absolute inset-0 flex flex-col items-center justify-center gap-0.5">
          {loading ? (
            <Skeleton className="h-7 w-16" />
          ) : value != null ? (
            <span className={cn("text-2xl font-bold tabular-nums leading-none", accent!.text)}>
              {value.toFixed(1)}
              <span className="text-base font-normal opacity-70">%</span>
            </span>
          ) : (
            <span className="text-2xl font-bold text-muted-foreground/20">—</span>
          )}
        </div>
      </div>

      {/* Label */}
      <div className="flex items-center gap-1.5">
        <span className="text-muted-foreground">{icon}</span>
        <span className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">
          {label}
        </span>
      </div>
    </div>
  );
}

// ─── Services ─────────────────────────────────────────────────────────────────

function statusAccentClass(status: Service["status"]) {
  switch (status) {
    case "healthy":
      return "bg-primary";
    case "degraded":
      return "bg-warning";
    case "unhealthy":
      return "bg-destructive";
    default:
      return "bg-muted-foreground/30";
  }
}

function NodeServiceList({ services }: { services: Service[] }) {
  if (services.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-border px-4 py-5 text-center">
        <p className="text-sm text-muted-foreground">No services assigned to this node.</p>
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-border bg-card overflow-hidden divide-y divide-border">
      {services.map((svc) => (
        <div key={svc.id} className="flex items-stretch hover:bg-muted/20 transition-colors">
          {/* left accent stripe */}
          <div className={cn("w-0.5 shrink-0", statusAccentClass(svc.status))} />
          <div className="flex flex-1 items-center justify-between px-4 py-3 min-w-0">
            <div className="min-w-0">
              <p className="text-sm font-medium text-foreground truncate">{svc.name}</p>
              <p className="text-xs text-muted-foreground truncate mt-0.5">{svc.check_target}</p>
            </div>
            <div className="flex items-center gap-3 ml-4 shrink-0">
              {svc.latest_health_check?.response_time_ms != null && (
                <span className="text-xs tabular-nums text-muted-foreground hidden sm:inline">
                  {svc.latest_health_check.response_time_ms}ms
                </span>
              )}
              <StatusBadge status={svc.status} />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

// ─── Alerts ───────────────────────────────────────────────────────────────────

function NodeAlertList({ alerts }: { alerts: Alert[] }) {
  const shown = alerts.slice(0, 5);

  if (shown.length === 0) {
    return (
      <div className="flex items-center gap-2 py-1">
        <span className="h-1.5 w-1.5 rounded-full bg-primary shrink-0" />
        <p className="text-sm text-muted-foreground">No active alerts on this node.</p>
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-destructive/20 bg-destructive/20 overflow-hidden divide-y divide-destructive/10">
      {shown.map((alert) => (
        <div key={alert.id} className="flex items-start gap-3 px-4 py-3">
          <AlertTriangleIcon size={14} className="text-destructive shrink-0 mt-0.5" />
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium text-foreground">{alert.title}</p>
            <p className="text-xs text-muted-foreground mt-0.5">{alert.message}</p>
          </div>
          <div className="flex flex-col items-end gap-1 ml-2 shrink-0">
            <span className="text-xs text-muted-foreground whitespace-nowrap">
              {formatRelativeTime(alert.triggered_at)}
            </span>
            <StatusBadge status={alert.status} />
          </div>
        </div>
      ))}
    </div>
  );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────────

function PageSkeleton() {
  return (
    <div className="px-6 py-6">
      <div className="mx-auto max-w-6xl space-y-8">
        <Skeleton className="h-5 w-40" />
        <div className="rounded-xl border border-border bg-card p-6 space-y-4">
          <div className="flex items-center gap-4">
            <Skeleton className="h-12 w-12 rounded-xl" />
            <div className="space-y-2">
              <Skeleton className="h-6 w-48" />
              <Skeleton className="h-4 w-64" />
            </div>
          </div>
        </div>
        <div className="grid grid-cols-3 divide-x divide-border/60">
          {Array.from({ length: 3 }).map((_, i) => (
            <div
              key={i}
              className={cn(
                "flex flex-col items-center gap-3 py-4",
                i === 0 ? "pr-8" : i === 1 ? "px-8" : "pl-8",
              )}
            >
              <Skeleton className="h-35 w-35 rounded-full" />
              <Skeleton className="h-3.5 w-14" />
            </div>
          ))}
        </div>
        <Skeleton className="h-32 w-full rounded-lg" />
      </div>
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export function NodeDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const activeProjectId = useUIStore((s) => s.activeProjectId);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);

  const nodeId = id ? parseInt(id, 10) : 0;

  useMeta({
    title: `Node #${nodeId}`,
    description: `View details, metrics, services, and alerts for node #${nodeId}`,
  });

  const {
    data: nodeData,
    isLoading: nodeLoading,
    error: nodeError,
  } = useNode(nodeId, {
    enabled: nodeId > 0,
    refetchInterval: 15_000,
  });

  const deleteNode = useDeleteNode(activeProjectId ?? 0);

  function handleDelete() {
    deleteNode.mutate(nodeId, {
      onSuccess: () => {
        toast.success("Node deleted");
        navigate("/nodes");
      },
      onError: () => {
        toast.error("Failed to delete node. Remove all services from this node first.");
        setDeleteOpen(false);
      },
    });
  }

  const { data: servicesData, isLoading: servicesLoading } = useServices(activeProjectId ?? 0);
  const { data: alertsData, isLoading: alertsLoading } = useAlerts(activeProjectId ?? 0);

  const node: Node | undefined = nodeData?.node;
  const nodeServices = (servicesData?.services ?? []).filter((s) => s.node_id === nodeId);
  const nodeAlerts = (alertsData?.alerts ?? []).filter(
    (a) => a.node_id === nodeId && a.status === "active",
  );

  if (isNaN(nodeId) || nodeId <= 0) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <p className="text-sm text-muted-foreground">Invalid node ID.</p>
      </div>
    );
  }

  if (nodeLoading) return <PageSkeleton />;

  if (nodeError || !node) {
    return (
      <div className="px-6 py-6">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate("/nodes")}
          className="mb-4 -ml-2 gap-1.5 text-muted-foreground"
        >
          <ArrowLeftIcon size={14} />
          Nodes
        </Button>
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <UnplugIcon size={28} className="mb-3 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">
            {nodeError ? "Failed to load node." : "Node not found."}
          </p>
        </div>
      </div>
    );
  }

  const isOffline = node.current_state === "offline";
  const isOnline = node.current_state === "online";

  return (
    <div className="px-6 py-6">
      <div className="mx-auto max-w-6xl space-y-7">
        {/* Breadcrumb */}
        <nav className="flex items-center gap-1.5 text-sm text-muted-foreground">
          <Link to="/nodes" className="hover:text-foreground transition-colors">
            Nodes
          </Link>
          <ChevronRightIcon size={13} />
          <span className="text-foreground font-medium truncate">{node.name}</span>
        </nav>

        {/* ── Hero header ── */}
        <div className="rounded-xl border border-border bg-card overflow-hidden">
          <div className="px-6 py-5">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
              {/* Identity */}
              <div className="flex items-start gap-4 min-w-0">
                <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl mt-0.5 bg-muted">
                  <ServerIcon
                    size={20}
                    className={
                      isOnline
                        ? "text-primary"
                        : isOffline
                          ? "text-destructive"
                          : "text-muted-foreground"
                    }
                  />
                </div>
                <div className="min-w-0">
                  <h1 className="text-xl font-semibold tracking-tight text-foreground leading-snug">
                    {node.name}
                  </h1>
                  <p className="text-xs text-muted-foreground font-mono mt-1 truncate">
                    {node.identifier}
                  </p>
                </div>
              </div>

              {/* Status + delete */}
              <div className="flex items-center gap-2 shrink-0 sm:mt-0.5">
                <StatusBadge status={node.current_state} />
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="text-muted-foreground/40 hover:text-foreground"
                  onClick={() => setEditOpen(true)}
                >
                  <PencilIcon size={14} />
                </Button>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="text-muted-foreground/40 hover:bg-destructive hover:text-foreground"
                  onClick={() => setDeleteOpen(true)}
                >
                  <TrashIcon size={14} />
                </Button>
              </div>
            </div>

            {/* Meta row */}
            <div className="mt-4 pt-4 border-t border-border flex flex-wrap items-center gap-x-5 gap-y-2">
              <MetaItem
                label="Last seen"
                value={node.last_seen_at ? formatRelativeTime(node.last_seen_at) : "Never"}
              />
              <MetaItem label="Services" value={String(node.service_count)} />
              {node.active_alert_count > 0 ? (
                <MetaItem
                  label="Alerts"
                  value={String(node.active_alert_count)}
                  valueClass="text-destructive font-semibold"
                />
              ) : (
                <MetaItem label="Alerts" value="None" />
              )}
              {Object.entries(node.metadata ?? {}).map(([k, v]) => (
                <MetaItem key={k} label={k} value={String(v)} />
              ))}
            </div>
          </div>
        </div>

        {/* ── System metrics ── */}
        <section>
          <div className="mb-5 flex items-center justify-between">
            <h2 className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">
              System Metrics
            </h2>
            {node.latest_cpu?.observedAt && (
              <span className="text-xs text-muted-foreground tabular-nums">
                {formatRelativeTime(node.latest_cpu.observedAt)}
              </span>
            )}
          </div>

          {node.latest_cpu == null && node.latest_memory == null && node.latest_disk == null ? (
            <p className="text-sm text-muted-foreground py-2">
              {isOffline
                ? "Node is offline — metrics unavailable."
                : "Waiting for first metrics from the agent…"}
            </p>
          ) : (
            <div className="grid grid-cols-3 divide-x divide-border/60">
              <div className="flex items-center justify-center py-4 pr-8">
                <MetricGauge
                  label="CPU"
                  value={node.latest_cpu?.value ?? null}
                  icon={<CpuIcon size={13} />}
                  loading={nodeLoading}
                />
              </div>
              <div className="flex items-center justify-center py-4 px-8">
                <MetricGauge
                  label="Memory"
                  value={node.latest_memory?.value ?? null}
                  icon={<MemoryStickIcon size={13} />}
                  loading={nodeLoading}
                />
              </div>
              <div className="flex items-center justify-center py-4 pl-8">
                <MetricGauge
                  label="Disk"
                  value={node.latest_disk?.value ?? null}
                  icon={<HardDriveIcon size={13} />}
                  loading={nodeLoading}
                />
              </div>
            </div>
          )}
        </section>

        {/* ── Services ── */}
        <section>
          <SectionHeading
            icon={<ActivityIcon size={13} />}
            label="Services"
            aside={
              <span className="text-xs text-muted-foreground">
                {servicesLoading ? "…" : nodeServices.length}
              </span>
            }
            action={
              <Button
                variant="ghost"
                size="sm"
                className="h-7 px-2 text-xs"
                onClick={() => navigate("/services")}
              >
                View all
              </Button>
            }
          />
          {servicesLoading ? (
            <div className="space-y-px rounded-lg overflow-hidden border border-border">
              {Array.from({ length: 2 }).map((_, i) => (
                <div key={i} className="flex items-center gap-4 px-4 py-3 bg-card">
                  <Skeleton className="h-3 w-3 rounded-full" />
                  <div className="space-y-1.5">
                    <Skeleton className="h-3.5 w-36" />
                    <Skeleton className="h-3 w-24" />
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <NodeServiceList services={nodeServices} />
          )}
        </section>

        {/* ── Alerts ── */}
        <section>
          <SectionHeading
            icon={<WifiIcon size={13} />}
            label="Active Alerts"
            aside={
              nodeAlerts.length > 0 ? (
                <span className="text-xs font-semibold text-destructive">{nodeAlerts.length}</span>
              ) : undefined
            }
          />
          {alertsLoading ? (
            <Skeleton className="h-10 w-full rounded-lg" />
          ) : (
            <NodeAlertList alerts={nodeAlerts} />
          )}
        </section>

        {/* ── Footer ── */}
        <div className="flex flex-wrap gap-x-6 gap-y-1 text-xs text-muted-foreground border-t border-border pt-4 pb-2">
          <span>Created {formatDate(node.created_at)}</span>
          <span>Updated {formatDate(node.updated_at)}</span>
          <span className="font-mono opacity-60">#{node.id}</span>
        </div>
      </div>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete node"
        description="This will permanently delete the node and cannot be undone. The node must have no services assigned before it can be deleted."
        onConfirm={handleDelete}
        loading={deleteNode.isPending}
      />
      {activeProjectId && (
        <CreateNodeDialog
          projectId={activeProjectId}
          open={editOpen}
          onOpenChange={setEditOpen}
          node={node}
        />
      )}
    </div>
  );
}
