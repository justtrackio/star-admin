import { createFileRoute, Link } from '@tanstack/react-router';
import { useMutation } from '@tanstack/react-query';
import { Alert, Button, Card, Descriptions, Popconfirm, Result, Space, Spin, Tabs, Tooltip, Typography } from 'antd';
import { apiClient } from '../api/client';
import type { ApiError } from '../api/client';
import { MaterializedViewTasks } from '../components/MaterializedViewTasks';
import { useMaterializedView } from '../hooks/useMaterializedViews';
import { MaterializedViewStatus } from '../components/MaterializedViewStatus';
import { formatRefreshSchedule } from '../utils/formatRefreshSchedule';

export const Route = createFileRoute('/materialized-views_/$database/$id')({ component: MaterializedViewDetailPage });

function MaterializedViewDetailPage() {
  const { database, id } = Route.useParams();
  const { data, error, isPending, isFetching, refetch } = useMaterializedView(database, id);
  const view = data?.views.find((item) => item.database === database && item.id === id);
  const control = useMutation<{ action: string }, ApiError, 'pause' | 'resume'>({
    mutationFn: (action) =>
      apiClient.post(`/api/materialized-views/${encodeURIComponent(database)}/${encodeURIComponent(id)}/${action}`),
    onSuccess: refetch,
  });
  const controlsDisabled = !!error || control.isPending || isFetching;

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <Link to="/materialized-views">← Materialized views</Link>
      {isPending && (
        <Spin tip="Loading materialized view">
          <div style={{ height: 160 }} />
        </Spin>
      )}
      {error && (
        <Alert
          type="error"
          showIcon
          title="Could not load materialized view"
          description={error.message}
          action={
            <Button loading={isFetching} onClick={refetch}>
              Retry
            </Button>
          }
        />
      )}
      {data && !view && (
        <Result
          status="404"
          title="Materialized view not found"
          subTitle={`This view is no longer available in database ${database} on the connected cluster.`}
        />
      )}
      {view && (
        <>
          <div>
            <Typography.Title level={2} style={{ overflowWrap: 'anywhere' }}>
              {view.name}
            </Typography.Title>
            <Typography.Text type="secondary">{view.database} · Materialized view</Typography.Text>
          </div>
          {view.refreshType.toUpperCase() === 'ASYNC' && (
            <Space orientation="vertical">
              <Space wrap>
                <Popconfirm
                  title="Pause refresh and deactivate this view?"
                  description="This disables refresh and query rewrite. Dependent materialized views may also become inactive and need separate reactivation. The refresh schedule is preserved."
                  okText="Pause refresh"
                  onConfirm={() => control.mutate('pause')}
                  disabled={view.active.toLowerCase() !== 'true' || controlsDisabled}
                >
                  <Button
                    disabled={view.active.toLowerCase() !== 'true' || controlsDisabled}
                    loading={control.isPending && control.variables === 'pause'}
                  >
                    Pause refresh
                  </Button>
                </Popconfirm>
                <Button
                  type="primary"
                  disabled={view.active.toLowerCase() !== 'false' || controlsDisabled}
                  loading={control.isPending && control.variables === 'resume'}
                  onClick={() => control.mutate('resume')}
                >
                  Resume refresh
                </Button>
              </Space>
            </Space>
          )}
          {control.error && (
            <Alert type="error" showIcon title="Could not change refresh state" description={control.error.message} />
          )}
          {control.isSuccess && (
            <Alert
              type="success"
              showIcon
              title={`${control.variables === 'pause' ? 'Pause' : 'Resume'} command accepted. The live snapshot will show the updated state.`}
            />
          )}
          {view.inactiveReason && (
            <Alert type="warning" showIcon title="Inactive reason" description={view.inactiveReason} />
          )}
          {view.lastRefreshError && (
            <Alert type="error" showIcon title="Last refresh error" description={view.lastRefreshError} />
          )}
          <Card size="small" title="Overview">
            <Descriptions
              size="small"
              column={{ xs: 1, sm: 2, lg: 3 }}
              items={[
                { key: 'database', label: 'Database', children: view.database },
                { key: 'id', label: 'View ID', children: view.id },
                { key: 'refreshType', label: 'Refresh type', children: view.refreshType || '—' },
                {
                  key: 'schedule',
                  label: 'Refresh schedule',
                  children: (
                    <Tooltip
                      title={`Configured refresh policy; actual execution can be delayed or skipped if no data changed.${view.refreshPolicy ? ` ${view.refreshPolicy}` : ''}`}
                    >
                      {formatRefreshSchedule(view)}
                    </Tooltip>
                  ),
                },
                {
                  key: 'active',
                  label: 'Status',
                  children: <MaterializedViewStatus active={view.active} reason={view.inactiveReason} />,
                },
                { key: 'rows', label: 'Rows (approx.)', children: view.rows?.toLocaleString() ?? '—' },
              ]}
            />
          </Card>
          <Tabs
            defaultActiveKey="tasks"
            destroyOnHidden
            items={[
              { key: 'tasks', label: 'Refresh tasks', children: <MaterializedViewTasks database={database} id={id} /> },
              {
                key: 'sql',
                label: 'SQL definition',
                children: (
                  <Card
                    extra={
                      view.definition && (
                        <Typography.Text copyable={{ text: view.definition }}>Copy SQL</Typography.Text>
                      )
                    }
                  >
                    <pre className="view-definition">{view.definition || 'No SQL definition available.'}</pre>
                  </Card>
                ),
              },
            ]}
          />
        </>
      )}
    </Space>
  );
}
