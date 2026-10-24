import { Alert, Button, Space, Table, Typography } from 'antd';
import type { TableProps } from 'antd';
import { useSnapshotStream } from '../hooks/useSnapshotStream';
import { formatPartitionName } from '../utils/formatPartitionName';
import { formatTimestamp } from '../utils/formatTimestamp';
import { MaterializedViewRefreshStatus } from './MaterializedViewStatus';

interface RefreshTask {
  partitions: string[] | null;
  queueTime: number | null;
  processingTime: number | null;
  queryId: string;
  state: string;
  created: string;
  finished: string;
  progress: string;
  error: string;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function isRefreshTasksResponse(value: unknown): value is { tasks: RefreshTask[] } {
  if (!value || typeof value !== 'object' || !('tasks' in value) || !Array.isArray(value.tasks)) return false;
  return value.tasks.every((task: unknown) => isRecord(task) &&
    ['queryId', 'state', 'created', 'finished', 'progress', 'error'].every(field =>
      typeof task[field] === 'string') &&
    (task.partitions === null || (Array.isArray(task.partitions) && task.partitions.every((part: unknown) => typeof part === 'string'))) &&
    (task.queueTime === null || typeof task.queueTime === 'number') &&
    (task.processingTime === null || typeof task.processingTime === 'number'));
}

function formatDuration(seconds: number | null): string {
  if (seconds == null) return '—';
  if (seconds < 60) return `${seconds.toFixed(1)} s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${(seconds % 60).toFixed(0)}s`;
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
}

const taskColumns: TableProps<RefreshTask>['columns'] = [
  { title: 'Created', dataIndex: 'created', render: (value: string) => formatTimestamp(value) },
  { title: 'Finished', dataIndex: 'finished', render: (value: string) => formatTimestamp(value) },
  { title: 'State', key: 'state', render: (_, task) => <MaterializedViewRefreshStatus state={task.state} error={task.error} /> },
  { title: 'Progress', dataIndex: 'progress', render: (value: string) => value || '—' },
  { title: 'Partitions', key: 'partitionCount', render: (_, task) => task.partitions?.length ?? '—' },
  { title: 'Partition names', dataIndex: 'partitions', render: (partitions: string[] | null) => {
    const names = partitions?.map(formatPartitionName).join(', ') || '—';
    return <Typography.Text style={{ maxWidth: 220 }} ellipsis={{ tooltip: names }}>{names}</Typography.Text>;
  } },
  { title: 'Queue time', dataIndex: 'queueTime', render: formatDuration },
  { title: 'Processing time', dataIndex: 'processingTime', render: formatDuration },
  { title: 'Query ID', dataIndex: 'queryId', render: (value: string) => <Typography.Text copyable>{value}</Typography.Text> },
];

const taskExpandable: TableProps<RefreshTask>['expandable'] = {
  rowExpandable: task => Boolean(task.error),
  expandedRowRender: task => <Alert type="error" showIcon title="Refresh error" description={task.error} />,
};

export function MaterializedViewTasks({ database, id }: { database: string; id: string }) {
  const { data, error, isFetching, isPending, refetch } = useSnapshotStream(
    `/api/materialized-views/${encodeURIComponent(database)}/${encodeURIComponent(id)}/tasks`,
    isRefreshTasksResponse,
  );

  return (
    <Space orientation="vertical" style={{ width: '100%' }}>
      {error && <Alert type="error" showIcon title="Could not load refresh tasks" description={error.message} action={<Button loading={isFetching} onClick={refetch}>Retry</Button>} />}
      <Table<RefreshTask>
        size="small"
        rowKey="queryId"
        loading={isPending}
        dataSource={data?.tasks ?? []}
        scroll={{ x: 1600 }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, hideOnSinglePage: true }}
        locale={{ emptyText: 'No retained refresh tasks for this materialized view.' }}
        columns={taskColumns}
        expandable={taskExpandable}
      />
    </Space>
  );
}
