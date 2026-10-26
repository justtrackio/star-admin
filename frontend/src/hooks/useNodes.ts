import { isNodesResponse } from '../api/snapshots';
import { useSnapshotStream } from './useSnapshotStream';

export function useNodes() {
  return useSnapshotStream('/api/nodes', isNodesResponse);
}
