import { useMemo, useState } from "react";
import { type ColumnDef } from "@tanstack/react-table";
import { AlertTriangleIcon, ClockIcon, SirenIcon } from "lucide-react";
import { SectionHeading } from "@/components/section-heading.tsx";
import {
  alertResourceLabel,
  deduplicateAlerts,
  historyFilterTriggerClass,
  ruleTypeLabel,
} from "@/app/alerts/alert-presentation.ts";
import { Button } from "@/components/ui/button.tsx";
import { DataTable } from "@/components/ui/data-table.tsx";
import { PageHeader } from "@/components/ui/page-header.tsx";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { StatusBadge } from "@/components/ui/status-badge.tsx";
import { useActiveAlerts, useAlertHistory, useAlertHistoryHeadSync } from "@/hooks/use-alerts.ts";
import { useMeta } from "@/hooks/use-meta";
import { useNodes } from "@/hooks/use-nodes.ts";
import { useServices } from "@/hooks/use-services.ts";
import { formatAlertDuration } from "@/lib/alert-duration.ts";
import { cn, formatRelativeTime } from "@/lib/utils.ts";
import { useUIStore } from "@/stores/ui-store.ts";
import type { Alert, AlertCategory, AlertHistoryFilters, Severity } from "@/types/alert.ts";

function severityBadgeClass(severity: Severity) {
  switch (severity) {
    case "critical":
      return "text-destructive border-destructive/25 bg-destructive/10";
    case "warning":
      return "text-warning border-warning/25 bg-warning/10";
    default:
      return "text-muted-foreground border-border bg-muted/30";
  }
}

function severityLabel(severity: Severity) {
  return severity.charAt(0).toUpperCase() + severity.slice(1);
}

export function ActiveAlertRow({ alert }: { alert: Alert }) {
  return (
    <div className="flex items-stretch" data-alert-id={alert.id}>
      <div className="w-0.5 shrink-0 bg-destructive/60" />
      <div className="flex min-w-0 flex-1 items-start justify-between gap-4 bg-destructive/3 px-4 py-3.5">
        <div className="flex min-w-0 flex-1 items-start gap-3">
          <AlertTriangleIcon size={14} className="mt-0.5 shrink-0 text-destructive" />
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold leading-snug text-foreground">
              {alert.title}
            </p>
            <p className="mt-0.5 truncate text-xs text-muted-foreground">
              {alertResourceLabel(alert)} · {ruleTypeLabel(alert.rule_type)}
            </p>
            {alert.message ? (
              <p className="mt-0.5 truncate text-xs text-muted-foreground/70">{alert.message}</p>
            ) : null}
            <p className="mt-1 text-xs text-muted-foreground/60">
              Triggered {formatRelativeTime(alert.triggered_at)}
            </p>
          </div>
        </div>
        <div className="mt-0.5 flex shrink-0 items-center gap-1.5">
          <span
            className={cn(
              "rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide",
              severityBadgeClass(alert.severity),
            )}
          >
            {severityLabel(alert.severity)}
          </span>
          <StatusBadge status={alert.status} />
        </div>
      </div>
    </div>
  );
}

