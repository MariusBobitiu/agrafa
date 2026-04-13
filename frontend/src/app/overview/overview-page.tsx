import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import {
  AlertTriangleIcon,
  ArrowRightIcon,
  CheckCircle2Icon,
  CircleIcon,
  CpuIcon,
  HardDriveIcon,
  InfoIcon,
  MemoryStickIcon,
  ServerIcon,
  TriangleAlertIcon,
  XCircleIcon,
  ZapIcon,
} from "lucide-react";
import {
  ActivityIcon,
  BellIcon,
  UnplugIcon,
  WifiIcon,
} from "@/components/animate-ui/icons/index.ts";
import { Button } from "@/components/ui/button.tsx";
import { MetricBar } from "@/components/metric-bar";
import { PageHeader } from "@/components/ui/page-header.tsx";
import { StatCard } from "@/components/ui/stat-card.tsx";
import { StatusBadge } from "@/components/ui/status-badge.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { overviewApi } from "@/data/overview.ts";
import { servicesApi } from "@/data/services.ts";
import { useAlertRules } from "@/hooks/use-alerts.ts";
import { useUIStore } from "@/stores/ui-store.ts";
import { formatRelativeTime } from "@/lib/utils.ts";
import { cn } from "@/lib/utils.ts";
import type { Overview, NodeSummary, OverviewEvent } from "@/types/overview.ts";
import type { Service } from "@/types/service.ts";
import { useMeta } from "@/hooks/use-meta";

// ─── helpers ──────────────────────────────────────────────────────────────────

type NodeStateStatus = "online" | "offline" | "unknown";

function statusDotClass(status: NodeStateStatus | "healthy" | "degraded" | "unhealthy") {
  return cn("rounded-full shrink-0", {
    "bg-primary": status === "online" || status === "healthy",
    "bg-warning": status === "degraded",
    "bg-destructive": status === "offline" || status === "unhealthy",
    "bg-muted-foreground/40": status === "unknown",
  });
}

function severityConfig(severity: string) {
  switch (severity) {
    case "critical":
    case "error":
      return {
        icon: <XCircleIcon size={13} className="text-destructive shrink-0 mt-0.5" />,
        dotClass: "bg-destructive",
      };
    case "warning":
      return {
        icon: <TriangleAlertIcon size={13} className="text-warning shrink-0 mt-0.5" />,
        dotClass: "bg-warning",
      };
    default:
      return {
        icon: <InfoIcon size={13} className="text-blue-400 shrink-0 mt-0.5" />,
        dotClass: "bg-blue-400",
      };
  }
}

// ─── Status banner ────────────────────────────────────────────────────────────

type BannerProps = {
  data: Overview;
  services: Service[];
};

