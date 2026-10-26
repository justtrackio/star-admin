import { createFileRoute, Link } from '@tanstack/react-router';
import { Alert, Button, Card, Descriptions, Result, Space, Spin, Typography } from 'antd';
import { FrontendConfig } from '../components/FrontendConfig';
import { NodeStatus } from '../components/NodeStatus';
import { useNodes } from '../hooks/useNodes';
import { formatTimestamp } from '../utils/formatTimestamp';

export const Route = createFileRoute('/nodes_/$type/$id')({ component: NodeDetailPage });

function NodeDetailPage() {
  const { type, id } = Route.useParams();
  const { data, error, isPending, isFetching, dataUpdatedAt, refetch } = useNodes();
  const node = data?.nodes.find((item) => item.type === type && item.id === id);

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <Link to="/nodes">← Nodes</Link>
      {isPending && (
        <Spin tip="Loading node">
          <div style={{ height: 160 }} />
        </Spin>
      )}
      {error && (
        <Alert
          type="error"
          showIcon
          title="Could not update node"
          description={error.message}
          action={
            <Button loading={isFetching} onClick={refetch}>
              Retry
            </Button>
          }
        />
      )}
      {error && data && (
        <Alert
          type="warning"
          showIcon
          title="Showing the last successful snapshot. Node status and usage may be out of date."
        />
      )}
      {data && !node && (
        <Result
          status="404"
          title="Node not found"
          subTitle="This node is no longer available on the connected cluster."
        />
      )}
      {node && (
        <>
          <div>
            <Typography.Title level={2} style={{ overflowWrap: 'anywhere' }}>
              {node.host || node.id}
            </Typography.Title>
            <Typography.Paragraph type="secondary">
              {node.type} node · ID {node.id} · Live updates every 5 seconds.
            </Typography.Paragraph>
          </div>
          {node.error && <Alert type="error" showIcon title="Node error" description={node.error} />}
          <Card
            title="Node details"
            extra={
              <Typography.Text type="secondary">
                Last updated: {dataUpdatedAt ? formatTimestamp(dataUpdatedAt) : '—'}
              </Typography.Text>
            }
          >
            <Descriptions
              column={{ xs: 1, md: 2 }}
              items={[
                { key: 'status', label: 'Status', children: <NodeStatus node={node} /> },
                { key: 'host', label: 'Host', children: node.host || '—' },
                { key: 'heartbeat', label: 'Last heartbeat', children: formatTimestamp(node.lastHeartbeat) },
                { key: 'start', label: 'Last started', children: formatTimestamp(node.lastStartTime) },
                {
                  key: 'memory',
                  label: node.type === 'frontend' ? 'JVM heap used' : 'Memory used',
                  children: node.alive === false ? '—' : node.memoryUsed || '—',
                },
                {
                  key: 'memoryLimit',
                  label: node.type === 'frontend' ? 'JVM heap maximum' : 'Memory limit',
                  children: node.memoryLimit || '—',
                },
                { key: 'version', label: 'Version', children: node.version || '—' },
              ]}
            />
          </Card>
          {node.type === 'frontend' && <FrontendConfig />}
          <Card title="Reported metadata">
            <Descriptions
              bordered
              size="small"
              column={{ xs: 1, lg: 2 }}
              items={Object.entries(node.details).map(([key, value]) => ({
                key,
                label: key,
                children: <span style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{value || '—'}</span>,
              }))}
            />
          </Card>
        </>
      )}
    </Space>
  );
}
