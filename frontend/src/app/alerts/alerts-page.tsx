import { useState } from "react";
import { type ColumnDef } from "@tanstack/react-table";
import { AlertTriangleIcon, ClockIcon, SirenIcon } from "lucide-react";
import { Button } from "@/components/ui/button.tsx";
import { DataTable } from "@/components/ui/data-table.tsx";
import { EmptyState } from "@/components/ui/empty-state.tsx";
import { PageHeader } from "@/components/ui/page-header.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { StatusBadge } from "@/components/ui/status-badge.tsx";
import { Switch } from "@/components/ui/switch.tsx";
import { SectionHeading } from "@/components/section-heading.tsx";
import { useAlertRules, useAlerts, useUpdateAlertRule } from "@/hooks/use-alerts.ts";
import { formatRelativeTime, cn } from "@/lib/utils.ts";
import { useUIStore } from "@/stores/ui-store.ts";
import { CreateAlertRuleDialog } from "./components/create-alert-rule-dialog.tsx";
import { AnimateIcon, BellIcon, PlusIcon } from "@/components/animate-ui/icons";
import type { Alert, AlertRule } from "@/types/alert.ts";

// ─── Helpers ──────────────────────────────────────────────────────────────────

function severityFromAlert(alert: Alert): "critical" | "warning" | "info" {
  const t = alert.title.toLowerCase();
  const m = alert.message.toLowerCase();
  if (t.includes("unhealthy") || t.includes("offline") || m.includes("refused") || m.includes("unreachable")) {
    return "critical";
  }
  if (t.includes("degraded") || t.includes("threshold") || m.includes("above")) {
    return "warning";
  }
  return "info";
}

function severityBadgeClass(severity: "critical" | "warning" | "info") {
  switch (severity) {
    case "critical":
      return "text-destructive border-destructive/25 bg-destructive/10";
    case "warning":
      return "text-warning border-warning/25 bg-warning/10";
    default:
      return "text-muted-foreground border-border bg-muted/30";
  }
}

function severityLabel(severity: "critical" | "warning" | "info") {
  switch (severity) {
    case "critical": return "Critical";
    case "warning": return "Warning";
    default: return "Info";
  }
}

// ─── Active alert row ─────────────────────────────────────────────────────────

function ActiveAlertRow({ alert }: { alert: Alert }) {
  const severity = severityFromAlert(alert);

  return (
    <div className="flex items-stretch">
      {/* Left accent stripe */}
      <div className="w-0.5 shrink-0 bg-destructive/60" />
      <div className={cn(
        "flex flex-1 items-start justify-between gap-4 px-4 py-3.5 min-w-0",
        "bg-destructive/3",
      )}>
        {/* Identity */}
        <div className="flex items-start gap-3 min-w-0 flex-1">
          <AlertTriangleIcon
            size={14}
            className="text-destructive shrink-0 mt-0.5"
          />
          <div className="min-w-0">
            <p className="text-sm font-semibold text-foreground leading-snug truncate">
              {alert.title}
            </p>
            {alert.message && (
              <p className="text-xs text-muted-foreground mt-0.5 truncate">
                {alert.message}
              </p>
            )}
            <p className="text-xs text-muted-foreground/60 mt-1">
              Triggered {formatRelativeTime(alert.triggered_at)}
            </p>
          </div>
        </div>

        {/* Right: badges */}
        <div className="flex items-center gap-1.5 shrink-0 mt-0.5">
          <span className={cn(
            "rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide",
            severityBadgeClass(severity),
          )}>
            {severityLabel(severity)}
          </span>
          <StatusBadge status={alert.status} />
        </div>
      </div>
    </div>
  );
}

// ─── Alert history row (table) ────────────────────────────────────────────────

