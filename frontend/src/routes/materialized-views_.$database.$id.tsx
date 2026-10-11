import { createFileRoute, Link } from '@tanstack/react-router';
import { Alert, Button, Card, Descriptions, Result, Space, Spin, Tooltip, Typography } from 'antd';
import { useMaterializedViews } from '../hooks/useMaterializedViews';
import { MaterializedViewRefreshStatus, MaterializedViewStatus } from '../components/MaterializedViewStatus';
import { formatRefreshSchedule } from '../utils/formatRefreshSchedule';
import { formatTimestamp } from '../utils/formatTimestamp';

export const Route = createFileRoute('/materialized-views_/$database/$id')({ component: MaterializedViewDetailPage });

function MaterializedViewDetailPage() {
  const { database, id } = Route.useParams();
  const { data, error, isPending, isFetching, refetch } = useMaterializedViews();
  const view = data?.views.find(item => item.database === database && item.id === id);

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <Link to="/materialized-views">← Materialized views</Link>
      {isPending && <Spin tip="Loading materialized view"><div style={{ height: 160 }} /></Spin>}
      {error && <Alert type="error" showIcon title="Could not load materialized view" description={error.message} action={<Button loading={isFetching} onClick={() => void refetch()}>Retry</Button>} />}
      {data && !view && <Result status="404" title="Materialized view not found" subTitle={`This view is no longer available in database ${database} on the connected cluster.`} extra={<Button loading={isFetching} onClick={() => void refetch()}>Reload details</Button>} />}
      {view && <>
        <div>
          <Typography.Title level={2} style={{ overflowWrap: 'anywhere' }}>{view.name}</Typography.Title>
          <Typography.Text type="secondary">{view.database} · Materialized view</Typography.Text>
        </div>
        {view.inactiveReason && <Alert type="warning" showIcon title="Inactive reason" description={view.inactiveReason} />}
        {view.lastRefreshError && <Alert type="error" showIcon title="Last refresh error" description={view.lastRefreshError} />}
        <Card title="Overview" extra={<Button loading={isFetching} onClick={() => void refetch()}>Reload details</Button>}>
          <Descriptions column={{ xs: 1, sm: 2 }} items={[
            { key: 'database', label: 'Database', children: view.database },
            { key: 'id', label: 'View ID', children: view.id },
            { key: 'refreshType', label: 'Refresh type', children: view.refreshType || '—' },
            { key: 'schedule', label: 'Refresh schedule', children: <Tooltip title={`Configured refresh policy; actual execution can be delayed or skipped if no data changed.${view.refreshPolicy ? ` ${view.refreshPolicy}` : ''}`}>{formatRefreshSchedule(view)}</Tooltip> },
            { key: 'active', label: 'Status', children: <MaterializedViewStatus active={view.active} reason={view.inactiveReason} /> },
            { key: 'rows', label: 'Rows (approx.)', children: view.rows?.toLocaleString() ?? '—' },
          ]} />
        </Card>
        <Card title="Last refresh">
          <Descriptions column={{ xs: 1, sm: 2 }} items={[
            { key: 'state', label: 'State', children: <MaterializedViewRefreshStatus state={view.lastRefreshState} error={view.lastRefreshError} /> },
            { key: 'lastSuccess', label: 'Last successful refresh at', children: <Tooltip title="Completion time of the most recent SUCCESS refresh in the available metadata or retained refresh history. Pending and skipped attempts do not replace it.">{formatTimestamp(view.lastSuccessfulRefreshFinished)}</Tooltip> },
          ]} />
        </Card>
        <Card title="SQL definition" extra={view.definition && <Typography.Text copyable={{ text: view.definition }}>Copy SQL</Typography.Text>}>
          <pre className="view-definition">{view.definition || 'No SQL definition available.'}</pre>
        </Card>
      </>}
    </Space>
  );
}
