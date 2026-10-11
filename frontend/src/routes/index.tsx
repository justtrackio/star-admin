import { createFileRoute, Link } from '@tanstack/react-router';
import { Alert, Button, Card, Space, Spin, Tag, Typography } from 'antd';
import { useHealth } from '../hooks/useHealth';

export const Route = createFileRoute('/')({ component: Dashboard });

function Dashboard() {
  const { data, error, isPending, isFetching, refetch } = useHealth();

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <Typography.Title level={2}>Cluster administration</Typography.Title>
      <Card title="Application status" extra={<Button loading={isFetching} onClick={() => void refetch()}>Refresh</Button>}>
        {isPending && <Spin tip="Checking backend"><div style={{ height: 48 }} /></Spin>}
        {error && <Alert type="error" showIcon title="Backend unavailable" description={error.message} />}
        {data && <Space><Tag color="green">Backend online</Tag><Typography.Text>{data.service}</Typography.Text></Space>}
      </Card>
      <Card title="Cluster overview">
        <Typography.Paragraph>See compute node availability, capacity, resource usage, and heartbeat details.</Typography.Paragraph>
        <Link to="/overview">View cluster overview →</Link>
      </Card>
      <Card title="Materialized views">
        <Typography.Paragraph>Browse materialized views across the connected cluster's databases, inspect their status, and read their SQL definitions.</Typography.Paragraph>
        <Link to="/materialized-views">View materialized views →</Link>
      </Card>
    </Space>
  );
}