function AlertHistoryTable({ alerts }: { alerts: Alert[] }) {
  const columns: ColumnDef<Alert>[] = [
    {
      accessorKey: "title",
      header: "Alert",
      meta: {
        headClassName: "h-auto w-1/2 px-4 pb-2 text-xs font-semibold uppercase tracking-widest text-muted-foreground/50",
        cellClassName: "px-4 py-3",
      },
      cell: ({ row }) => {
        const alert = row.original;
        const isResolved = alert.status === "resolved";

        return (
          <>
            <p
              className={cn(
                "text-sm leading-snug",
                isResolved ? "font-normal text-muted-foreground" : "font-medium text-foreground",
              )}
            >
              {alert.title}
            </p>
            {alert.message && (
              <p className="mt-0.5 max-w-xs truncate text-xs text-muted-foreground/70">
                {alert.message}
              </p>
            )}
          </>
        );
      },
    },
    {
      accessorKey: "triggered_at",
      header: "Triggered",
      meta: {
        headClassName: "hidden h-auto px-4 pb-2 text-xs font-semibold uppercase tracking-widest text-muted-foreground/50 sm:table-cell",
        cellClassName: "hidden px-4 py-3 text-xs text-muted-foreground tabular-nums sm:table-cell",
      },
      cell: ({ row }) => formatRelativeTime(row.original.triggered_at),
    },
    {
      accessorKey: "resolved_at",
      header: "Resolved",
      meta: {
        headClassName: "hidden h-auto px-4 pb-2 text-xs font-semibold uppercase tracking-widest text-muted-foreground/50 md:table-cell",
        cellClassName: "hidden px-4 py-3 text-xs text-muted-foreground tabular-nums md:table-cell",
      },
      cell: ({ row }) =>
        row.original.resolved_at ? formatRelativeTime(row.original.resolved_at) : "—",
    },
    {
      id: "status",
      header: "Status",
      meta: {
        headClassName: "h-auto px-4 pb-2 text-right text-xs font-semibold uppercase tracking-widest text-muted-foreground/50",
        cellClassName: "px-4 py-3 text-right",
      },
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
    },
  ];

  return (
    <DataTable
      columns={columns}
      data={alerts}
      stickyHeader
      tableWrapperClassName="max-h-80"
      rowClassName={(row) => (row.original.status === "resolved" ? "opacity-50" : undefined)}
    />
  );
}

// ─── Alert rules table ────────────────────────────────────────────────────────

function AlertRulesTable({
  rules,
  onToggle,
}: {
  rules: AlertRule[];
  onToggle: (id: number, enabled: boolean) => void;
}) {
  const columns: ColumnDef<AlertRule>[] = [
    {
      accessorKey: "rule_type",
      header: "Rule",
      meta: {
        headClassName: "h-auto px-4 pb-2 text-xs font-semibold uppercase tracking-widest text-muted-foreground/50",
        cellClassName: "px-4 py-3",
      },
      cell: ({ row }) => (
        <p className="text-sm font-medium capitalize text-foreground">
          {row.original.rule_type.replaceAll("_", " ")}
        </p>
      ),
    },
    {
      accessorKey: "threshold_value",
      header: "Threshold",
      meta: {
        headClassName: "hidden h-auto px-4 pb-2 text-xs font-semibold uppercase tracking-widest text-muted-foreground/50 sm:table-cell",
        cellClassName: "hidden px-4 py-3 text-xs text-muted-foreground sm:table-cell",
      },
      cell: ({ row }) =>
        row.original.threshold_value != null ? `${row.original.threshold_value}%` : "—",
    },
    {
      id: "enabled",
      header: "Enabled",
      meta: {
        headClassName: "h-auto px-4 pb-2 text-right text-xs font-semibold uppercase tracking-widest text-muted-foreground/50",
        cellClassName: "px-4 py-3 text-right",
      },
      cell: ({ row }) => (
        <Switch
          checked={row.original.is_enabled}
          onCheckedChange={(checked) => onToggle(row.original.id, checked)}
        />
      ),
    },
  ];

  return (
    <DataTable columns={columns} data={rules} stickyHeader tableWrapperClassName="max-h-80" />
  );
}

// ─── Skeletons ────────────────────────────────────────────────────────────────

function ActiveAlertsSkeleton() {
  return (
    <div className="rounded-lg border border-border bg-card overflow-hidden divide-y divide-border">
      {Array.from({ length: 2 }).map((_, i) => (
        <div key={i} className="flex items-stretch">
          <div className="w-0.5 bg-destructive/40 shrink-0" />
          <div className="flex-1 px-4 py-3.5 space-y-2">
            <Skeleton className="h-3.5 w-56" />
            <Skeleton className="h-3 w-32" />
            <Skeleton className="h-3 w-24" />
          </div>
        </div>
      ))}
    </div>
  );
}

