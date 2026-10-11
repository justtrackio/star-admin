import { createFileRoute } from '@tanstack/react-router';
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
      <Alert type="info" showIcon title="Project foundation ready" description="StarRocks administration features have not been implemented yet." />
    </Space>
  );
}
