import { Tag, Tooltip } from 'antd';
import type { ClusterNode } from '../api/schema';

export function NodeStatus({ node }: { node: ClusterNode }) {
  const color =
    node.alive === false ? 'red' : node.decommissioned ? 'orange' : node.alive === true ? 'green' : 'default';
  const label =
    node.alive === false
      ? 'Offline'
      : node.decommissioned
        ? 'Decommissioned'
        : node.alive === true
          ? 'Online'
          : 'Unknown';
  return (
    <Tooltip title={node.error || node.status || undefined}>
      <Tag color={color}>{label}</Tag>
    </Tooltip>
  );
}
