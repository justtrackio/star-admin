export interface HealthResponse {
  status: 'ok';
  service: string;
}

export interface MaterializedView {
  id: string;
  database: string;
  name: string;
  refreshType: string;
  refreshTrigger: string;
  refreshPolicy: string;
  active: string;
  inactiveReason: string;
  lastRefreshState: string;
  lastRefreshFinished: string;
  lastSuccessfulRefreshFinished: string;
  lastRefreshDuration: number | null;
  lastRefreshError: string;
  rows: number | null;
  definition: string;
}

export interface MaterializedViewsResponse {
  views: MaterializedView[];
}

export interface ComputeNode {
  id: string;
  host: string;
  alive: boolean | null;
  decommissioned: boolean;
  status: string;
  warehouse: string;
  cpuCores: number | null;
  memoryLimit: string;
  cpuUsed: string;
  memoryUsed: string;
  runningQueries: number | null;
  tablets: number | null;
  lastStartTime: string;
  lastHeartbeat: string;
  version: string;
  error: string;
  dataCache: string;
}

export interface OverviewResponse {
  nodes: ComputeNode[];
}

export interface Dictionary {
  id: string;
  name: string;
  catalog: string;
  database: string;
  source: string;
  keys: string;
  values: string;
  status: string;
  lastRefreshStarted: string;
  lastRefreshFinished: string;
  nextRefresh: string;
  error: string;
  memoryUsage: string;
}

export interface DictionariesResponse {
  dictionaries: Dictionary[];
}

export interface ClusterTable {
  database: string;
  name: string;
  engine: string;
  rows: number | null;
  dataBytes: number | null;
}

export interface TablesResponse {
  tables: ClusterTable[];
}

export interface RoutineLoad {
  id: string;
  database: string;
  table: string;
  name: string;
  state: string;
  topic: string;
  loadedRows: string;
  errorRows: string;
  rowsPerSecond: string;
  progress: string;
  reason: string;
}

export interface RoutineLoadsResponse {
  jobs: RoutineLoad[];
}