function SystemStatusBanner({ data, services }: BannerProps) {
  const navigate = useNavigate();
  const offlineNodes = data.node_summaries.filter((n) => n.current_state === "offline");
  const unhealthyServices = services.filter(
    (s) => s.status === "unhealthy" || s.status === "degraded",
  );
  const issueCount =
    offlineNodes.length + unhealthyServices.length + (data.active_alerts > 0 ? 1 : 0);
  const hasIssues = issueCount > 0;

  if (!hasIssues) {
    return (
      <div className="flex items-center gap-4 rounded-xl border border-primary/20 bg-primary/5 px-5 py-4">
        <div className="flex h-9 w-9 items-center justify-center rounded-full bg-primary/10 shrink-0">
          <CheckCircle2Icon size={18} className="text-primary" />
        </div>
        <div>
          <p className="text-sm font-semibold text-lime-600 dark:text-lime-400">
            All systems operational
          </p>
          <p className="text-xs text-muted-foreground mt-0.5">
            {data.nodes_online} node{data.nodes_online !== 1 ? "s" : ""} online
            {data.total_services > 0
              ? ` · ${data.services_healthy} service${data.services_healthy !== 1 ? "s" : ""} healthy`
              : ""}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-destructive/20 bg-destructive/5 overflow-hidden">
      <div className="flex items-center gap-4 px-5 py-4 border-b border-destructive/10">
        <div className="flex h-9 w-9 items-center justify-center rounded-full bg-destructive/10 shrink-0">
          <AlertTriangleIcon size={18} className="text-destructive" />
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-semibold text-foreground">
            {issueCount} issue{issueCount > 1 ? "s" : ""} require attention
          </p>
          <p className="text-xs text-muted-foreground mt-0.5">
            {[
              offlineNodes.length > 0 &&
                `${offlineNodes.length} node${offlineNodes.length > 1 ? "s" : ""} offline`,
              unhealthyServices.length > 0 &&
                `${unhealthyServices.length} service${unhealthyServices.length > 1 ? "s" : ""} degraded`,
              data.active_alerts > 0 &&
                `${data.active_alerts} active alert${data.active_alerts > 1 ? "s" : ""}`,
            ]
              .filter(Boolean)
              .join(" · ")}
          </p>
        </div>
      </div>
      <div className="divide-y divide-destructive/10">
        {offlineNodes.map((node) => (
          <BannerIssueRow
            key={`node-${node.id}`}
            label={node.name}
            detail="Node offline"
            severity="critical"
            href={`/nodes/${node.id}`}
          />
        ))}
        {unhealthyServices.map((svc) => (
          <BannerIssueRow
            key={`svc-${svc.id}`}
            label={svc.name}
            detail={svc.status === "degraded" ? "Service degraded" : "Service unhealthy"}
            severity={svc.status === "degraded" ? "warning" : "critical"}
          />
        ))}
        {data.active_alerts > 0 && (
          <BannerIssueRow
            label={`${data.active_alerts} active alert${data.active_alerts > 1 ? "s" : ""}`}
            detail="Firing right now"
            severity="critical"
            href="/alerts"
            onClick={() => navigate("/alerts")}
          />
        )}
      </div>
    </div>
  );
}

type BannerIssueRowProps = {
  label: string;
  detail: string;
  severity: "critical" | "warning";
  href?: string;
  onClick?: () => void;
};

function BannerIssueRow({ label, detail, severity, href, onClick }: BannerIssueRowProps) {
  const navigate = useNavigate();
  return (
    <button
      type="button"
      onClick={onClick ?? (href ? () => navigate(href) : undefined)}
      className={cn(
        "w-full flex items-center justify-between px-5 py-2.5 text-left text-sm transition-colors",
        href ? "cursor-pointer hover:bg-destructive/5" : "cursor-default",
      )}
    >
      <div className="flex items-center gap-2.5">
        <CircleIcon
          size={7}
          className={cn(
            "fill-current shrink-0",
            severity === "critical" ? "text-destructive" : "text-warning",
          )}
        />
        <span className="font-medium text-foreground">{label}</span>
      </div>
      <div className="flex items-center gap-1.5">
        <span className="text-xs text-muted-foreground">{detail}</span>
        {href && <ArrowRightIcon size={11} className="text-muted-foreground" />}
      </div>
    </button>
  );
}

// ─── Node cards (machine cards) ───────────────────────────────────────────────

function NodeMachineCard({ node }: { node: NodeSummary }) {
  const navigate = useNavigate();
  const cpu = node.latest_cpu?.value ?? null;
  const mem = node.latest_memory?.value ?? null;
  const disk = node.latest_disk?.value ?? null;
  const hasMetrics = cpu !== null || mem !== null || disk !== null;
  const isOnline = node.current_state === "online";
  const isOffline = node.current_state === "offline";

  return (
    <div
      className={cn(
        "rounded-xl border bg-card p-5 cursor-pointer transition-colors hover:bg-muted/20",
        isOffline ? "border-destructive/30 bg-destructive/2" : "border-border",
      )}
      onClick={() => navigate(`/nodes/${node.id}`)}
    >
      {/* ── Header ── */}
      <div className="flex items-start justify-between gap-3 mb-4">
        <div className="flex items-start gap-3 min-w-0">
          <div
            className={cn(
              "mt-0.5 flex h-8 w-8 items-center justify-center rounded-lg shrink-0",
              isOnline ? "bg-primary/10" : isOffline ? "bg-destructive/10" : "bg-muted",
            )}
          >
            <ServerIcon
              size={15}
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
            <p className="text-sm font-semibold text-foreground leading-snug truncate">
              {node.name}
            </p>
            <p className="text-xs text-muted-foreground mt-0.5">
              {node.last_seen_at
                ? isOnline
                  ? `Active · ${formatRelativeTime(node.last_seen_at)}`
                  : `Last seen ${formatRelativeTime(node.last_seen_at)}`
                : "Never seen"}
            </p>
          </div>
        </div>
        <StatusBadge status={node.current_state} />
      </div>

      {/* ── Metrics ── */}
      {hasMetrics ? (
        <div className="space-y-2.5 mb-4">
          {cpu !== null && (
            <MetricBar icon={<CpuIcon size={12} />} label="CPU" value={cpu} variant="cpu" />
          )}
          {mem !== null && (
            <MetricBar icon={<MemoryStickIcon size={12} />} label="Mem" value={mem} variant="mem" />
          )}
          {disk !== null && (
            <MetricBar
              icon={<HardDriveIcon size={12} />}
              label="Disk"
              value={disk}
              variant="disk"
            />
          )}
        </div>
      ) : (
        <div className="mb-4 rounded-lg bg-muted/40 px-3 py-2.5">
          <p className="text-xs text-muted-foreground">
            {isOnline ? "Waiting for first metrics…" : "No metrics available"}
          </p>
        </div>
      )}

      {/* ── Footer ── */}
      <div className="flex items-center gap-3 pt-3 border-t border-border">
        <span className="text-xs text-muted-foreground">
          {node.service_count} service{node.service_count !== 1 ? "s" : ""}
        </span>
        {node.active_alert_count > 0 && (
          <>
            <span className="text-muted-foreground/30">·</span>
            <span className="text-xs font-medium text-destructive">
              {node.active_alert_count} alert{node.active_alert_count > 1 ? "s" : ""}
            </span>
          </>
        )}
        <ArrowRightIcon size={11} className="text-muted-foreground/40 ml-auto" />
      </div>
    </div>
  );
}

function OverviewNodeGrid({ nodes }: { nodes: NodeSummary[] }) {
  const navigate = useNavigate();
  const shown = nodes.slice(0, 6);

  return (
    <section>
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h2 className="text-sm font-semibold text-foreground">Infrastructure</h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            {nodes.length} node{nodes.length !== 1 ? "s" : ""} registered
          </p>
        </div>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 px-2 text-xs"
          onClick={() => navigate("/nodes")}
        >
          View all
          <ArrowRightIcon size={11} />
        </Button>
      </div>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {shown.map((node) => (
          <NodeMachineCard key={node.id} node={node} />
        ))}
      </div>
      {nodes.length > 6 && (
        <button
          type="button"
          onClick={() => navigate("/nodes")}
          className="mt-3 w-full rounded-lg border border-dashed border-border py-2.5 text-xs text-muted-foreground hover:text-foreground hover:border-border/70 transition-colors text-center"
        >
          +{nodes.length - 6} more nodes — view all
        </button>
      )}
    </section>
  );
}

// ─── Services (compact secondary) ────────────────────────────────────────────

function OverviewServiceList({ services }: { services: Service[] }) {
  const navigate = useNavigate();
  const shown = services.slice(0, 8);
  const unhealthyCount = services.filter(
    (s) => s.status === "unhealthy" || s.status === "degraded",
  ).length;

  return (
    <section>
      <div className="mb-3 flex items-center justify-between">
        <div>
          <h2 className="text-sm font-semibold text-foreground">Services</h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            {services.length} monitored
            {unhealthyCount > 0 && (
              <span className="text-warning ml-1">· {unhealthyCount} with issues</span>
            )}
          </p>
        </div>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 px-2 text-xs"
          onClick={() => navigate("/services")}
        >
          View all
          <ArrowRightIcon size={11} />
        </Button>
      </div>
      <div className="rounded-xl border border-border bg-card divide-y divide-border overflow-hidden">
        {shown.map((svc) => (
          <div
            key={svc.id}
            className="flex items-center justify-between px-4 py-2.5 hover:bg-muted/20 transition-colors"
          >
            <div className="flex items-center gap-2.5 min-w-0">
              <span
                className={cn(
                  statusDotClass(svc.status as Parameters<typeof statusDotClass>[0]),
                  "h-1.5 w-1.5",
                )}
              />
              <div className="min-w-0">
                <p className="text-sm font-medium text-foreground truncate leading-snug">
                  {svc.name}
                </p>
                <p className="text-xs text-muted-foreground truncate">{svc.check_target}</p>
              </div>
            </div>
            <div className="flex items-center gap-3 shrink-0 ml-3">
              {svc.latest_health_check?.response_time_ms != null && (
                <span className="text-xs tabular-nums text-muted-foreground">
                  {svc.latest_health_check.response_time_ms}ms
                </span>
              )}
              <StatusBadge status={svc.status} />
            </div>
          </div>
        ))}
        {services.length > 8 && (
          <button
            type="button"
            onClick={() => navigate("/services")}
            className="w-full px-4 py-2.5 text-xs text-muted-foreground hover:text-foreground hover:bg-muted/20 transition-colors text-center"
          >
            +{services.length - 8} more — view all
          </button>
        )}
      </div>
    </section>
  );
}

// ─── Recent events ────────────────────────────────────────────────────────────

function OverviewRecentEvents({ events }: { events: OverviewEvent[] }) {
  if (events.length === 0) return null;

  const shown = events.slice(0, 6);

  return (
    <section>
      <div className="mb-3 flex items-center justify-between">
        <h2 className="text-sm font-semibold text-foreground">Recent Events</h2>
      </div>
      <div className="rounded-xl border border-border bg-card divide-y divide-border overflow-hidden">
        {shown.map((event) => {
          const { icon } = severityConfig(event.severity);
          return (
            <div key={event.id} className="flex items-start gap-3 px-4 py-3">
              {icon}
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium text-foreground leading-snug">{event.title}</p>
                <p className="text-xs text-muted-foreground mt-0.5">
                  {event.event_type.replace(/_/g, " ")}
                </p>
              </div>
              <span className="text-xs text-muted-foreground shrink-0 tabular-nums mt-0.5">
                {formatRelativeTime(event.occurred_at)}
              </span>
            </div>
          );
        })}
      </div>
    </section>
  );
}

// ─── Alert setup prompt ───────────────────────────────────────────────────────

function AlertSetupPrompt() {
  const navigate = useNavigate();
  return (
    <div className="flex items-start gap-4 rounded-xl border border-border bg-card px-5 py-4">
      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-muted mt-0.5">
        <BellIcon size={15} className="text-muted-foreground" />
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium text-foreground">Alerts are not configured yet</p>
        <p className="mt-0.5 text-xs text-muted-foreground">
          Events are being recorded, but notifications won't fire until you create alert rules.
        </p>
      </div>
      <div className="flex items-center gap-2 shrink-0">
        <Button
          variant="ghost"
          size="sm"
          className="h-7 px-2 text-xs"
          onClick={() => navigate("/alerts")}
        >
          View alerts
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="h-7 px-3 text-xs"
          onClick={() => navigate("/alerts?create=1")}
        >
          Create alert rule
        </Button>
      </div>
    </div>
  );
}

// ─── Empty state ──────────────────────────────────────────────────────────────

function OverviewEmptyState() {
  const navigate = useNavigate();
  return (
    <div className="rounded-xl border border-border bg-card px-8 py-12 text-center">
      <div className="mx-auto mb-5 flex h-12 w-12 items-center justify-center rounded-full bg-muted">
        <ZapIcon size={20} className="text-muted-foreground" />
      </div>
      <h3 className="text-base font-semibold text-foreground">Start monitoring</h3>
      <p className="mt-2 text-sm text-muted-foreground max-w-sm mx-auto">
        Add a server to run checks from infrastructure you control, or start monitoring a service
        endpoint right away.
      </p>
      <div className="mt-6 flex flex-col sm:flex-row items-center justify-center gap-3">
        <Button
          variant="default"
          size="sm"
          onClick={() => navigate("/nodes?setup=1")}
          className="gap-2 min-w-40"
        >
          <ServerIcon size={14} />
          Add a server
        </Button>
        <Button
          variant="primary"
          size="sm"
          onClick={() => navigate("/services?create=1")}
          className="gap-2 min-w-40"
        >
          <ActivityIcon size={14} />
          Monitor a service
        </Button>
      </div>
    </div>
  );
}

// ─── Loading skeletons ────────────────────────────────────────────────────────

function NodeCardSkeleton() {
  return (
    <div className="rounded-xl border border-border bg-card p-5">
      <div className="flex items-start justify-between gap-3 mb-4">
        <div className="flex items-start gap-3">
          <Skeleton className="h-8 w-8 rounded-lg" />
          <div className="space-y-1.5">
            <Skeleton className="h-3.5 w-32" />
            <Skeleton className="h-3 w-24" />
          </div>
        </div>
        <Skeleton className="h-5 w-16 rounded-full" />
      </div>
      <div className="space-y-2.5 mb-4">
        <Skeleton className="h-3 w-full" />
        <Skeleton className="h-3 w-full" />
        <Skeleton className="h-3 w-full" />
      </div>
      <div className="pt-3 border-t border-border flex gap-3">
        <Skeleton className="h-3 w-16" />
      </div>
    </div>
  );
}

function ServiceRowSkeleton({ rows = 4 }: { rows?: number }) {
  return (
    <div className="rounded-xl border border-border bg-card divide-y divide-border overflow-hidden">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex items-center justify-between px-4 py-2.5">
          <div className="flex items-center gap-2.5">
            <Skeleton className="h-1.5 w-1.5 rounded-full" />
            <div className="space-y-1.5">
              <Skeleton className="h-3.5 w-28" />
              <Skeleton className="h-3 w-20" />
            </div>
          </div>
          <Skeleton className="h-5 w-16 rounded-full" />
        </div>
      ))}
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export function OverviewPage() {
  useMeta({
    title: "Overview",
    description: "Live infrastructure status",
  });
  const activeProjectId = useUIStore((s) => s.activeProjectId);
  const navigate = useNavigate();

  const {
    data,
    isLoading: overviewLoading,
    error,
  } = useQuery({
    queryKey: ["overview", activeProjectId],
    queryFn: () => overviewApi.get(activeProjectId!),
    enabled: !!activeProjectId,
    refetchInterval: 15_000,
  });

  const { data: servicesData, isLoading: servicesLoading } = useQuery({
    queryKey: ["services", activeProjectId],
    queryFn: () => servicesApi.list(activeProjectId!),
    enabled: !!activeProjectId,
    refetchInterval: 15_000,
  });

  const { data: alertRulesData } = useAlertRules(activeProjectId ?? 0);

  const isLoading = overviewLoading || servicesLoading;
  const services = servicesData?.services ?? [];
  const nodes = data?.node_summaries ?? [];
  const events = data?.recent_events ?? [];
  const isEmpty = !isLoading && nodes?.length === 0 && services?.length === 0;
  const noAlertRules = alertRulesData !== undefined && alertRulesData.alert_rules?.length === 0;

  if (!activeProjectId) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <UnplugIcon size={32} className="mx-auto mb-3 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">Select a project to view your overview.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="px-6 py-6">
      <div className="mx-auto max-w-7xl space-y-6">
        <PageHeader
          title="Overview"
          description="Live infrastructure status"
          actions={
            !isEmpty && !isLoading ? (
              <Button variant="outline" size="sm" onClick={() => navigate("/alerts")}>
                <BellIcon size={14} />
                Alerts
                {(data?.active_alerts ?? 0) > 0 && (
                  <span className="ml-1 rounded-full bg-destructive/15 px-1.5 py-0.5 text-xs font-medium text-destructive">
                    {data?.active_alerts}
                  </span>
                )}
              </Button>
            ) : undefined
          }
        />

        {error && <p className="text-sm text-destructive">Failed to load overview data.</p>}

        {/* ── 1. Stat cards ── */}
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <StatCard
            title="Total Nodes"
            value={data?.total_nodes ?? 0}
            icon={UnplugIcon}
            loading={isLoading}
          />
          <StatCard
            title="Online"
            value={data?.nodes_online ?? 0}
            icon={WifiIcon}
            description={
              (data?.nodes_offline ?? 0) > 0 ? `${data?.nodes_offline} offline` : undefined
            }
            loading={isLoading}
          />
          <StatCard
            title="Services"
            value={data?.total_services ?? 0}
            icon={ActivityIcon}
            description={
              (data?.services_unhealthy ?? 0) + (data?.services_degraded ?? 0) > 0
                ? `${(data?.services_unhealthy ?? 0) + (data?.services_degraded ?? 0)} issues`
                : undefined
            }
            loading={isLoading}
          />
          <StatCard
            title="Active Alerts"
            value={data?.active_alerts ?? 0}
            icon={BellIcon}
            loading={isLoading}
          />
        </div>

        {/* ── 2. Empty state ── */}
        {isEmpty && <OverviewEmptyState />}

        {!isEmpty && (
          <>
            {/* ── 3. System status banner ── */}
            {isLoading ? (
              <Skeleton className="h-16 w-full rounded-xl" />
            ) : data ? (
              <SystemStatusBanner data={data} services={services} />
            ) : null}

            {/* ── 4. Alert setup prompt ── */}
            {!isEmpty && noAlertRules && <AlertSetupPrompt />}

            {/* ── 5. Infrastructure — full width, primary ── */}
            {overviewLoading ? (
              <section>
                <div className="mb-4 flex items-center justify-between">
                  <Skeleton className="h-4 w-28" />
                </div>
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
                  <NodeCardSkeleton />
                  <NodeCardSkeleton />
                  <NodeCardSkeleton />
                </div>
              </section>
            ) : nodes?.length > 0 ? (
              <OverviewNodeGrid nodes={nodes} />
            ) : (
              <section>
                <h2 className="mb-3 text-sm font-semibold text-foreground">Infrastructure</h2>
                <div className="rounded-xl border border-dashed border-border bg-card/50 px-4 py-8 text-center">
                  <p className="text-sm text-muted-foreground mb-3">No nodes registered yet</p>
                  <Button variant="outline" size="sm" onClick={() => navigate("/nodes?setup=1")}>
                    Add a server
                  </Button>
                </div>
              </section>
            )}

            {/* ── 5. Services — secondary, below nodes ── */}
            {servicesLoading ? (
              <section>
                <div className="mb-3 flex items-center justify-between">
                  <Skeleton className="h-4 w-16" />
                </div>
                <ServiceRowSkeleton rows={4} />
              </section>
            ) : services?.length > 0 ? (
              <OverviewServiceList services={services} />
            ) : (
              <section>
                <h2 className="mb-3 text-sm font-semibold text-foreground">Services</h2>
                <div className="rounded-xl border border-dashed border-border bg-card/50 px-4 py-8 text-center">
                  <p className="text-sm text-muted-foreground mb-3">No services monitored yet</p>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => navigate("/services?create=1")}
                  >
                    Monitor a service
                  </Button>
                </div>
              </section>
            )}

            {/* ── 6. Recent events ── */}
            {!overviewLoading && events?.length > 0 && <OverviewRecentEvents events={events} />}
          </>
        )}
      </div>
    </div>
  );
}
