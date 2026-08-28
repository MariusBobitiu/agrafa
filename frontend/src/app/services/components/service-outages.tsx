import { useEffect, useMemo, useState } from "react";
import { ClockIcon, LoaderCircleIcon, TriangleAlertIcon } from "lucide-react";
import { Link } from "react-router-dom";
import {
  hasApplicableServiceOutageRule,
  resolvedServiceOutages,
  SERVICE_OUTAGE_HISTORY_LIMIT,
  serviceOutageAlerts,
  serviceOutageHistoryFilters,
} from "@/app/services/service-outage-utils.ts";
import { SectionHeading } from "@/components/section-heading.tsx";
import { Button } from "@/components/ui/button.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import {
  useAlertHistory,
  useAlertHistoryIdentityHeadSync,
  useAlertRules,
} from "@/hooks/use-alerts.ts";
import { useCanWrite } from "@/hooks/use-project-role.ts";
import { formatAlertDuration, formatOngoingAlertDuration } from "@/lib/alert-duration.ts";
import { cn, formatDate } from "@/lib/utils.ts";
import type { Alert, Severity } from "@/types/alert.ts";
import type { ServiceAlert } from "@/types/service.ts";

function severityBadgeClass(severity: Severity) {
  switch (severity) {
    case "critical":
      return "border-destructive/25 bg-destructive/10 text-destructive";
    case "warning":
      return "border-warning/25 bg-warning/10 text-warning";
    default:
      return "border-border bg-muted/30 text-muted-foreground";
  }
}

function severityLabel(severity: Severity) {
  return severity.charAt(0).toUpperCase() + severity.slice(1);
}

function useOutageClock(outages: readonly ServiceAlert[]) {
  const [now, setNow] = useState(() => Date.now());
  const outageKey = outages.map((outage) => `${outage.id}:${outage.triggered_at}`).join(",");

  useEffect(() => {
    if (outages.length === 0) return;

    const interval = setInterval(() => setNow(Date.now()), 60_000);
    return () => clearInterval(interval);
  }, [outageKey, outages.length]);

  return now;
}

export function CurrentOutages({
  outages,
  now,
}: {
  outages: readonly ServiceAlert[];
  now: number;
}) {
  if (outages.length === 0) return null;

  return (
    <div className="space-y-2" aria-label="Current outages">
      {outages.map((outage) => (
        <div
          key={outage.id}
          className="overflow-hidden rounded-lg border border-destructive/25 bg-destructive/5"
          data-outage-id={outage.id}
        >
          <div className="flex items-center gap-2 border-b border-destructive/15 px-4 py-2.5">
            <span className="relative flex size-2">
              <span className="absolute inline-flex size-full animate-ping rounded-full bg-destructive opacity-50" />
              <span className="relative inline-flex size-2 rounded-full bg-destructive" />
            </span>
            <p className="text-xs font-semibold uppercase tracking-widest text-destructive">
              {outages.length === 1 ? "Current outage" : "Current outages"}
            </p>
          </div>
          <dl className="grid grid-cols-2 gap-x-5 gap-y-3 px-4 py-3.5 sm:grid-cols-4">
            <div>
              <dt className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">
                Started
              </dt>
              <dd className="mt-0.5 text-sm font-semibold tabular-nums text-foreground">
                {formatOngoingAlertDuration(outage.triggered_at, now)} ago
              </dd>
            </div>
            <div>
              <dt className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">
                Since
              </dt>
              <dd className="mt-0.5 text-sm tabular-nums text-foreground">
                {formatDate(outage.triggered_at)}
              </dd>
            </div>
            <div>
              <dt className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">
                Severity
              </dt>
              <dd className="mt-1">
                <span
                  className={cn(
                    "rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide",
                    severityBadgeClass(outage.severity),
                  )}
                >
                  {severityLabel(outage.severity)}
                </span>
              </dd>
            </div>
            <div>
              <dt className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">
                State
              </dt>
              <dd className="mt-1">
                <span className="rounded-full border border-destructive/25 bg-destructive/10 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-destructive">
                  Ongoing
                </span>
              </dd>
            </div>
          </dl>
        </div>
      ))}
    </div>
  );
}

