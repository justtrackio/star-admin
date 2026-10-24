import { createFileRoute, Link } from '@tanstack/react-router';
import { useMutation } from '@tanstack/react-query';
import { Alert, Button, Card, Col, Descriptions, Popconfirm, Result, Row, Space, Spin, Statistic, Typography } from 'antd';
import { apiClient } from '../api/client';
import type { ApiError } from '../api/client';
import { RoutineLoadStatus } from '../components/RoutineLoadStatus';
import { useRoutineLoads } from '../hooks/useRoutineLoads';
import { formatTimestamp } from '../utils/formatTimestamp';
import { isKafkaOffsetError } from '../utils/isKafkaOffsetError';

export const Route = createFileRoute('/routine-loads_/$database/$id')({ component: RoutineLoadDetailPage });

function RoutineLoadDetailPage() {
  const { database, id } = Route.useParams();
  const jobs = useRoutineLoads();
  const job = jobs.data?.jobs.find(item => item.database === database && item.id === id);
  const control = useMutation<{ action: string }, ApiError, 'pause' | 'resume' | 'resume-earliest' | 'resume-latest'>({
    mutationFn: action => apiClient.post(`/api/routine-loads/${encodeURIComponent(database)}/${encodeURIComponent(id)}/${action}`),
    onSuccess: () => jobs.refetch(),
  });
  const reason = job?.reason || control.error?.message || '';
  const controlsDisabled = job?.state !== 'PAUSED' || !!jobs.error || control.isPending || jobs.isFetching;

  return <Space orientation="vertical" size="large" style={{ width: '100%' }}>
    <Link to="/routine-loads">← Kafka routines</Link>
    {jobs.isPending && <Spin tip="Loading Kafka routine"><div style={{ height: 160 }} /></Spin>}
    {jobs.error && <Alert type="error" showIcon title="Could not update Kafka routine" description={jobs.error.message} action={<Button loading={jobs.isFetching} onClick={jobs.refetch}>Retry</Button>} />}
    {jobs.error && jobs.data && <Alert type="warning" showIcon title="Showing the last successful ingestion snapshot. Job state may be out of date." />}
    {jobs.data && !job && <Result status="404" title="Kafka routine not found" subTitle={`This routine is no longer available in database ${database} on the connected cluster.`} />}
    {job && <>
      <div>
        <Typography.Title level={2} style={{ overflowWrap: 'anywhere' }}>{job.name}</Typography.Title>
        <Typography.Paragraph type="secondary">{job.database} · Kafka Routine Load · Live updates every 5 seconds.</Typography.Paragraph>
      </div>
      <Space wrap>
        <Button disabled={!['RUNNING', 'NEED_SCHEDULE', 'UNSTABLE'].includes(job.state) || !!jobs.error || control.isPending || jobs.isFetching} loading={control.isPending && control.variables === 'pause'} onClick={() => control.mutate('pause')}>Pause routine</Button>
        <Button type="primary" disabled={job.state !== 'PAUSED' || !!jobs.error || control.isPending || jobs.isFetching} loading={control.isPending && control.variables === 'resume'} onClick={() => control.mutate('resume')}>Resume routine</Button>
      </Space>
      {control.error && control.error.message !== reason && <Alert type="error" showIcon title="Could not change routine state" description={control.error.message} />}
      {reason && <Alert type={job.reason ? 'warning' : 'error'} showIcon title={job.reason ? 'State change reason' : 'Could not change routine state'} description={<Space orientation="vertical">
        <Typography.Text style={{ overflowWrap: 'anywhere' }}>{reason}</Typography.Text>
        {isKafkaOffsetError(reason) && <>
          <Typography.Text>Reset offsets for all reported Kafka partitions and resume this routine. Earliest can replay data; latest skips the retained backlog.</Typography.Text>
          <Space wrap>
            <Popconfirm title="Resume from earliest offsets?" description="Reset all reported partitions to their earliest retained offsets. This can replay rows already loaded. Deleted Kafka messages cannot be recovered." okText="Reset and resume" onConfirm={() => control.mutate('resume-earliest')} disabled={controlsDisabled}>
              <Button disabled={controlsDisabled} loading={control.isPending && control.variables === 'resume-earliest'}>Resume from earliest</Button>
            </Popconfirm>
            <Popconfirm title="Resume from latest offsets?" description="Reset all reported partitions to their latest offsets. This skips all messages currently waiting to be loaded." okText="Skip backlog and resume" onConfirm={() => control.mutate('resume-latest')} disabled={controlsDisabled}>
              <Button danger disabled={controlsDisabled} loading={control.isPending && control.variables === 'resume-latest'}>Resume from latest</Button>
            </Popconfirm>
          </Space>
        </>}
      </Space>} />}
      {control.isSuccess && <Alert type="success" showIcon title={`${control.variables === 'pause' ? 'Pause' : 'Resume'} command accepted. The live snapshot will show the updated state.`} />}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={8}><Card><Statistic title="Loaded rows" value={job.loadedRows || '—'} /></Card></Col>
        <Col xs={24} sm={8}><Card><Statistic title="Error rows" value={job.errorRows || '—'} /></Card></Col>
        <Col xs={24} sm={8}><Card><Statistic title="Reported rows/s" value={job.rowsPerSecond || '—'} /></Card></Col>
      </Row>
      <Card title="Overview" extra={<Space wrap>
        <Typography.Text type="secondary">Last updated: {jobs.dataUpdatedAt ? formatTimestamp(jobs.dataUpdatedAt) : '—'}</Typography.Text>
        <Button loading={jobs.isFetching} onClick={jobs.refetch}>Reload routine</Button>
      </Space>}>
        <Descriptions size="small" column={{ xs: 1, sm: 2, lg: 3 }} items={[
          { key: 'state', label: 'State', children: <RoutineLoadStatus state={job.state} /> },
          { key: 'id', label: 'Job ID', children: job.id },
          { key: 'database', label: 'Database', children: job.database },
          { key: 'table', label: 'Target table', children: job.table },
          { key: 'topic', label: 'Kafka topic', children: job.topic || '—' },
        ]} />
        <Typography.Paragraph type="secondary" style={{ marginTop: 16, marginBottom: 0 }}>Row totals belong to this job. Throughput is reported by StarRocks.</Typography.Paragraph>
      </Card>
      <Card title="Consumed offsets by Kafka partition">
        <pre style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{job.progress || 'No partition offsets reported.'}</pre>
        <Typography.Text type="secondary">Consumed offsets do not indicate consumer lag. Kafka end offsets are not available in this metadata.</Typography.Text>
      </Card>
    </>}
  </Space>;
}
