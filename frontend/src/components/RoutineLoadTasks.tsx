import { Alert, Button, Card, Space, Table, Tag, Tooltip, Typography } from 'antd';
import type { TableColumnsType } from 'antd';
import type { RoutineLoadTask } from '../api/schema';
import { isRoutineLoadTasksResponse } from '../api/snapshots';
import { useSnapshotStream } from '../hooks/useSnapshotStream';
import { formatTimestamp } from '../utils/formatTimestamp';

const columns: TableColumnsType<RoutineLoadTask> = [
  { title: 'Task ID', dataIndex: 'id', render: (value: string) => <Typography.Text copyable>{value}</Typography.Text> },
  { title: 'Transaction ID', dataIndex: 'transactionId', render: (value: string) => value === '-1' ? <Tooltip title="StarRocks reports TxnId = -1 for this queued task. This does not mean the routine has stopped ingesting.">Pending</Tooltip> : value || '—' },
  { title: 'Transaction status', dataIndex: 'transactionStatus', render: (value: string) => <Tag>{value || 'Unknown'}</Tag> },
  { title: 'Execution node', dataIndex: 'nodeId', render: (value: string) => value === '-1' ? <Tooltip title="StarRocks reports BeId = -1. A node has not yet been assigned to this current task.">Pending assignment</Tooltip> : value || '—' },
  { title: 'Created', dataIndex: 'created', render: formatTimestamp },
  { title: 'Last scheduled', dataIndex: 'lastScheduled', render: formatTimestamp },
  { title: 'Execution started', dataIndex: 'executeStarted', render: formatTimestamp },
  { title: 'Timeout (s)', dataIndex: 'timeout', render: (value: string) => value || '—' },
  { title: 'Warehouse', dataIndex: 'warehouse', render: (value: string) => value || '—' },
  { title: 'Message', dataIndex: 'message', width: 300, render: (value: string) => value || '—' },
];

const expandable = {
  expandedRowRender: (task: RoutineLoadTask) => <Space orientation="vertical" style={{ width: '100%' }}>
    <Typography.Text>Partition progress and latest offsets (as reported by StarRocks)</Typography.Text>
    <pre style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{task.progress || 'No partition offsets reported.'}</pre>
    <Typography.Text>Message: {task.message || '—'}</Typography.Text>
  </Space>,
};

export function RoutineLoadTasks({ database, id }: { database: string; id: string }) {
  const tasks = useSnapshotStream(
    `/api/routine-loads/${encodeURIComponent(database)}/${encodeURIComponent(id)}/tasks`,
    isRoutineLoadTasksResponse,
  );

  return <Card title={`Current tasks${tasks.data ? ` (${tasks.data.tasks.length})` : ''}`} extra={<Typography.Text type="secondary">Last updated: {tasks.dataUpdatedAt ? formatTimestamp(tasks.dataUpdatedAt) : '—'}</Typography.Text>}>
    <Space orientation="vertical" style={{ width: '100%' }}>
      <Typography.Text type="secondary">Current task snapshots update every second. Tasks can execute briefly and be replaced by new queued tasks between snapshots. Pending transactions and node assignments do not mean ingestion has stopped. Completed tasks are not retained here. Expand a task for partition progress and latest offsets.</Typography.Text>
      {tasks.error && <Alert type="error" showIcon title="Could not update routine tasks" description={tasks.error.message} action={<Button onClick={tasks.refetch}>Retry</Button>} />}
      {tasks.error && tasks.data && <Alert type="warning" showIcon title="Showing the last successful task snapshot. Task details may be out of date." />}
      <Table<RoutineLoadTask> size="small" rowKey="id" columns={columns} dataSource={tasks.data?.tasks ?? []} loading={tasks.isPending} scroll={{ x: 2100 }} pagination={{ defaultPageSize: 10, showSizeChanger: true, hideOnSinglePage: true }} locale={{ emptyText: tasks.error ? 'Task metadata unavailable' : 'No current tasks. Paused or stopped routines may have no tasks.' }} expandable={expandable} />
    </Space>
  </Card>;
}
