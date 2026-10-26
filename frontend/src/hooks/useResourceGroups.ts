import { isResourceGroupsResponse } from '../api/snapshots';
import { useSnapshotStream } from './useSnapshotStream';

export function useResourceGroups() {
  return useSnapshotStream('/api/resource-groups', isResourceGroupsResponse);
}
