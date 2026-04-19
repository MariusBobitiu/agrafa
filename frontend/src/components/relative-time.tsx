import { useRelativeTimeTicker } from "@/hooks/use-relative-time-ticker.ts";
import { formatRelativeTime } from "@/lib/utils.ts";

type RelativeTimeProps = {
  value: string | Date | null | undefined;
  prefix?: string;
  fallback?: string;
};

export function RelativeTime({ value, prefix, fallback = "—" }: RelativeTimeProps) {
  const now = useRelativeTimeTicker();

  if (!value) {
    return <>{fallback}</>;
  }

  const text = formatRelativeTime(value, now);
  return <>{prefix ? `${prefix} ${text}` : text}</>;
}
