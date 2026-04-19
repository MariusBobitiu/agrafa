import type { ReactNode } from "react";
import { cn } from "@/lib/utils.ts";

type MetaItemProps = {
  label: string;
  value: ReactNode;
  valueClass?: string;
};

export function MetaItem({ label, value, valueClass }: MetaItemProps) {
  return (
    <div className="flex items-baseline gap-1.5">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className={cn("text-xs font-medium text-foreground", valueClass)}>{value}</span>
    </div>
  );
}
