import { createFileRoute, Link } from '@tanstack/react-router';
import { Alert, Card, Space, Spin, Tag, Typography } from 'antd';
import { useHealth } from '../hooks/useHealth';

export const Route = createFileRoute('/')({ component: Dashboard });

function Dashboard() {
  const { data, error, isPending } = useHealth();

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <Typography.Title level={2}>Cluster administration</Typography.Title>
      <Card title="Application status">
        {isPending && <Spin tip="Checking backend"><div style={{ height: 48 }} /></Spin>}
        {error && <Alert type="error" showIcon title="Backend unavailable" description={error.message} />}
        {data && <Space><Tag color="green">Backend online</Tag><Typography.Text>{data.service}</Typography.Text></Space>}
      </Card>
      <Card title="Nodes">
        <Typography.Paragraph>Browse frontend, backend, and compute nodes and inspect their live details.</Typography.Paragraph>
        <Link to="/nodes">View nodes →</Link>
      </Card>
      <Card title="Resource groups">
        <Typography.Paragraph>Browse resource groups, configured limits, and classifier rules.</Typography.Paragraph>
        <Link to="/resource-groups">View resource groups →</Link>
      </Card>
      <Card title="Tables">
        <Typography.Paragraph>Browse regular tables, approximate row counts, and data size.</Typography.Paragraph>
        <Link to="/tables">View tables →</Link>
      </Card>
      <Card title="Kafka routines">
        <Typography.Paragraph>Inspect Kafka ingestion jobs, throughput, errors, and consumed offsets.</Typography.Paragraph>
        <Link to="/routine-loads">View Kafka routines →</Link>
      </Card>
      <Card title="Dictionaries">
        <Typography.Paragraph>Browse available dictionaries, refresh status, errors, and memory details.</Typography.Paragraph>
        <Link to="/dictionaries">View dictionaries →</Link>
      </Card>
      <Card title="Materialized views">
        <Typography.Paragraph>Browse materialized views across the connected cluster's databases, inspect their status, and read their SQL definitions.</Typography.Paragraph>
        <Link to="/materialized-views">View materialized views →</Link>
      </Card>
    </Space>
  );
}
