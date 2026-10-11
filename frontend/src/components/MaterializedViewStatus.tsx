import { Tag, Tooltip } from 'antd';

const refreshDescriptions: Record<string, string> = {
  PENDING: 'The refresh is queued and waiting to run.',
  RUNNING: 'The refresh is currently running.',
  SUCCESS: 'The refresh completed successfully.',
  FAILED: 'The refresh failed. See the error message for details.',
  SKIPPED: 'No changes were detected in the relevant base-table partitions, so refreshing them was unnecessary. This is a normal outcome.',
  MERGED: 'This refresh request was combined with another pending refresh request.',
};

const refreshColors: Record<string, string> = {
  PENDING: 'blue',
  RUNNING: 'yellow',
  SUCCESS: 'green',
  FAILED: 'red',
  SKIPPED: 'default',
  MERGED: 'purple',
};

export function MaterializedViewStatus({ active, reason }: { active: string; reason: string }) {
  const normalized = active.toLowerCase();
  if (normalized !== 'true' && normalized !== 'false') return '—';

  const isActive = normalized === 'true';
  const description = isActive
    ? 'The materialized view is active and available for queries and refreshes. This does not indicate its data freshness.'
    : 'The materialized view is inactive and cannot be queried or refreshed.';

  return (
    <Tooltip title={`${description}${!isActive && reason ? ` Reason: ${reason}` : ''}`} trigger={['hover', 'focus']}>
      <Tag color={isActive ? 'green' : 'red'} tabIndex={0}>{isActive ? 'Active' : 'Inactive'}</Tag>
    </Tooltip>
  );
}

export function MaterializedViewRefreshStatus({ state, error }: { state: string; error: string }) {
  if (!state) return '—';

  const description = refreshDescriptions[state] ?? `StarRocks reported the refresh state ${state}.`;

  return (
    <Tooltip title={`${description}${error ? ` Last refresh error: ${error}` : ''}`} trigger={['hover', 'focus']}>
      <Tag color={refreshColors[state] ?? 'orange'} tabIndex={0}>{state}</Tag>
    </Tooltip>
  );
}
