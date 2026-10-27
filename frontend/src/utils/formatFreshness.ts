export function formatFreshness(seconds: number | null | undefined): string {
  if (seconds == null || !Number.isFinite(seconds)) return '—';
  const elapsed = Math.max(0, Math.floor(seconds));
  if (elapsed < 60) return `${elapsed}s`;
  if (elapsed < 3600) return `${Math.floor(elapsed / 60)}m ${elapsed % 60}s`;
  if (elapsed < 86400) return `${Math.floor(elapsed / 3600)}h ${Math.floor((elapsed % 3600) / 60)}m`;
  return `${Math.floor(elapsed / 86400)}d ${Math.floor((elapsed % 86400) / 3600)}h`;
}
