function formatDurationMilliseconds(durationMilliseconds: number): string {
  if (!Number.isFinite(durationMilliseconds)) return "—";

  const totalSeconds = Math.max(0, Math.floor(durationMilliseconds / 1_000));
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

export function formatAlertDuration(triggeredAt: string, resolvedAt: string): string {
  const triggered = Date.parse(triggeredAt);
  const resolved = Date.parse(resolvedAt);
  if (!Number.isFinite(triggered) || !Number.isFinite(resolved)) return "—";

  return formatDurationMilliseconds(resolved - triggered);
}

export function formatOngoingAlertDuration(triggeredAt: string, now = Date.now()): string {
  const triggered = Date.parse(triggeredAt);
  if (!Number.isFinite(triggered)) return "—";

  return formatDurationMilliseconds(now - triggered);
}