export function AlertHistoryTable({ alerts }: { alerts: Alert[] }) {
  const columns: ColumnDef<Alert>[] = [
    {
      accessorKey: "title",
      header: "Alert",
      meta: {
        headClassName:
          "h-auto min-w-52 px-4 pb-2 text-xs font-semibold uppercase tracking-widest text-muted-foreground/50",
        cellClassName: "px-4 py-3",
      },
      cell: ({ row }) => (
        <>
          <p className="text-sm leading-snug text-foreground">{row.original.title}</p>
          {row.original.message ? (
            <p className="mt-0.5 max-w-xs truncate text-xs text-muted-foreground/70">
              {row.original.message}
            </p>
          ) : null}
          <p className="mt-1 text-[11px] text-muted-foreground sm:hidden">
            {ruleTypeLabel(row.original.rule_type)} · {alertResourceLabel(row.original)}
          </p>
          <p className="mt-0.5 text-[11px] text-muted-foreground/70 sm:hidden">
            Triggered {formatRelativeTime(row.original.triggered_at)} ·{" "}
            {row.original.resolved_at
              ? formatAlertDuration(row.original.triggered_at, row.original.resolved_at)
              : "Ongoing"}
          </p>
        </>
      ),
    },
    {
      accessorKey: "rule_type",
      header: "Type",
      meta: {
        headClassName:
          "hidden h-auto px-4 pb-2 text-xs font-semibold uppercase tracking-widest text-muted-foreground/50 sm:table-cell",
        cellClassName: "hidden px-4 py-3 text-xs text-muted-foreground sm:table-cell",
      },
      cell: ({ row }) => ruleTypeLabel(row.original.rule_type),
    },
    {
      id: "resource",
      header: "Resource",
      meta: {
        headClassName:
          "hidden h-auto px-4 pb-2 text-xs font-semibold uppercase tracking-widest text-muted-foreground/50 md:table-cell",
        cellClassName: "hidden px-4 py-3 text-xs text-muted-foreground md:table-cell",
      },
      cell: ({ row }) => alertResourceLabel(row.original),
    },
    {
      accessorKey: "triggered_at",
      header: "Triggered",
      meta: {
        headClassName:
          "hidden h-auto px-4 pb-2 text-xs font-semibold uppercase tracking-widest text-muted-foreground/50 lg:table-cell",
        cellClassName: "hidden px-4 py-3 text-xs text-muted-foreground tabular-nums lg:table-cell",
      },
      cell: ({ row }) => (
        <div>
          <p>{formatRelativeTime(row.original.triggered_at)}</p>
          {row.original.resolved_at ? (
            <p className="mt-0.5 text-[11px] text-muted-foreground/60">
              Recovered {formatRelativeTime(row.original.resolved_at)}
            </p>
          ) : null}
        </div>
      ),
    },
    {
      id: "duration",
      header: "Duration",
      meta: {
        headClassName:
          "hidden h-auto px-4 pb-2 text-xs font-semibold uppercase tracking-widest text-muted-foreground/50 sm:table-cell",
        cellClassName: "hidden px-4 py-3 text-xs text-muted-foreground tabular-nums sm:table-cell",
      },
      cell: ({ row }) =>
        row.original.resolved_at
          ? formatAlertDuration(row.original.triggered_at, row.original.resolved_at)
          : "—",
    },
    {
      id: "severity-status",
      header: "Severity / Status",
      meta: {
        headClassName:
          "h-auto px-4 pb-2 text-right text-xs font-semibold uppercase tracking-widest text-muted-foreground/50",
        cellClassName: "px-4 py-3 text-right",
      },
      cell: ({ row }) => (
        <div className="flex flex-wrap justify-end gap-1">
          <span
            className={cn(
              "rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide",
              severityBadgeClass(row.original.severity),
            )}
          >
            {severityLabel(row.original.severity)}
          </span>
          <StatusBadge status={row.original.status} />
        </div>
      ),
    },
  ];

  return (
    <DataTable
      columns={columns}
      data={alerts}
      stickyHeader
      tableWrapperClassName="max-h-128"
      rowClassName={() => "opacity-80"}
    />
  );
}

export function ActiveAlertsSkeleton() {
  return (
    <div className="divide-y divide-border overflow-hidden rounded-lg border border-border bg-card">
      {Array.from({ length: 2 }).map((_, index) => (
        <div key={index} className="flex items-stretch">
          <div className="w-0.5 shrink-0 bg-destructive/40" />
          <div className="flex-1 space-y-2 px-4 py-3.5">
            <Skeleton className="h-3.5 w-56" />
            <Skeleton className="h-3 w-32" />
            <Skeleton className="h-3 w-24" />
          </div>
        </div>
      ))}
    </div>
  );
}

