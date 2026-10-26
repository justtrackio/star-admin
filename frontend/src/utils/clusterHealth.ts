import type { ClusterNode, Dictionary, MaterializedView, RoutineLoad } from '../api/schema';

export function isNodeHealthy(node: Pick<ClusterNode, 'alive' | 'decommissioned' | 'error'>) {
  return node.alive === true && !node.decommissioned && !node.error;
}

export function isRoutineHealthy(job: Pick<RoutineLoad, 'state'>) {
  return job.state === 'RUNNING' || job.state === 'NEED_SCHEDULE';
}

export function isDictionaryHealthy(dictionary: Pick<Dictionary, 'status' | 'error'>) {
  return ['FINISHED', 'REFRESHING', 'COMMITTING'].includes(dictionary.status) && !dictionary.error;
}

export function isViewHealthy(
  view: Pick<MaterializedView, 'active' | 'inactiveReason' | 'lastRefreshState' | 'lastRefreshError'>,
) {
  return (
    view.active.toLowerCase() === 'true' &&
    !view.inactiveReason &&
    !view.lastRefreshError &&
    ['SUCCESS', 'SKIPPED', 'PENDING', 'RUNNING', 'MERGED'].includes(view.lastRefreshState)
  );
}

export function resourceStatus(total: number | undefined, healthy: number, error: Error | null, required = false) {
  if (error) return 'unavailable';
  if (total === undefined) return 'loading';
  if (total === 0) return required ? 'unhealthy' : 'empty';
  return healthy === total ? 'healthy' : 'unhealthy';
}
