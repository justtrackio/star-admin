import type { MaterializedView } from '../api/schema';

export function formatRefreshSchedule(
  view: Pick<MaterializedView, 'refreshType' | 'refreshTrigger' | 'refreshPolicy'>,
): string {
  if (view.refreshType === 'SYNC') return 'With base-table updates (synchronous)';
  if (view.refreshTrigger === 'MANUAL' || view.refreshPolicy === 'MANUAL') return 'Manual';
  if (view.refreshTrigger === 'ON_BASE_TABLE_CHANGE' || view.refreshPolicy === 'ON_BASE_TABLE_CHANGE')
    return 'When base tables change';

  const interval = /\bEVERY\s*\(\s*INTERVAL\s+(\d+)\s+([A-Z]+)\s*\)/i.exec(view.refreshPolicy);
  if (interval) {
    const amount = Number(interval[1]);
    const unit = interval[2].toLowerCase();
    return `Every ${amount} ${unit}${amount === 1 ? '' : 's'}`;
  }

  if (view.refreshPolicy && view.refreshPolicy !== 'NONE') return view.refreshPolicy;
  return view.refreshTrigger === 'SCHEDULED' ? 'Scheduled (interval unavailable)' : '—';
}