function TableSkeleton() {
  return (
    <div className="space-y-1">
      {Array.from({ length: 3 }).map((_, i) => (
        <Skeleton key={i} className="h-11 w-full" />
      ))}
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export function AlertsPage() {
  const activeProjectId = useUIStore((s) => s.activeProjectId);
  const { data: alertsData, isLoading: alertsLoading } = useAlerts(activeProjectId ?? 0);
  const { data: rulesData, isLoading: rulesLoading } = useAlertRules(activeProjectId ?? 0);
  const updateRule = useUpdateAlertRule(activeProjectId ?? 0);
  const [createOpen, setCreateOpen] = useState(false);

  const allAlerts = alertsData?.alerts ?? [];
  const activeAlerts = allAlerts.filter((a) => a.status === "active");
  // History = resolved + any active not shown above (de-dup by id)
  const historyAlerts = allAlerts.filter((a) => a.status === "resolved");
  const rules = rulesData?.alert_rules ?? [];

  return (
    <div className="p-6 space-y-8 max-w-6xl mx-auto">
      <PageHeader
        title="Alerts"
        description="Active alerts and alert rules"
        actions={
          <AnimateIcon asChild animateOnHover>
            <Button variant={"secondary"} size="sm" onClick={() => setCreateOpen(true)}>
              <PlusIcon size={14} className="mr-1.5" />
              Add rule
            </Button>
          </AnimateIcon>
        }
      />

      {/* ── 1. Active Alerts ── */}
      <section>
        <SectionHeading
          icon={<SirenIcon size={13} />}
          label="Active Alerts"
          aside={
            activeAlerts.length > 0 ? (
              <span className="text-xs font-semibold text-destructive">
                {activeAlerts.length}
              </span>
            ) : undefined
          }
        />

        {alertsLoading ? (
          <ActiveAlertsSkeleton />
        ) : activeAlerts.length === 0 ? (
          <div className="flex items-center gap-2 py-1">
            <span className="h-1.5 w-1.5 rounded-full bg-primary shrink-0" />
            <p className="text-sm text-muted-foreground">No active alerts right now.</p>
          </div>
        ) : (
          <div className="rounded-lg border border-destructive/20 overflow-hidden divide-y divide-border/60">
            {activeAlerts.map((alert) => (
              <ActiveAlertRow key={alert.id} alert={alert} />
            ))}
          </div>
        )}
      </section>

      {/* ── 2. Alert History ── */}
      <section>
        <SectionHeading
          icon={<ClockIcon size={13} />}
          label="Alert History"
          aside={
            historyAlerts.length > 0 ? (
              <span className="text-xs text-muted-foreground">{historyAlerts.length}</span>
            ) : undefined
          }
        />

        {alertsLoading ? (
          <TableSkeleton />
        ) : historyAlerts.length === 0 ? (
          <div className="flex items-center gap-2 py-1">
            <span className="h-1.5 w-1.5 rounded-full bg-muted-foreground/30 shrink-0" />
            <p className="text-sm text-muted-foreground">No resolved alerts yet.</p>
          </div>
        ) : (
          <AlertHistoryTable alerts={historyAlerts} />
        )}
      </section>

      {/* ── 3. Alert Rules ── */}
      <section>
        <SectionHeading
          icon={<BellIcon size={13} />}
          label="Alert Rules"
          aside={
            rules.length > 0 ? (
              <span className="text-xs text-muted-foreground">{rules.length}</span>
            ) : undefined
          }
          action={
            <button
              onClick={() => setCreateOpen(true)}
              className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
            >
              <PlusIcon size={12} />
              Add rule
            </button>
          }
        />

        {rulesLoading ? (
          <TableSkeleton />
        ) : rules.length === 0 ? (
          <EmptyState
            icon={BellIcon}
            title="No alert rules"
            description="Create a rule to get notified when something goes wrong."
            action={{ label: "Add rule", onClick: () => setCreateOpen(true) }}
          />
        ) : (
          <AlertRulesTable
            rules={rules}
            onToggle={(id, enabled) =>
              updateRule.mutate({ id, payload: { is_enabled: enabled } })
            }
          />
        )}
      </section>

      {activeProjectId && (
        <CreateAlertRuleDialog
          projectId={activeProjectId}
          open={createOpen}
          onOpenChange={setCreateOpen}
        />
      )}
    </div>
  );
}