export function TableSkeleton() {
  return (
    <div className="space-y-1">
      {Array.from({ length: 3 }).map((_, index) => (
        <Skeleton key={index} className="h-14 w-full" />
      ))}
    </div>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="flex items-center justify-between gap-4 rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3">
      <p className="text-sm text-destructive" role="alert">
        {message}
      </p>
      <Button variant="ghost" size="sm" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}

export function BackgroundRefreshError({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="mt-2 flex items-center gap-2 text-xs text-destructive" role="alert">
      <span>Refresh failed. Showing saved active alerts.</span>
      <Button variant="link" size="sm" className="h-auto p-0 text-xs" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}

export function EmptyState({ message, active = false }: { message: string; active?: boolean }) {
  return (
    <div className="flex items-center gap-2 py-1">
      <span
        className={cn(
          "h-1.5 w-1.5 shrink-0 rounded-full",
          active ? "bg-primary" : "bg-muted-foreground/30",
        )}
      />
      <p className="text-sm text-muted-foreground">{message}</p>
    </div>
  );
}

export function AlertHistoryPagination({
  hasNextPage,
  isFetchingNextPage,
  isFetchNextPageError,
  onLoadMore,
}: {
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  isFetchNextPageError: boolean;
  onLoadMore: () => void;
}) {
  if (!hasNextPage && !isFetchNextPageError) return null;
  return (
    <div className="flex flex-col items-center gap-1.5 pt-3">
      {isFetchNextPageError ? (
        <p className="text-xs text-destructive" role="alert">
          Couldn’t load older alerts. Your loaded history is still shown.
        </p>
      ) : null}
      <Button
        variant="ghost"
        size="sm"
        disabled={isFetchingNextPage}
        onClick={onLoadMore}
        className="text-xs text-muted-foreground"
      >
        {isFetchingNextPage
          ? "Loading older alerts…"
          : isFetchNextPageError
            ? "Try again"
            : "Load older alerts"}
      </Button>
    </div>
  );
}

function HistoryFilters({
  category,
  severity,
  resource,
  nodes,
  services,
  onCategoryChange,
  onSeverityChange,
  onResourceChange,
}: {
  category: AlertCategory | "all";
  severity: Severity | "all";
  resource: string;
  nodes: Array<{ id: number; name: string }>;
  services: Array<{ id: number; name: string }>;
  onCategoryChange: (value: AlertCategory | "all") => void;
  onSeverityChange: (value: Severity | "all") => void;
  onResourceChange: (value: string) => void;
}) {
  return (
    <div className="ml-auto flex flex-wrap justify-end gap-2" aria-label="Alert history filters">
      <Select
        value={category}
        onValueChange={(value) => onCategoryChange(value as AlertCategory | "all")}
      >
        <SelectTrigger
          className={historyFilterTriggerClass(category !== "all")}
          aria-label="Alert type"
        >
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All types</SelectItem>
          <SelectItem value="node">Node</SelectItem>
          <SelectItem value="service">Service</SelectItem>
          <SelectItem value="metric">Metric</SelectItem>
        </SelectContent>
      </Select>
      <Select
        value={severity}
        onValueChange={(value) => onSeverityChange(value as Severity | "all")}
      >
        <SelectTrigger
          className={historyFilterTriggerClass(severity !== "all")}
          aria-label="Alert severity"
        >
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All severities</SelectItem>
          <SelectItem value="critical">Critical</SelectItem>
          <SelectItem value="warning">Warning</SelectItem>
          <SelectItem value="info">Info</SelectItem>
        </SelectContent>
      </Select>
      <Select value={resource} onValueChange={onResourceChange}>
        <SelectTrigger
          className={historyFilterTriggerClass(resource !== "all", "min-w-40")}
          aria-label="Affected resource"
        >
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All resources</SelectItem>
          {nodes.map((node) => (
            <SelectItem key={`node:${node.id}`} value={`node:${node.id}`}>
              Node · {node.name}
            </SelectItem>
          ))}
          {services.map((service) => (
            <SelectItem key={`service:${service.id}`} value={`service:${service.id}`}>
              Service · {service.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

export function AlertsPage() {
  const activeProjectId = useUIStore((state) => state.activeProjectId) ?? 0;
  const [category, setCategory] = useState<AlertCategory | "all">("all");
  const [severity, setSeverity] = useState<Severity | "all">("all");
  const [resource, setResource] = useState("all");

  const filters = useMemo<AlertHistoryFilters>(() => {
    const next: AlertHistoryFilters = {};
    if (category !== "all") next.category = category;
    if (severity !== "all") next.severity = severity;
    const [resourceType, rawID] = resource.split(":");
    const resourceID = Number(rawID);
    if (resourceType === "node" && Number.isInteger(resourceID)) next.nodeId = resourceID;
    if (resourceType === "service" && Number.isInteger(resourceID)) next.serviceId = resourceID;
    return next;
  }, [category, resource, severity]);

  const activeQuery = useActiveAlerts(activeProjectId);
  const historyQuery = useAlertHistory(activeProjectId, filters);
  const nodesQuery = useNodes(activeProjectId);
  const servicesQuery = useServices(activeProjectId, { refetchInterval: false });
  const activeAlerts = activeQuery.data?.alerts ?? [];
  useAlertHistoryHeadSync(activeProjectId, filters, activeQuery.data, activeQuery.dataUpdatedAt);

  const historyAlerts = useMemo(
    () => deduplicateAlerts(historyQuery.data?.pages.flatMap((page) => page.alerts) ?? []),
    [historyQuery.data],
  );

  useMeta({
    title: "Alerts",
    description: "View active alerts and resolved alert history for your project",
  });

  return (
    <div className="mx-auto max-w-6xl space-y-8 p-6">
      <PageHeader title="Alerts" description="Active alerts and resolved history" />

      <section>
        <SectionHeading
          icon={<SirenIcon size={13} />}
          label="Active Alerts"
          aside={
            activeAlerts.length > 0 ? (
              <span className="text-xs font-semibold text-destructive">{activeAlerts.length}</span>
            ) : undefined
          }
        />
        {activeQuery.isPending && activeQuery.data == null ? (
          <ActiveAlertsSkeleton />
        ) : activeQuery.isError && activeQuery.data == null ? (
          <ErrorState
            message="Couldn’t load active alerts."
            onRetry={() => void activeQuery.refetch()}
          />
        ) : activeAlerts.length === 0 ? (
          <EmptyState active message="No active alerts right now." />
        ) : (
          <div className="divide-y divide-border/60 overflow-hidden rounded-lg border border-destructive/20">
            {activeAlerts.map((alert) => (
              <ActiveAlertRow key={alert.id} alert={alert} />
            ))}
          </div>
        )}
        {activeQuery.isError && activeQuery.data != null ? (
          <BackgroundRefreshError onRetry={() => void activeQuery.refetch()} />
        ) : null}
      </section>

      <section className="flex flex-col gap-3">
        <SectionHeading
          icon={<ClockIcon size={13} />}
          label="Alert History"
          aside={
            historyAlerts.length > 0 ? (
              <span className="text-xs text-muted-foreground">{historyAlerts.length} loaded</span>
            ) : undefined
          }
          action={
            <HistoryFilters
              category={category}
              severity={severity}
              resource={resource}
              nodes={nodesQuery.data?.nodes ?? []}
              services={servicesQuery.data?.services ?? []}
              onCategoryChange={(value) => {
                setCategory(value);
                setResource("all");
              }}
              onSeverityChange={setSeverity}
              onResourceChange={setResource}
            />
          }
        />

        {historyQuery.isPending ? (
          <TableSkeleton />
        ) : historyQuery.isError && historyAlerts.length === 0 ? (
          <ErrorState
            message="Couldn’t load alert history."
            onRetry={() => void historyQuery.refetch()}
          />
        ) : historyAlerts.length === 0 ? (
          <EmptyState message="No resolved alerts match these filters." />
        ) : (
          <>
            <AlertHistoryTable alerts={historyAlerts} />
            <AlertHistoryPagination
              hasNextPage={historyQuery.hasNextPage}
              isFetchingNextPage={historyQuery.isFetchingNextPage}
              isFetchNextPageError={historyQuery.isFetchNextPageError}
              onLoadMore={() => void historyQuery.fetchNextPage()}
            />
          </>
        )}
      </section>
    </div>
  );
}
