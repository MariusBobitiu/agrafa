import { cn } from "@/lib/utils.ts";

type MetricBarProps = {
  icon: React.ReactNode;
  label: string;
  value: number;
  variant: "cpu" | "mem" | "disk";
};

export function MetricBar({ icon, label, value }: MetricBarProps) {
  const trackColor =
    value > 90
      ? "bg-destructive"
      : value > 70
        ? "bg-warning"
        : "bg-primary/90";

  const textColor =
    value > 90 ? "text-destructive" : value > 70 ? "text-warning" : "text-primary/90";

  return (
    <div className="flex items-center gap-2">
      <span className="text-muted-foreground shrink-0">{icon}</span>
      <span className="text-xs text-muted-foreground w-8 shrink-0">{label}</span>
      <div className="flex-1 h-1.5 rounded-full bg-muted overflow-hidden">
        <div
          className={cn("h-full rounded-full transition-all", trackColor)}
          style={{ width: `${Math.min(value, 100)}%` }}
        />
      </div>
      <span className={cn("text-xs tabular-nums font-medium w-10 text-right shrink-0", textColor)}>
        {value.toFixed(1)}%
      </span>
    </div>
  );
}
