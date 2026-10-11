import type { MaterializedView, MaterializedViewsResponse } from './schema';

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function isMaterializedView(value: unknown): value is MaterializedView {
  if (!isRecord(value)) return false;
  const stringFields = [
    'id', 'database', 'name', 'refreshType', 'refreshTrigger', 'refreshPolicy',
    'active', 'inactiveReason', 'lastRefreshState', 'lastRefreshFinished',
    'lastSuccessfulRefreshFinished', 'lastRefreshError', 'definition',
  ];
  return stringFields.every(field => typeof value[field] === 'string') &&
    (value.rows === null || typeof value.rows === 'number') &&
    (value.lastRefreshDuration === null || typeof value.lastRefreshDuration === 'number');
}

export function isMaterializedViewsResponse(value: unknown): value is MaterializedViewsResponse {
  return isRecord(value) && Array.isArray(value.views) && value.views.every(isMaterializedView);
}