export function ResolvedOutagesTable({ outages }: { outages: readonly Alert[] }) {
  return (
    <div
      className="overflow-hidden rounded-lg border border-border bg-card"
      aria-label="Resolved outages"
    >
      <div className="hidden grid-cols-[1.25fr_1.25fr_0.75fr_0.65fr] gap-4 border-b border-border bg-muted/20 px-4 py-2.5 text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/50 sm:grid">
        <span>Started</span>
        <span>Recovered</span>
        <span>Duration</span>
        <span>Severity</span>
      </div>
      <div className="divide-y divide-border/70">
        {outages.map((outage) => (
          <div
            key={outage.id}
            className="grid gap-2 px-4 py-3 sm:grid-cols-[1.25fr_1.25fr_0.75fr_0.65fr] sm:items-center sm:gap-4"
            data-outage-id={outage.id}
          >
            <div>
              <span className="mr-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/50 sm:hidden">
                Started
              </span>
              <span className="text-sm tabular-nums text-foreground">
                {formatDate(outage.triggered_at)}
              </span>
            </div>
            <div>
              <span className="mr-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/50 sm:hidden">
                Recovered
              </span>
              <span className="text-sm tabular-nums text-muted-foreground">
                {formatDate(outage.resolved_at)}
              </span>
            </div>
            <div>
              <span className="mr-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/50 sm:hidden">
                Duration
              </span>
              <span className="text-sm tabular-nums text-muted-foreground">
                {outage.resolved_at
                  ? formatAlertDuration(outage.triggered_at, outage.resolved_at)
                  : "—"}
              </span>
            </div>
            <div>
              <span
                className={cn(
                  "rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide",
                  severityBadgeClass(outage.severity),
                )}
              >
                {severityLabel(outage.severity)}
              </span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export function OutageHistoryPagination({
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
    <div className="flex flex-col items-center gap-2 pt-1">
      {isFetchNextPageError ? (
        <p className="text-xs text-destructive" role="alert">
          Older outages couldn’t be loaded; existing outages are still shown.
        </p>
      ) : null}
      <Button
        variant="outline"
        size="sm"
        className="h-8 text-xs"
        disabled={isFetchingNextPage}
        onClick={onLoadMore}
      >
        {isFetchingNextPage ? (
          <>
            <LoaderCircleIcon className="size-3 animate-spin" />
            Loading older outages
          </>
        ) : isFetchNextPageError ? (
          "Try again"
        ) : (
          "Load older outages"
        )}
      </Button>
    </div>
  );
}

function OutageHistoryLoadingState() {
  return (
    <div className="overflow-hidden rounded-lg border border-border" aria-label="Loading outages">
      {Array.from({ length: 3 }).map((_, index) => (
        <div
          key={index}
          className="grid grid-cols-4 gap-4 border-b border-border/70 px-4 py-3 last:border-0"
        >
          <Skeleton className="h-3.5 w-24" />
          <Skeleton className="h-3.5 w-24" />
          <Skeleton className="h-3.5 w-14" />
          <Skeleton className="h-5 w-16 rounded-full" />
        </div>
      ))}
    </div>
  );
}

function NoOutagesRecorded() {
  return (
    <div className="rounded-lg border border-dashed border-border px-4 py-4">
      <p className="text-sm font-medium text-foreground">No outages recorded</p>
      <p className="mt-0.5 text-xs text-muted-foreground">
        This service has no recorded service-unhealthy alerts.
      </p>
    </div>
  );
}

export function OutageAlertingNotConfigured({ canManageRules }: { canManageRules: boolean }) {
  return (
    <div className="flex flex-col gap-3 rounded-lg border border-dashed border-border bg-muted/10 px-4 py-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-start gap-2.5">
        <TriangleAlertIcon className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
        <div>
          <p className="text-sm font-medium text-foreground">
            Outage alerting isn’t configured for this service.
          </p>
          <p className="mt-0.5 text-xs text-muted-foreground">
            Add an enabled Service Unhealthy rule to record future outages.
          </p>
        </div>
      </div>
      <Button asChild variant="outline" size="sm" className="h-8 shrink-0 text-xs">
        <Link to="/settings?tab=alert-rules">
          {canManageRules ? "Configure rule" : "View alert rules"}
        </Link>
      </Button>
    </div>
  );
}

type OutageHistoryState = {
  status: "pending" | "error" | "success";
  nextPageStatus: "idle" | "loading" | "error";
  hasNextPage: boolean;
};

type OutageRuleCoverageState =
  | { status: "pending" }
  | { status: "error" }
  | { status: "success"; covered: boolean };

export type ServiceOutagesContentProps = {
  currentOutages: readonly ServiceAlert[];
  resolvedOutages: readonly Alert[];
  history: OutageHistoryState;
  ruleCoverage: OutageRuleCoverageState;
  canManageRules: boolean;
  onLoadMore: () => void;
  onRetryHistory: () => void;
  onRetryRules: () => void;
  now?: number;
};

export function ServiceOutagesContent({
  currentOutages,
  resolvedOutages,
  history,
  ruleCoverage,
  canManageRules,
  onLoadMore,
  onRetryHistory,
  onRetryRules,
  now,
}: ServiceOutagesContentProps) {
  const liveNow = useOutageClock(currentOutages);
  const displayNow = now ?? liveNow;

  return (
    <section>
      <SectionHeading
        icon={<ClockIcon size={13} />}
        label="Outages"
        aside={
          currentOutages.length > 0 ? (
            <span className="text-xs font-semibold text-destructive">
              {currentOutages.length} ongoing
            </span>
          ) : undefined
        }
      />
      <div className="space-y-3">
        <CurrentOutages outages={currentOutages} now={displayNow} />

        {history.status === "pending" ? (
          <OutageHistoryLoadingState />
        ) : history.status === "error" && resolvedOutages.length === 0 ? (
          <div className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-4">
            <p className="text-sm text-destructive" role="alert">
              Couldn’t load outage history.
            </p>
            <Button
              variant="ghost"
              size="sm"
              className="mt-2 -ml-3 h-7 text-xs"
              onClick={onRetryHistory}
            >
              Try again
            </Button>
          </div>
        ) : resolvedOutages.length > 0 ? (
          <>
            <ResolvedOutagesTable outages={resolvedOutages} />
            <OutageHistoryPagination
              hasNextPage={history.hasNextPage}
              isFetchingNextPage={history.nextPageStatus === "loading"}
              isFetchNextPageError={history.nextPageStatus === "error"}
              onLoadMore={onLoadMore}
            />
          </>
        ) : currentOutages.length > 0 ? (
          <p className="px-1 text-xs text-muted-foreground">No previous outages recorded.</p>
        ) : ruleCoverage.status === "pending" ? (
          <Skeleton className="h-20 w-full rounded-lg" />
        ) : ruleCoverage.status === "success" && !ruleCoverage.covered ? (
          <OutageAlertingNotConfigured canManageRules={canManageRules} />
        ) : (
          <>
            <NoOutagesRecorded />
            {ruleCoverage.status === "error" ? (
              <div className="flex items-center gap-2 px-1 text-xs text-muted-foreground">
                <span>Couldn’t verify outage alert rule coverage.</span>
                <button
                  type="button"
                  className="font-medium text-foreground hover:underline"
                  onClick={onRetryRules}
                >
                  Try again
                </button>
              </div>
            ) : null}
          </>
        )}

        {resolvedOutages.length > 0 && history.status === "error" ? (
          <div className="flex items-center gap-2 px-1 text-xs text-destructive" role="alert">
            <span>Outage history refresh failed. Showing saved outages.</span>
            <button type="button" className="font-medium hover:underline" onClick={onRetryHistory}>
              Try again
            </button>
          </div>
        ) : null}
      </div>
    </section>
  );
}

export function ServiceOutagesSection({
  projectId,
  serviceId,
  activeAlerts,
  serviceDataUpdatedAt,
}: {
  projectId: number;
  serviceId: number;
  activeAlerts: readonly ServiceAlert[];
  serviceDataUpdatedAt: number;
}) {
  const filters = useMemo(() => serviceOutageHistoryFilters(serviceId), [serviceId]);
  const currentOutages = useMemo(() => serviceOutageAlerts(activeAlerts), [activeAlerts]);
  const historyQuery = useAlertHistory(projectId, filters, SERVICE_OUTAGE_HISTORY_LIMIT);
  const rulesQuery = useAlertRules(projectId);
  const canManageRules = useCanWrite(projectId);
  useAlertHistoryIdentityHeadSync(
    projectId,
    filters,
    currentOutages,
    serviceDataUpdatedAt,
    SERVICE_OUTAGE_HISTORY_LIMIT,
  );

  const outages = useMemo(
    () =>
      resolvedServiceOutages(
        historyQuery.data?.pages.flatMap((page) => page.alerts) ?? [],
        serviceId,
      ),
    [historyQuery.data, serviceId],
  );
  const ruleCoverage = rulesQuery.data
    ? hasApplicableServiceOutageRule(rulesQuery.data.alert_rules, serviceId)
    : undefined;
  const contentKey = `${serviceId}:${currentOutages
    .map((outage) => `${outage.id}:${outage.triggered_at}`)
    .join(",")}`;

  return (
    <ServiceOutagesContent
      key={contentKey}
      currentOutages={currentOutages}
      resolvedOutages={outages}
      history={{
        status: historyQuery.isPending ? "pending" : historyQuery.isError ? "error" : "success",
        nextPageStatus: historyQuery.isFetchingNextPage
          ? "loading"
          : historyQuery.isFetchNextPageError
            ? "error"
            : "idle",
        hasNextPage: historyQuery.hasNextPage,
      }}
      ruleCoverage={
        ruleCoverage != null
          ? { status: "success", covered: ruleCoverage }
          : rulesQuery.isPending
            ? { status: "pending" }
            : { status: "error" }
      }
      canManageRules={canManageRules}
      onLoadMore={() => void historyQuery.fetchNextPage()}
      onRetryHistory={() => void historyQuery.refetch()}
      onRetryRules={() => void rulesQuery.refetch()}
    />
  );
}
