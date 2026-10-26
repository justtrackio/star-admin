import { createFileRoute, Link } from '@tanstack/react-router';
import { Badge, Button, Card, Col, Row, Space, Typography } from 'antd';
import { useHealth } from '../hooks/useHealth';
import { useNodes } from '../hooks/useNodes';
import { useDictionaries } from '../hooks/useDictionaries';
import { useMaterializedViews } from '../hooks/useMaterializedViews';
import { useRoutineLoads } from '../hooks/useRoutineLoads';
import { formatTimestamp } from '../utils/formatTimestamp';
import {
  isDictionaryHealthy,
  isNodeHealthy,
  isRoutineHealthy,
  isViewHealthy,
  resourceStatus,
} from '../utils/clusterHealth';

export const Route = createFileRoute('/')({ component: Dashboard });

const indicators = {
  healthy: { status: 'success', text: 'Healthy' },
  unhealthy: { status: 'error', text: 'Needs attention' },
  unavailable: { status: 'error', text: 'Status unavailable' },
  loading: { status: 'processing', text: 'Checking status' },
  empty: { status: 'default', text: 'Not configured' },
} as const;

type ResourcePath = '/nodes' | '/routine-loads' | '/dictionaries' | '/materialized-views';

function StatusCard({
  title,
  to,
  total,
  healthy,
  error,
  updatedAt,
  retry,
  rule,
  required = false,
}: {
  title: string;
  to: ResourcePath;
  total: number | undefined;
  healthy: number;
  error: Error | null;
  updatedAt: number;
  retry: () => void;
  rule: string;
  required?: boolean;
}) {
  const state = resourceStatus(total, healthy, error, required);
  return (
    <Col xs={24} md={12} xxl={6}>
      <Card title={title} extra={<Badge {...indicators[state]} />} style={{ height: '100%' }}>
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Typography.Text strong style={{ fontSize: 24 }}>
            {total === undefined ? '—' : `${healthy} / ${total} healthy`}
          </Typography.Text>
          <Typography.Text type="secondary">{rule}</Typography.Text>
          {required && total === 0 && <Typography.Text type="danger">No nodes reported.</Typography.Text>}
          {error && (
            <div role="alert">
              <Typography.Text type="danger">{error.message} Previous data may be out of date.</Typography.Text>{' '}
              <Button size="small" onClick={retry}>
                Retry
              </Button>
            </div>
          )}
          <Typography.Text type="secondary">
            Last updated: {updatedAt ? formatTimestamp(updatedAt) : '—'}
          </Typography.Text>
          <Link to={to}>View {title.toLowerCase()} →</Link>
        </Space>
      </Card>
    </Col>
  );
}

function Dashboard() {
  const application = useHealth();
  const nodes = useNodes();
  const routines = useRoutineLoads();
  const dictionaries = useDictionaries();
  const views = useMaterializedViews();

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <div>
        <Typography.Title level={2}>Cluster status</Typography.Title>
        <Typography.Paragraph type="secondary">
          Live resource checks from the connected cluster. Updates every 5 seconds.
        </Typography.Paragraph>
        <Badge
          status={application.error ? 'error' : application.data ? 'success' : 'processing'}
          text={
            application.error
              ? 'Dashboard backend unavailable'
              : application.data
                ? 'Dashboard backend online'
                : 'Checking dashboard backend'
          }
        />
      </div>
      <Row gutter={[24, 24]}>
        <StatusCard
          title="Nodes"
          to="/nodes"
          total={nodes.data?.nodes.length}
          healthy={nodes.data?.nodes.filter(isNodeHealthy).length ?? 0}
          error={nodes.error}
          updatedAt={nodes.dataUpdatedAt}
          retry={nodes.refetch}
          required
          rule="All reported nodes must be online, with no decommissioning or reported errors."
        />
        <StatusCard
          title="Kafka routines"
          to="/routine-loads"
          total={routines.data?.jobs.length}
          healthy={routines.data?.jobs.filter(isRoutineHealthy).length ?? 0}
          error={routines.error}
          updatedAt={routines.dataUpdatedAt}
          retry={routines.refetch}
          rule="All routines must be running or waiting to be scheduled."
        />
        <StatusCard
          title="Dictionaries"
          to="/dictionaries"
          total={dictionaries.data?.dictionaries.length}
          healthy={dictionaries.data?.dictionaries.filter(isDictionaryHealthy).length ?? 0}
          error={dictionaries.error}
          updatedAt={dictionaries.dataUpdatedAt}
          retry={dictionaries.refetch}
          rule="All dictionaries must be ready or refreshing, with no reported refresh errors."
        />
        <StatusCard
          title="Materialized views"
          to="/materialized-views"
          total={views.data?.views.length}
          healthy={views.data?.views.filter(isViewHealthy).length ?? 0}
          error={views.error}
          updatedAt={views.dataUpdatedAt}
          retry={views.refetch}
          rule="All views must be active, with a known refresh state and no reported errors. This does not measure data freshness."
        />
      </Row>
    </Space>
  );
}
