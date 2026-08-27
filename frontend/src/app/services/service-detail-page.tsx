import { useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import {
  AlertTriangleIcon,
  ArrowLeftIcon,
  ChevronRightIcon,
  GlobeIcon,
  LoaderCircleIcon,
  NetworkIcon,
  PencilIcon,
  SirenIcon,
  TrashIcon,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { StatusBadge } from "@/components/ui/status-badge.tsx";
import { MetaItem } from "@/components/meta-item.tsx";
import { RelativeTime } from "@/components/relative-time.tsx";
import { SectionHeading } from "@/components/section-heading.tsx";
import { ConfirmDialog } from "@/components/ui/confirm-dialog.tsx";
import { CreateServiceDialog } from "./components/create-service-dialog.tsx";
import {
  useDeleteService,
  useService,
  useServiceDetailStream,
  useServiceHistory,
  useServiceHistoryWindow,
} from "@/hooks/use-services.ts";
import { useUIStore } from "@/stores/ui-store.ts";
import { formatDate, cn } from "@/lib/utils.ts";
import type { Service, ServiceAlert } from "@/types/service.ts";
import { useMeta } from "@/hooks/use-meta.ts";
import {
  HistoryLoadingState,
  LatencyChart,
  RecentChecksPagination,
  RecentChecksHeading,
  RecentChecksList,
  RecentCheckStrip,
  ServiceHealthLoadingState,
  ServiceHistoryEmptyState,
  ServiceHistoryRefreshError,
  ServiceHistorySummaryMetrics,
  ServiceHistorySuccessCount,
} from "./components/service-history.tsx";
import {
  deduplicateHistory,
  SERVICE_HISTORY_RANGES,
  type ServiceHistoryRangeHours,
} from "./service-history-utils.ts";

// ─── Helpers ──────────────────────────────────────────────────────────────────

function severityClass(severity: ServiceAlert["severity"]) {
  switch (severity) {
    case "critical":
      return "text-destructive border-destructive/25 bg-destructive/10";
    case "warning":
      return "text-warning border-warning/25 bg-warning/10";
    default:
      return "text-muted-foreground border-border bg-muted/30";
  }
}

function stateLabel(service: Service): string {
  const latest = service.latest_health_check;
  if (!latest) return "Unknown";
  if (latest.is_success) return "Healthy";

  const msg = (latest.message ?? "").toLowerCase();
  if (msg.includes("connection refused")) return "Unreachable";
  if (msg.includes("timeout") || msg.includes("timed out")) return "Timed Out";
  if (msg.includes("no such host") || msg.includes("name resolution")) return "DNS Failure";
  return "Failing";
}

function executionModeLabel(mode: Service["execution_mode"]) {
  return mode === "agent" ? "Project node" : "This instance";
}

// ─── Skeleton ─────────────────────────────────────────────────────────────────

function PageSkeleton() {
  return (
    <div className="px-6 py-6">
      <div className="mx-auto max-w-6xl space-y-7">
        <Skeleton className="h-5 w-48" />
        <div className="rounded-xl border border-border bg-card p-6 space-y-4">
          <div className="flex items-center gap-4">
            <Skeleton className="h-11 w-11 rounded-xl" />
            <div className="space-y-2">
              <Skeleton className="h-5 w-48" />
              <Skeleton className="h-3.5 w-64" />
            </div>
          </div>
          <div className="pt-4 border-t border-border flex gap-5">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-3.5 w-20" />
            ))}
          </div>
        </div>
        <div className="space-y-4">
          <div className="flex gap-2">
            {Array.from({ length: 12 }).map((_, i) => (
              <Skeleton key={i} className="h-4 w-4 rounded-full" />
            ))}
          </div>
          <Skeleton className="h-10 w-36" />
          <Skeleton className="h-4 w-48" />
        </div>
        <Skeleton className="h-24 w-full rounded-lg" />
      </div>
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export function ServiceDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const activeProjectId = useUIStore((s) => s.activeProjectId);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [rangeHours, setRangeHours] = useState<ServiceHistoryRangeHours>(24);
  const serviceId = id ? parseInt(id, 10) : 0;

  const { data, isLoading, error } = useService(serviceId, { refetchInterval: false });
  const historyWindow = useServiceHistoryWindow(serviceId, rangeHours);
  const historyList = useServiceHistory(serviceId);
  useServiceDetailStream(serviceId, { enabled: serviceId > 0 });
  const service = data?.service;
  const deleteService = useDeleteService(activeProjectId ?? 0);
  const rangeObservations = historyWindow.data?.observations ?? [];
  const historyObservations = useMemo(
    () => deduplicateHistory(historyList.data?.pages.flatMap((page) => page.observations) ?? []),
    [historyList.data],
  );
  const summary = historyWindow.data?.summary;

  useMeta({
    title: service ? `${service.name} Details` : "Service Details",
    description: service
      ? `View recent health checks, active alerts, and configuration for ${service.name}`
      : "View service details",
  });

  function handleDelete() {
    deleteService.mutate(serviceId, {
      onSuccess: () => {
        toast.success("Service deleted");
        void navigate("/services");
      },
      onError: () => {
        toast.error("Failed to delete service.");
        setDeleteOpen(false);
      },
    });
  }

  if (isNaN(serviceId) || serviceId <= 0) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <p className="text-sm text-muted-foreground">Invalid service ID.</p>
      </div>
    );
  }

  if (isLoading) return <PageSkeleton />;

  if (error || !service) {
    return (
      <div className="px-6 py-6">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => void navigate("/services")}
          className="mb-4 -ml-2 gap-1.5 text-muted-foreground"
        >
          <ArrowLeftIcon size={14} />
          Services
        </Button>
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <AlertTriangleIcon size={28} className="mb-3 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">
            {error ? "Failed to load service." : "Service not found."}
          </p>
        </div>
      </div>
    );
  }

  const latest = service.latest_health_check;
  const alerts = service.active_alerts ?? [];
  const isHealthy = latest?.is_success ?? service.status === "healthy";
  const isUnhealthy = latest ? !latest.is_success : service.status === "unhealthy";
  const isDegraded = latest
    ? !latest.is_success && service.status === "degraded"
    : service.status === "degraded";
  const hasAgentBackedNode = service.execution_mode === "agent" && service.node_id > 0;

  return (
    <div className="px-6 py-6">
      <div className="mx-auto max-w-6xl space-y-7">
        {/* Breadcrumb */}
        <nav className="flex items-center gap-1.5 text-sm text-muted-foreground">
          <Link to="/services" className="hover:text-foreground transition-colors">
            Services
          </Link>
          <ChevronRightIcon size={13} />
          <span className="text-foreground font-medium truncate">{service.name}</span>
        </nav>

        {/* ── 1. Header card ── */}
        <div className="rounded-xl border border-border bg-card overflow-hidden">
          <div className="px-6 py-5">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
              <div className="flex items-start gap-4 min-w-0">
                <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl mt-0.5 bg-muted">
                  <GlobeIcon
                    size={20}
                    className={
                      isUnhealthy
                        ? "text-destructive"
                        : isDegraded
                          ? "text-warning"
                          : isHealthy
                            ? "text-primary"
                            : "text-muted-foreground"
                    }
                  />
                </div>
                <div className="min-w-0">
                  <h1 className="text-xl font-semibold tracking-tight text-foreground leading-snug">
                    {service.name}
                  </h1>
                  <p className="mt-1 text-xs text-muted-foreground font-mono truncate">
                    {service.check_target}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-2 shrink-0 sm:mt-0.5">
                <StatusBadge status={service.status} />
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

            {/* Meta row — mirrors Node page header */}
            <div className="mt-4 pt-4 border-t border-border flex flex-wrap items-center gap-x-5 gap-y-2">
              <MetaItem
                label="Last checked"
                value={<RelativeTime value={service.last_checked_at} fallback="Never" />}
              />
              <MetaItem label="Type" value={service.check_type.toUpperCase()} />
              {service.consecutive_failures > 0 ? (
                <MetaItem
                  label="Failures"
                  value={String(service.consecutive_failures)}
                  valueClass="text-destructive font-semibold"
                />
              ) : (
                <MetaItem label="Failures" value="None" />
              )}
              {alerts.length > 0 ? (
                <MetaItem
                  label="Alerts"
                  value={String(alerts.length)}
                  valueClass="text-destructive font-semibold"
                />
              ) : (
                <MetaItem label="Alerts" value="None" />
              )}
            </div>
          </div>
        </div>

        {/* ── 2. Service health — visual section ── */}
        <section>
          <div className="mb-5 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <h2 className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">
                Service Health
              </h2>
              <span className="inline-flex size-3.5 items-center justify-center" aria-live="polite">
                {historyWindow.isFetching && !historyWindow.isPending ? (
                  <LoaderCircleIcon
                    className="size-3 animate-spin text-muted-foreground"
                    aria-label={`Loading ${rangeHours === 168 ? "7D" : `${rangeHours}H`} history`}
                  />
                ) : null}
              </span>
            </div>
            <div
              className="inline-flex rounded-md border border-border bg-card p-0.5"
              aria-label="History time range"
            >
              {SERVICE_HISTORY_RANGES.map((range) => (
                <button
                  key={range.label}
                  type="button"
                  aria-pressed={rangeHours === range.hours}
                  onClick={() => setRangeHours(range.hours)}
                  className={cn(
                    "rounded px-2.5 py-1 text-[11px] font-semibold transition-colors",
                    rangeHours === range.hours
                      ? "bg-muted text-foreground"
                      : "text-muted-foreground hover:text-foreground",
                  )}
                >
                  {range.label}
                </button>
              ))}
            </div>
          </div>

          {historyWindow.isPending ? (
            <ServiceHealthLoadingState />
          ) : historyWindow.isError && !historyWindow.data ? (
            <div className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-4">
              <p className="text-sm text-destructive">Failed to load service history.</p>
              <Button
                variant="ghost"
                size="sm"
                className="mt-2 -ml-3 h-7 text-xs"
                onClick={() => void historyWindow.refetch()}
              >
                Try again
              </Button>
            </div>
          ) : summary?.totalChecks === 0 ? (
            <ServiceHistoryEmptyState
              refreshError={historyWindow.isError}
              onRetry={() => void historyWindow.refetch()}
            />
          ) : (
            <div className="space-y-5">
              {historyWindow.isError ? (
                <ServiceHistoryRefreshError onRetry={() => void historyWindow.refetch()} />
              ) : null}
              <div className="flex flex-col gap-5 md:flex-row md:items-start md:justify-between">
                <div className="space-y-1">
                  <p
                    className={cn(
                      "text-4xl font-bold leading-none tracking-tight font-heading",
                      isHealthy ? "text-primary" : "text-destructive",
                    )}
                  >
                    {stateLabel(service)}
                  </p>
                  {summary ? <ServiceHistorySuccessCount summary={summary} /> : null}
                </div>
                <div className="min-w-0 space-y-2 md:max-w-[55%]">
                  <p className="text-right text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/50">
                    Recent Checks
                  </p>
                  <RecentCheckStrip observations={rangeObservations} rangeHours={rangeHours} />
                </div>
              </div>

              {summary ? (
                <ServiceHistorySummaryMetrics
                  summary={summary}
                  liveLastCheckedAt={service.last_checked_at}
                />
              ) : null}

              <LatencyChart observations={rangeObservations} rangeHours={rangeHours} />
              {historyWindow.data?.isTruncated ? (
                <p className="text-xs text-muted-foreground">
                  Chart shows the latest 2,000 observations; metrics include the full range.
                </p>
              ) : null}
            </div>
          )}
        </section>

        {/* ── 3. Recent checks — dense list section ── */}
        <section>
          <RecentChecksHeading />
          {historyList.isPending ? (
            <HistoryLoadingState rows={10} />
          ) : historyList.isError && !historyList.data ? (
            <div className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-4">
              <p className="text-sm text-destructive">Failed to load recent checks.</p>
              <Button
                variant="ghost"
                size="sm"
                className="mt-2 -ml-3 h-7 text-xs"
                onClick={() => void historyList.refetch()}
              >
                Try again
              </Button>
            </div>
          ) : historyObservations.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border px-4 py-5 text-center">
              <p className="text-sm text-muted-foreground">No check history yet.</p>
            </div>
          ) : (
            <div className="space-y-3">
              <RecentChecksList observations={historyObservations} />
              <RecentChecksPagination
                hasNextPage={historyList.hasNextPage}
                isFetchingNextPage={historyList.isFetchingNextPage}
                isFetchNextPageError={historyList.isFetchNextPageError}
                onLoadMore={() => void historyList.fetchNextPage()}
              />
            </div>
          )}
        </section>

        {/* ── 4. Active alerts ── */}
        <section>
          <SectionHeading
            icon={<SirenIcon size={13} />}
            label="Active Alerts"
            aside={
              alerts.length > 0 ? (
                <span className="text-xs font-semibold text-destructive">{alerts.length}</span>
              ) : undefined
            }
          />
          {alerts.length === 0 ? (
            <div className="flex items-center gap-2 py-1">
              <span className="h-1.5 w-1.5 rounded-full bg-primary shrink-0" />
              <p className="text-sm text-muted-foreground">No active alerts on this service.</p>
            </div>
          ) : (
            <div className="rounded-lg border border-destructive/20 bg-destructive/5 overflow-hidden divide-y divide-destructive/10">
              {alerts.map((alert) => (
                <div key={alert.id} className="flex items-start gap-3 px-4 py-3">
                  <AlertTriangleIcon size={14} className="text-destructive shrink-0 mt-0.5" />
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium text-foreground">{alert.title}</p>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      <RelativeTime value={alert.triggered_at} prefix="Triggered" />
                    </p>
                  </div>
                  <div className="flex flex-col items-end gap-1 ml-2 shrink-0">
                    <span className="text-xs text-muted-foreground whitespace-nowrap">
                      <RelativeTime value={alert.triggered_at} />
                    </span>
                    <div className="flex items-center gap-1.5">
                      <span
                        className={cn(
                          "rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide",
                          severityClass(alert.severity),
                        )}
                      >
                        {alert.severity}
                      </span>
                      <StatusBadge status={alert.status} />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </section>

        {/* ── 5. Service configuration — low emphasis ── */}
        <section>
          <SectionHeading
            icon={<NetworkIcon size={13} />}
            label="Configuration"
            action={
              hasAgentBackedNode ? (
                <Link
                  to={`/nodes/${service.node_id}`}
                  className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
                >
                  View node
                  <ChevronRightIcon size={12} />
                </Link>
              ) : undefined
            }
          />
          <div className="flex flex-wrap gap-x-6 gap-y-2">
            <MetaItem label="Check type" value={service.check_type.toUpperCase()} />
            <MetaItem label="Runs from" value={executionModeLabel(service.execution_mode)} />
            {hasAgentBackedNode ? <MetaItem label="Node" value={`#${service.node_id}`} /> : null}
            <MetaItem label="Service ID" value={String(service.id)} />
            <MetaItem label="Updated" value={formatDate(service.updated_at)} />
          </div>
        </section>

        {/* ── Footer ── */}
        <div className="flex flex-wrap gap-x-6 gap-y-1 text-xs text-muted-foreground border-t border-border pt-4 pb-2">
          <span>Created {formatDate(service.created_at)}</span>
          <span>Updated {formatDate(service.updated_at)}</span>
          <span className="font-mono opacity-60">#{service.id}</span>
        </div>
      </div>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete service"
        description="This will permanently delete the service and all its check history. This action cannot be undone."
        onConfirm={handleDelete}
        loading={deleteService.isPending}
      />
      {activeProjectId && (
        <CreateServiceDialog
          projectId={activeProjectId}
          open={editOpen}
          onOpenChange={setEditOpen}
          onRequestNodeSetup={() => {}}
          service={service}
        />
      )}
    </div>
  );
}
