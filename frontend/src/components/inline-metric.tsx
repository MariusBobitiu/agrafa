import { cn } from "@/lib/utils.ts";

type InlineMetricProps = {
  icon: React.ReactNode;
  label: string;
  value: number;
  variant: "cpu" | "mem" | "disk";
};

export function InlineMetric({ icon, label, value }: InlineMetricProps) {
  const barColor =
    value > 90
      ? "bg-destructive"
      : value > 70
        ? "bg-warning"
        : "bg-primary/90";

  const textColor =
    value > 90 ? "text-destructive" : value > 70 ? "text-warning" : "text-primary/90";

  return (
    <div className="flex flex-col gap-1 w-16">
      <div className="flex items-center justify-between gap-1">
        <span className="flex items-center gap-1 text-muted-foreground/60">
          {icon}
          <span className="text-[10px] leading-none">{label}</span>
        </span>
        <span className={cn("text-[10px] tabular-nums font-medium leading-none", textColor)}>
          {value.toFixed(0)}%
        </span>
      </div>
      <div className="h-1 rounded-full bg-muted overflow-hidden">
        <div
          className={cn("h-full rounded-full transition-all", barColor)}
          style={{ width: `${Math.min(value, 100)}%` }}
        />
      </div>
    </div>
  );
}
