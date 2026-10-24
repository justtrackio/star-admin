import type { DictionariesResponse, RoutineLoadsResponse, TablesResponse } from './schema';

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

export function isTablesResponse(value: unknown): value is TablesResponse {
  return isRecord(value) && Array.isArray(value.tables) && value.tables.every((table: unknown) =>
    isRecord(table) &&
    ['database', 'name', 'engine'].every(field => typeof table[field] === 'string') &&
    (table.rows === null || typeof table.rows === 'number') &&
    (table.dataBytes === null || typeof table.dataBytes === 'number'));
}

export function isRoutineLoadsResponse(value: unknown): value is RoutineLoadsResponse {
  return isRecord(value) && Array.isArray(value.jobs) && value.jobs.every((job: unknown) =>
    isRecord(job) &&
    ['id', 'database', 'table', 'name', 'state', 'topic', 'loadedRows', 'errorRows', 'rowsPerSecond', 'progress', 'reason']
      .every(field => typeof job[field] === 'string'));
}

export function isDictionariesResponse(value: unknown): value is DictionariesResponse {
  return isRecord(value) && Array.isArray(value.dictionaries) && value.dictionaries.every((dictionary: unknown) =>
    isRecord(dictionary) &&
    ['id', 'name', 'catalog', 'database', 'source', 'keys', 'values', 'status', 'lastRefreshStarted', 'lastRefreshFinished', 'nextRefresh', 'error', 'memoryUsage']
      .every(field => typeof dictionary[field] === 'string'));
}
