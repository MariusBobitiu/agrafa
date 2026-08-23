import { CheckCircle2Icon, XCircleIcon } from "lucide-react";
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip as ChartTooltip,
  XAxis,
  YAxis,
} from "recharts";
import { RelativeTime } from "@/components/relative-time.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip.tsx";
import { formatDate, cn } from "@/lib/utils.ts";
import type { ServiceHistoryObservation } from "@/types/service.ts";
import {
  formatLatency,
  getHistoryRowPresentation,
  type ServiceHistoryRangeHours,
} from "../service-history-utils.ts";

type ChartPoint = {
  id: number;
  timestamp: number;
  successLatency: number | null;
  failureLatency: number | null;
  observation: ServiceHistoryObservation;
};

function buildChartData(observations: ServiceHistoryObservation[]): ChartPoint[] {
  return [...observations]
    .sort((a, b) => Date.parse(a.observedAt) - Date.parse(b.observedAt))
    .map((observation) => ({
      id: observation.id,
      timestamp: Date.parse(observation.observedAt),
      successLatency: observation.isSuccess ? observation.latencyMs : null,
      failureLatency: observation.isSuccess ? null : (observation.latencyMs ?? 0),
      observation,
    }));
}

function tickTime(timestamp: number, rangeHours: ServiceHistoryRangeHours): string {
  return new Intl.DateTimeFormat(
    "en",
    rangeHours > 24 ? { month: "short", day: "numeric" } : { hour: "numeric", minute: "2-digit" },
  ).format(timestamp);
}

function LatencyTooltip({
  active,
  payload,
}: {
  active?: boolean;
  payload?: ReadonlyArray<{ payload: ChartPoint }>;
}) {
  const observation = payload?.[0]?.payload.observation;
  if (!active || !observation) return null;

  const presentation = getHistoryRowPresentation(observation);
  return (
    <div className="rounded-md border border-border bg-popover px-3 py-2 text-xs shadow-md">
      <p className="font-medium text-popover-foreground">{presentation.result}</p>
      <p className="mt-1 text-muted-foreground">
        {formatLatency(observation.latencyMs)} · {formatDate(observation.observedAt)}
      </p>
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
  const chartData = buildChartData(observations);
  const hasChartData = observations.some(
    (observation) => observation.latencyMs != null || !observation.isSuccess,
  );

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
        <LineChart data={chartData} margin={{ top: 10, right: 8, bottom: 0, left: -18 }}>
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
            content={<LatencyTooltip />}
            cursor={{ stroke: "var(--muted-foreground)", strokeDasharray: "3 3", opacity: 0.35 }}
          />
          <Line
            type="monotone"
            dataKey="successLatency"
            stroke="var(--primary)"
            strokeWidth={1.75}
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
              r: 3,
              fill: "var(--destructive)",
              stroke: "var(--background)",
              strokeWidth: 1.5,
            }}
            activeDot={{
              r: 4,
              fill: "var(--destructive)",
              stroke: "var(--background)",
              strokeWidth: 2,
            }}
            isAnimationActive={false}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

function ObservationTooltip({ observation }: { observation: ServiceHistoryObservation }) {
  const presentation = getHistoryRowPresentation(observation);
  return (
    <div className="space-y-0.5 text-xs">
      <p className="font-medium">{formatDate(observation.observedAt)}</p>
      <p>
        {observation.isSuccess ? "Success" : "Failed"}: {presentation.result}
      </p>
      <p>Latency: {presentation.latency}</p>
      {observation.checkType === "http" && observation.statusCode != null ? (
        <p>HTTP status: {observation.statusCode}</p>
      ) : null}
    </div>
  );
}

export function RecentCheckStrip({ observations }: { observations: ServiceHistoryObservation[] }) {
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
            <ObservationTooltip observation={observation} />
          </TooltipContent>
        </Tooltip>
      ))}
    </div>
  );
}

export function ServiceHistoryRow({ observation }: { observation: ServiceHistoryObservation }) {
  const presentation = getHistoryRowPresentation(observation);

  return (
    <div className="flex items-stretch">
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
    <div className="space-y-2" aria-label="Loading check history">
      {Array.from({ length: rows }).map((_, index) => (
        <Skeleton key={index} className="h-10 w-full" />
      ))}
    </div>
  );
}
