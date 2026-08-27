export function formatAlertDuration(triggeredAt: string, resolvedAt: string): string {
  const triggered = Date.parse(triggeredAt);
  const resolved = Date.parse(resolvedAt);
  if (!Number.isFinite(triggered) || !Number.isFinite(resolved)) return "—";

  const totalSeconds = Math.max(0, Math.floor((resolved - triggered) / 1_000));
  if (totalSeconds < 60) return `${totalSeconds}s`;

  const totalMinutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (totalMinutes < 60) {
    return seconds === 0 ? `${totalMinutes}m` : `${totalMinutes}m ${seconds}s`;
  }

  const totalHours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  if (totalHours < 24) {
    return minutes === 0 ? `${totalHours}h` : `${totalHours}h ${minutes}m`;
  }

  const days = Math.floor(totalHours / 24);
  const hours = totalHours % 24;
  return hours === 0 ? `${days}d` : `${days}d ${hours}h`;
}
