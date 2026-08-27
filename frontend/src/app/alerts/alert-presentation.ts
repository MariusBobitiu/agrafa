import type { Alert, RuleType } from "@/types/alert.ts";
import { cn } from "@/lib/utils.ts";

export function historyFilterTriggerClass(selected: boolean, className?: string) {
  return cn(
    "h-8 w-auto min-w-32 border-input bg-background text-xs transition-[border-color,opacity]",
    selected ? "border-solid opacity-100" : "border-dashed opacity-90",
    className,
  );
}

export function ruleTypeLabel(ruleType: RuleType) {
  switch (ruleType) {
    case "node_offline":
      return "Node offline";
    case "service_unhealthy":
      return "Service unhealthy";
    case "cpu_above_threshold":
      return "CPU threshold";
    case "memory_above_threshold":
      return "Memory threshold";
    case "disk_above_threshold":
      return "Disk threshold";
  }
}

export function alertResourceLabel(alert: Alert) {
  if (alert.service_id != null) return alert.service_name ?? `Service #${alert.service_id}`;
  if (alert.node_id != null) {
    return alert.node_name ?? alert.node_identifier ?? `Node #${alert.node_id}`;
  }
  return "Resource unavailable";
}

export function deduplicateAlerts(alerts: Alert[]) {
  const seen = new Set<number>();
  return alerts.filter((alert) => {
    if (seen.has(alert.id)) return false;
    seen.add(alert.id);
    return true;
  });
}
