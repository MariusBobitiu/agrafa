import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import {
  AlertTriangleIcon,
  ArrowLeftIcon,
  CheckCircle2Icon,
  ChevronRightIcon,
  ClockIcon,
  GlobeIcon,
  NetworkIcon,
  PencilIcon,
  SirenIcon,
  TrashIcon,
  XCircleIcon,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { StatusBadge } from "@/components/ui/status-badge.tsx";
import { MetaItem } from "@/components/meta-item.tsx";
import { SectionHeading } from "@/components/section-heading.tsx";
import { ConfirmDialog } from "@/components/ui/confirm-dialog.tsx";
import { CreateServiceDialog } from "./components/create-service-dialog.tsx";
import { useService, useDeleteService } from "@/hooks/use-services.ts";
import { useUIStore } from "@/stores/ui-store.ts";
import { formatDate, formatRelativeTime, cn } from "@/lib/utils.ts";
import type { Service, ServiceAlert, HealthCheckSummary } from "@/types/service.ts";

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

function checkRowMessage(check: HealthCheckSummary): string {
  if (check.is_success) return "Service responded normally";
  if (!check.message) return "Check failed";

  const n = check.message.toLowerCase();
  if (n.includes("connection refused")) return "Connection refused";
  if (n.includes("timeout") || n.includes("timed out")) return "Connection timed out";
  if (n.includes("no such host") || n.includes("name resolution")) return "Host not found";
  return check.message;
}

function approximateCheckHistory(service: Service, length = 12): boolean[] {
  const latest = service.latest_health_check;
  if (!latest) return [];

  const failures = Math.max(0, service.consecutive_failures);

  if (latest.is_success) {
    const recentFailures = Math.min(failures, length - 1);
    return Array.from({ length }, (_, i) => i >= recentFailures);
  }

  const trailingFailures = Math.max(1, Math.min(failures, length));
  const leadingSuccesses = length - trailingFailures;
  return Array.from({ length }, (_, i) => i < leadingSuccesses);
}

// ─── Check row ────────────────────────────────────────────────────────────────

function CheckRow({ check }: { check: HealthCheckSummary }) {
  return (
    <div className="flex items-center gap-3 px-4 py-3 min-w-0">
      {check.is_success ? (
        <CheckCircle2Icon size={14} className="text-primary shrink-0" />
      ) : (
        <XCircleIcon size={14} className="text-destructive shrink-0" />
      )}
      <div className="min-w-0 flex-1">
        <p
          className={cn(
            "text-sm font-medium truncate",
            check.is_success ? "text-foreground" : "text-destructive",
          )}
        >
          {checkRowMessage(check)}
        </p>
      </div>
      <div className="flex items-center gap-3 ml-2 shrink-0 text-xs text-muted-foreground tabular-nums">
        {check.response_time_ms != null && <span>{check.response_time_ms}ms</span>}
        <span>{check.status_code != null ? `HTTP ${check.status_code}` : "No response"}</span>
        <span className="hidden sm:inline">{formatRelativeTime(check.observed_at)}</span>
      </div>
    </div>
  );
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
  const serviceId = id ? parseInt(id, 10) : 0;

  const { data, isLoading, error } = useService(serviceId);
  const service = data?.service;
  const deleteService = useDeleteService(activeProjectId ?? 0);

  function handleDelete() {
    deleteService.mutate(serviceId, {
      onSuccess: () => {
        toast.success("Service deleted");
        navigate("/services");
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
          onClick={() => navigate("/services")}
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
  const checkHistory = approximateCheckHistory(service, 12);

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
                value={
                  service.last_checked_at ? formatRelativeTime(service.last_checked_at) : "Never"
                }
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
            <h2 className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">
              Service Health
            </h2>
          </div>

          {checkHistory.length === 0 ? (
            <p className="text-sm text-muted-foreground py-2">No checks have run yet.</p>
          ) : (
            <div className="flex items-start justify-between">
              {/* State label */}
              <div className="space-y-1">
                <p
                  className={cn(
                    "text-4xl font-bold tracking-tight leading-none font-heading",
                    isHealthy ? "text-primary" : "text-destructive",
                  )}
                >
                  {stateLabel(service)}
                </p>
                <p className="text-sm text-muted-foreground">
                  {service.consecutive_failures > 0
                    ? `${service.consecutive_failures} consecutive ${service.consecutive_failures === 1 ? "failure" : "failures"}`
                    : "No recent failures"}
                </p>
              </div>
              {/* Segmented check-history strip */}
              <div className="space-y-1.5">
                <p className="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/50">
                  Recent Checks
                </p>
                <div className="flex items-center gap-1">
                  {checkHistory.map((isSuccess, idx) => {
                    const opacity = idx * 0.07 + 0.25; // Older checks are more transparent
                    return (
                      <span
                        key={idx}
                        style={{ opacity }}
                        className={cn(
                          "size-3  rounded-full",
                          isSuccess ? "bg-primary" : "bg-destructive",
                        )}
                      />
                    );
                  })}
                </div>
                {latest?.observed_at && (
                  <span className="text-xs text-muted-foreground tabular-nums">
                    Last checked: {formatRelativeTime(latest.observed_at)}
                  </span>
                )}
              </div>
            </div>
          )}
        </section>

        {/* ── 3. Recent checks — dense list section ── */}
        <section>
          <SectionHeading
            icon={<ClockIcon size={13} />}
            label="Recent Checks"
            aside={latest ? <span className="text-xs text-muted-foreground">1</span> : undefined}
          />
          {!latest ? (
            <div className="rounded-lg border border-dashed border-border px-4 py-5 text-center">
              <p className="text-sm text-muted-foreground">No check results yet.</p>
            </div>
          ) : (
            <div className="rounded-lg border border-border bg-card overflow-hidden divide-y divide-border">
              {/* Left accent stripe matching Node page service rows */}
              <div className="flex items-stretch">
                <div
                  className={cn(
                    "w-0.5 shrink-0",
                    latest.is_success ? "bg-primary" : "bg-destructive",
                  )}
                />
                <div className="flex-1 min-w-0">
                  <CheckRow check={latest} />
                </div>
              </div>
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
                      Triggered {formatRelativeTime(alert.triggered_at)}
                    </p>
                  </div>
                  <div className="flex flex-col items-end gap-1 ml-2 shrink-0">
                    <span className="text-xs text-muted-foreground whitespace-nowrap">
                      {formatRelativeTime(alert.triggered_at)}
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
            label="Service Configuration"
            action={
              <Link
                to={`/nodes/${service.node_id}`}
                className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                View node #{service.node_id}
                <ChevronRightIcon size={12} />
              </Link>
            }
          />
          <div className="flex flex-wrap gap-x-6 gap-y-2">
            <MetaItem label="Check type" value={service.check_type.toUpperCase()} />
            <MetaItem label="Execution mode" value={service.execution_mode} />
            <MetaItem label="Node ID" value={String(service.node_id)} />
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
