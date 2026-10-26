import type { DictionariesResponse, FrontendConfigResponse, ResourceGroupsResponse, NodesResponse, RoutineLoadsResponse, RoutineLoadSettings, RoutineLoadTasksResponse, TablesResponse } from './schema';

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
    ['id', 'database', 'table', 'name', 'state', 'topic', 'loadedRows', 'errorRows', 'rowsPerSecond', 'progress', 'reason', 'taskConsumeSecond']
      .every(field => typeof job[field] === 'string'));
}

export function isRoutineLoadSettings(value: unknown): value is RoutineLoadSettings {
  return isRecord(value) && typeof value.maxBatchSize === 'string' && typeof value.taskConsumeSecond === 'string';
}

export function isRoutineLoadTasksResponse(value: unknown): value is RoutineLoadTasksResponse {
  return isRecord(value) && Array.isArray(value.tasks) && value.tasks.every((task: unknown) =>
    isRecord(task) &&
    ['id', 'jobId', 'transactionId', 'transactionStatus', 'created', 'lastScheduled', 'executeStarted', 'timeout', 'nodeId', 'progress', 'message', 'warehouse']
      .every(field => typeof task[field] === 'string'));
}

export function isDictionariesResponse(value: unknown): value is DictionariesResponse {
  return isRecord(value) && Array.isArray(value.dictionaries) && value.dictionaries.every((dictionary: unknown) =>
    isRecord(dictionary) &&
    ['id', 'name', 'catalog', 'database', 'source', 'keys', 'values', 'status', 'lastRefreshStarted', 'lastRefreshFinished', 'nextRefresh', 'error', 'memoryUsage']
      .every(field => typeof dictionary[field] === 'string'));
}

export function isNodesResponse(value: unknown): value is NodesResponse {
  return isRecord(value) && Array.isArray(value.nodes) && value.nodes.every((node: unknown) =>
    isRecord(node) && typeof node.type === 'string' && ['frontend', 'backend', 'compute'].includes(node.type) &&
    ['id', 'host', 'role', 'status', 'warehouse', 'memoryLimit', 'cpuUsed', 'memoryUsed', 'lastStartTime', 'lastHeartbeat', 'version', 'error', 'dataCache']
      .every(field => typeof node[field] === 'string') &&
    (node.alive === null || typeof node.alive === 'boolean') && typeof node.decommissioned === 'boolean' &&
    ['cpuCores', 'runningQueries', 'tablets'].every(field => node[field] === null || typeof node[field] === 'number') &&
    isRecord(node.details) && Object.values(node.details).every(detail => typeof detail === 'string'));
}

export function isResourceGroupsResponse(value: unknown): value is ResourceGroupsResponse {
  return isRecord(value) && Array.isArray(value.groups) && value.groups.every((group: unknown) =>
    isRecord(group) && typeof group.id === 'string' && typeof group.name === 'string' &&
    isRecord(group.properties) && Object.values(group.properties).every(property => typeof property === 'string') &&
    Array.isArray(group.classifiers) && group.classifiers.every((classifier: unknown) => typeof classifier === 'string'));
}

export function isFrontendConfigResponse(value: unknown): value is FrontendConfigResponse {
  if (!isRecord(value)) return false;
  const { columns, settings } = value;
  return Array.isArray(columns) && columns.every(column => typeof column === 'string') &&
    columns.includes('Key') && columns.includes('Value') && Array.isArray(settings) &&
    settings.every((setting: unknown) => isRecord(setting) &&
      columns.every((column: unknown) => typeof column === 'string' && typeof setting[column] === 'string'));
}
