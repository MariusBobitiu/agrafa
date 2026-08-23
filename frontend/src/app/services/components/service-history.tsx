import { useId } from "react";
import { CheckCircle2Icon, ClockIcon, XCircleIcon } from "lucide-react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  Line,
  ResponsiveContainer,
  Tooltip as ChartTooltip,
  XAxis,
  YAxis,
} from "recharts";
import { RelativeTime } from "@/components/relative-time.tsx";
import { SectionHeading } from "@/components/section-heading.tsx";
import { Button } from "@/components/ui/button.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip.tsx";
import { formatDate, cn } from "@/lib/utils.ts";
import type { ServiceHistoryObservation, ServiceHistorySummary } from "@/types/service.ts";
import {
  buildHistoryChartData,
  formatLatency,
  formatObservationTooltipTime,
  formatUptime,
  getHistoryRowPresentation,
  type ServiceHistoryChartPoint,
  type ServiceHistoryRangeHours,
} from "../service-history-utils.ts";

export function ServiceHistorySummaryMetrics({ summary }: { summary: ServiceHistorySummary }) {
  return (
    <div className="grid grid-cols-3 divide-x divide-border rounded-lg border border-border bg-card">
      <div className="px-4 py-3">
        <p className="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">
          Uptime
        </p>
        <p className="mt-1 text-lg font-semibold tabular-nums text-foreground">
          {formatUptime(summary.uptimePercent)}
        </p>
      </div>
      <div className="px-4 py-3">
        <p className="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">
          Avg latency
        </p>
        <p className="mt-1 text-lg font-semibold tabular-nums text-foreground">
          {formatLatency(summary.averageLatencyMs)}
        </p>
      </div>
      <div className="px-4 py-3">
        <p className="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">
          Last checked
        </p>
        <p className="mt-1 text-lg font-semibold tabular-nums text-foreground">
          <RelativeTime value={summary.lastCheckedAt} />
        </p>
      </div>
    </div>
  );
}

export function ServiceHistorySuccessCount({ summary }: { summary: ServiceHistorySummary }) {
  return (
    <p className="text-sm text-muted-foreground">
      {summary.successfulChecks} of {summary.totalChecks} checks successful
    </p>
  );
}

function tickTime(timestamp: number, rangeHours: ServiceHistoryRangeHours): string {
  return new Intl.DateTimeFormat(
    "en",
    rangeHours > 24 ? { month: "short", day: "numeric" } : { hour: "numeric", minute: "2-digit" },
  ).format(timestamp);
}

export function ObservationTooltip({
  observation,
  rangeHours,
}: {
  observation: ServiceHistoryObservation;
  rangeHours?: ServiceHistoryRangeHours;
}) {
  const presentation = getHistoryRowPresentation(observation);
  return (
    <div className="space-y-0.5 text-xs tabular-nums">
      <p className="text-muted-foreground">
        {formatObservationTooltipTime(observation.observedAt, rangeHours)}
      </p>
      <p
        className={cn(
          "font-medium",
          observation.isSuccess ? "text-popover-foreground" : "text-destructive",
        )}
      >
        {presentation.result}
      </p>
      <p className="text-muted-foreground">{presentation.latency}</p>
    </div>
  );
}

function LatencyTooltip({
  active,
  payload,
  rangeHours,
}: {
  active?: boolean;
  payload?: ReadonlyArray<{ payload: ServiceHistoryChartPoint }>;
  rangeHours: ServiceHistoryRangeHours;
}) {
  const observation = payload?.[0]?.payload.observation;
  if (!active || !observation) return null;

  return (
    <div className="rounded-md border border-border bg-popover px-3 py-2 text-xs shadow-md">
      <ObservationTooltip observation={observation} rangeHours={rangeHours} />
    </div>
  );
}

