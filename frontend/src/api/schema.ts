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
