import { Alert, Button, Card, Descriptions, Space, Spin, Typography } from 'antd';
import { isRoutineLoadSettings } from '../api/snapshots';
import { useSnapshotStream } from '../hooks/useSnapshotStream';

export function RoutineLoadSettings({ taskConsumeSecond }: { taskConsumeSecond: string }) {
  const settings = useSnapshotStream('/api/routine-load-settings', isRoutineLoadSettings);

  return <Card title="Routine load settings">
    <Space orientation="vertical" style={{ width: '100%' }}>
      {settings.isPending && <Spin aria-label="Loading routine settings" />}
      {settings.error && <Alert type="error" showIcon title="Could not update cluster settings" description={settings.error.message} action={<Button onClick={settings.refetch}>Retry</Button>} />}
      {settings.error && settings.data && <Alert type="warning" showIcon title="Showing the last successful cluster settings. Values may be out of date." />}
      <Descriptions size="small" column={1} items={[
        { key: 'maxBatchSize', label: 'max_routine_load_batch_size', children: settings.data?.maxBatchSize ? `${settings.data.maxBatchSize} bytes` : '—' },
        { key: 'defaultConsume', label: 'routine_load_task_consume_second', children: settings.data?.taskConsumeSecond ? `${settings.data.taskConsumeSecond} s (cluster default)` : '—' },
        { key: 'taskConsume', label: 'task_consume_second', children: taskConsumeSecond ? `${taskConsumeSecond} s (reported for this routine)` : '—' },
      ]} />
      <Typography.Text type="secondary">The batch size limit and consumption default are reported by the connected frontend. The routine value can be inherited or explicitly configured; this metadata does not distinguish the two. These are limits, not measured task durations.</Typography.Text>
    </Space>
  </Card>;
}