export function LatencyChart({
  observations,
  rangeHours,
}: {
  observations: ServiceHistoryObservation[];
  rangeHours: ServiceHistoryRangeHours;
}) {
  const gradientId = `latency-fill-${useId().replaceAll(":", "")}`;
  const chartData = buildHistoryChartData(observations);
  const hasChartData = observations.some((observation) => observation.latencyMs != null);

  if (!hasChartData) {
    return (
      <div className="flex h-48 items-center justify-center rounded-lg border border-dashed border-border">
        <p className="text-sm text-muted-foreground">No latency measurements in this range.</p>
      </div>
    );
  }

  return (
    <div className="h-56 w-full" aria-label="Latency over time">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={chartData} margin={{ top: 10, right: 8, bottom: 0, left: -18 }}>
          <defs>
            <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="var(--primary)" stopOpacity={0.6} />
              <stop offset="45%" stopColor="var(--primary)" stopOpacity={0.3} />
              <stop offset="100%" stopColor="var(--primary)" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid vertical={false} stroke="var(--border)" strokeDasharray="3 5" />
          <XAxis
            dataKey="timestamp"
            type="number"
            domain={["dataMin", "dataMax"]}
            tickFormatter={(value: number) => tickTime(value, rangeHours)}
            axisLine={false}
            tickLine={false}
            minTickGap={40}
            tick={{ fill: "var(--muted-foreground)", fontSize: 10 }}
          />
          <YAxis
            width={48}
            axisLine={false}
            tickLine={false}
            tickFormatter={(value: number) => `${value} ms`}
            tick={{ fill: "var(--muted-foreground)", fontSize: 10 }}
            domain={[0, "auto"]}
          />
          <ChartTooltip
            content={<LatencyTooltip rangeHours={rangeHours} />}
            cursor={{ stroke: "var(--muted-foreground)", strokeDasharray: "3 3", opacity: 0.35 }}
          />
          <Area
            type="linear"
            dataKey="successLatency"
            stroke="var(--primary)"
            strokeWidth={1.75}
            fill={`url(#${gradientId})`}
            fillOpacity={1}
            connectNulls={false}
            dot={{ r: 2, fill: "var(--primary)", strokeWidth: 0 }}
            activeDot={{
              r: 4,
              fill: "var(--primary)",
              stroke: "var(--background)",
              strokeWidth: 2,
            }}
            isAnimationActive={false}
          />
          <Line
            type="linear"
            dataKey="failureLatency"
            stroke="transparent"
            connectNulls={false}
            dot={{
              r: 4,
              fill: "var(--destructive)",
              stroke: "var(--background)",
              strokeWidth: 2,
            }}
            activeDot={{
              r: 4,
              fill: "var(--destructive)",
              stroke: "var(--background)",
              strokeWidth: 2,
            }}
            isAnimationActive={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

export function RecentCheckStrip({
  observations,
  rangeHours,
}: {
  observations: ServiceHistoryObservation[];
  rangeHours?: ServiceHistoryRangeHours;
}) {
  const recent = observations.slice(0, 24).reverse();

  return (
    <div className="flex items-center justify-end gap-1" aria-label="Recent check results">
      {recent.map((observation, index) => (
        <Tooltip key={observation.id} delayDuration={100}>
          <TooltipTrigger asChild>
            <button
              type="button"
              aria-label={`${observation.isSuccess ? "Successful" : "Failed"} check at ${formatDate(observation.observedAt)}`}
              className={cn(
                "size-3 rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                index < Math.max(0, recent.length - 16) && "hidden sm:inline-block",
                observation.isSuccess ? "bg-primary" : "bg-destructive",
              )}
            />
          </TooltipTrigger>
          <TooltipContent side="top">
            <ObservationTooltip observation={observation} rangeHours={rangeHours} />
          </TooltipContent>
        </Tooltip>
      ))}
    </div>
  );
}

export function ServiceHistoryRow({ observation }: { observation: ServiceHistoryObservation }) {
  const presentation = getHistoryRowPresentation(observation);

  return (
    <div className="flex items-stretch" data-history-observation-id={observation.id}>
      <div
        className={cn("w-0.5 shrink-0", observation.isSuccess ? "bg-primary" : "bg-destructive")}
      />
      <div className="flex min-w-0 flex-1 items-center gap-3 px-4 py-3">
        {observation.isSuccess ? (
          <CheckCircle2Icon size={14} className="shrink-0 text-primary" />
        ) : (
          <XCircleIcon size={14} className="shrink-0 text-destructive" />
        )}
        <p
          title={presentation.result}
          className={cn(
            "min-w-0 flex-1 truncate text-sm font-medium",
            observation.isSuccess ? "text-foreground" : "text-destructive",
          )}
        >
          {presentation.result}
        </p>
        <div className="ml-2 flex shrink-0 items-center gap-4 text-xs text-muted-foreground tabular-nums">
          <span>{presentation.latency}</span>
          <span className="w-20 text-right sm:w-24">
            <RelativeTime value={observation.observedAt} />
          </span>
        </div>
      </div>
    </div>
  );
}

export function HistoryLoadingState({ rows = 3 }: { rows?: number }) {
  return (
    <div
      className="max-h-[480px] overflow-hidden rounded-lg border border-border bg-card divide-y divide-border"
      aria-label="Loading recent checks"
    >
      {Array.from({ length: rows }).map((_, index) => (
        <div key={index} className="flex h-[45px] items-center gap-3 px-4">
          <Skeleton className="size-3.5 shrink-0 rounded-full" />
          <Skeleton className="h-3.5 w-36 max-w-[45%]" />
          <Skeleton className="ml-auto h-3 w-10" />
          <Skeleton className="h-3 w-20" />
        </div>
      ))}
    </div>
  );
}

export function ServiceHealthLoadingState() {
  return (
    <div className="space-y-5" aria-label="Loading service health">
      <div className="flex flex-col gap-5 md:flex-row md:items-start md:justify-between">
        <div className="space-y-2">
          <Skeleton className="h-9 w-36" />
          <Skeleton className="h-4 w-44" />
        </div>
        <div className="space-y-2 md:max-w-[55%]">
          <Skeleton className="ml-auto h-3 w-24" />
          <div className="flex justify-end gap-1">
            {Array.from({ length: 16 }).map((_, index) => (
              <Skeleton key={index} className="size-3 rounded-full" />
            ))}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-3 divide-x divide-border rounded-lg border border-border bg-card">
        {Array.from({ length: 3 }).map((_, index) => (
          <div key={index} className="space-y-2 px-4 py-3">
            <Skeleton className="h-3 w-20 max-w-full" />
            <Skeleton className="h-6 w-24 max-w-full" />
          </div>
        ))}
      </div>

      <Skeleton className="h-56 w-full rounded-lg" />
    </div>
  );
}

export function RecentChecksList({ observations }: { observations: ServiceHistoryObservation[] }) {
  return (
    <div
      className="max-h-[480px] overflow-y-auto overscroll-contain rounded-lg border border-border bg-card divide-y divide-border"
      aria-label="Recent checks history"
      tabIndex={0}
    >
      {observations.map((observation) => (
        <ServiceHistoryRow key={observation.id} observation={observation} />
      ))}
    </div>
  );
}

export function RecentChecksHeading() {
  return <SectionHeading icon={<ClockIcon size={13} />} label="Recent Checks" />;
}

export function ServiceHistoryRefreshError({ onRetry }: { onRetry: () => void }) {
  return (
    <div
      className="flex flex-wrap items-center gap-x-2 gap-y-1 rounded-md border border-destructive/20 bg-destructive/5 px-3 py-2"
      role="status"
    >
      <p className="text-xs text-muted-foreground">History refresh failed. Showing saved data.</p>
      <Button variant="ghost" size="sm" className="h-6 px-2 text-xs" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}

export function ServiceHistoryEmptyState({
  refreshError,
  onRetry,
}: {
  refreshError: boolean;
  onRetry: () => void;
}) {
  return (
    <div className="space-y-3">
      {refreshError ? <ServiceHistoryRefreshError onRetry={onRetry} /> : null}
      <div className="rounded-lg border border-dashed border-border px-4 py-8 text-center">
        <p className="text-sm text-muted-foreground">No check history in this range.</p>
      </div>
    </div>
  );
}

export function RecentChecksPagination({
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
    <div className="flex flex-col items-center gap-1.5">
      {isFetchNextPageError ? (
        <p className="text-xs text-destructive" role="alert">
          Couldn’t load older checks.
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
          ? "Loading older checks…"
          : isFetchNextPageError
            ? "Try again"
            : "Load older checks"}
      </Button>
    </div>
  );
}
