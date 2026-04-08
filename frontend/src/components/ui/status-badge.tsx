import { cn } from "@/lib/utils.ts";
import { Badge } from "./badge.tsx";

type Status =
  | "online"
  | "offline"
  | "unknown"
  | "healthy"
  | "degraded"
  | "unhealthy"
  | "active"
  | "resolved";

const STATUS_STYLES: Record<Status, string> = {
  online: "bg-primary/10 text-lime-600 dark:text-lime-400 border-primary/20",
  healthy: "bg-primary/10 text-lime-600 dark:text-lime-400 border-primary/20",
  resolved: "bg-primary/10 text-lime-600 dark:text-lime-400 border-primary/20",
  degraded: "bg-warning/10 text-yellow-600 dark:text-yellow-400 border-warning/20",
  offline: "bg-destructive/10 text-red-600 dark:text-red-400 border-destructive/20",
  unhealthy: "bg-destructive/10 text-red-600 dark:text-red-400 border-destructive/20",
  active: "bg-destructive/10 text-red-600 dark:text-red-400 border-destructive/20",
  unknown: "bg-muted text-muted-foreground border-muted-foreground/20",
};

export function StatusBadge({ status }: { status: Status }) {
  return (
    <Badge
      variant="outline"
      className={cn("capitalize text-xs font-medium", STATUS_STYLES[status])}
    >
      {status}
    </Badge>
  );
}
