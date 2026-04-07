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
  online: "bg-lime-500/10 text-lime-600 dark:text-lime-400 border-lime-500/20",
  healthy: "bg-lime-500/10 text-lime-600 dark:text-lime-400 border-lime-500/20",
  resolved: "bg-lime-500/10 text-lime-600 dark:text-lime-400 border-lime-500/20",
  degraded: "bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 border-yellow-500/20",
  offline: "bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20",
  unhealthy: "bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20",
  active: "bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20",
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
