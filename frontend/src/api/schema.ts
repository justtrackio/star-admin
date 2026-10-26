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

export interface ClusterNode {
  type: 'frontend' | 'backend' | 'compute';
  role: string;
  details: Record<string, string>;
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

export interface NodesResponse {
  nodes: ClusterNode[];
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
  taskConsumeSecond: string;
}

export interface RoutineLoadSettings {
  maxBatchSize: string;
  taskConsumeSecond: string;
}

export interface RoutineLoadsResponse {
  jobs: RoutineLoad[];
}

export interface RoutineLoadTask {
  id: string;
  jobId: string;
  transactionId: string;
  transactionStatus: string;
  created: string;
  lastScheduled: string;
  executeStarted: string;
  timeout: string;
  nodeId: string;
  progress: string;
  message: string;
  warehouse: string;
}

export interface RoutineLoadTasksResponse {
  tasks: RoutineLoadTask[];
}

export interface ResourceGroup {
  id: string;
  name: string;
  properties: Record<string, string>;
  classifiers: string[];
}

export interface ResourceGroupsResponse {
  groups: ResourceGroup[];
}

export interface FrontendConfigResponse {
  columns: string[];
  settings: Record<string, string>[];
}
