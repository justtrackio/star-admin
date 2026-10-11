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
