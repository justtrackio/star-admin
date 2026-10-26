import { createFileRoute, Link } from '@tanstack/react-router';
import { Alert, Button, Card, Col, Empty, Row, Space, Statistic, Table, Typography } from 'antd';
import type { TableColumnsType } from 'antd';
import type { ClusterNode } from '../api/schema';
import { NodeStatus } from '../components/NodeStatus';
import { useNodes } from '../hooks/useNodes';
import { formatTimestamp } from '../utils/formatTimestamp';

export const Route = createFileRoute('/nodes')({ component: NodesPage });

const columns = (type: ClusterNode['type']): TableColumnsType<ClusterNode> => [
  {
    title: 'Node',
    dataIndex: 'host',
    sorter: (a, b) => a.host.localeCompare(b.host),
    render: (host: string, node) => (
      <Link
        to="/nodes/$type/$id"
        params={{ type: node.type, id: node.id }}
        onClick={(event) => event.stopPropagation()}
      >
        <Space orientation="vertical" size={0}>
          <Typography.Text strong>{host || node.id}</Typography.Text>
          <Typography.Text type="secondary">ID {node.id}</Typography.Text>
        </Space>
      </Link>
    ),
  },
  { title: 'Status', render: (_, node) => <NodeStatus node={node} /> },
  { title: 'Role', hidden: type !== 'frontend', dataIndex: 'role', render: (value: string) => value || '—' },
  {
    title: 'CPU cores',
    dataIndex: 'cpuCores',
    align: 'right',
    render: (value: number | null) => value?.toLocaleString() ?? '—',
  },
  {
    title: 'CPU used',
    dataIndex: 'cpuUsed',
    align: 'right',
    render: (value: string, node) => (node.alive === false ? '—' : value || '—'),
  },
  {
    title: type === 'frontend' ? 'JVM heap used' : 'Memory used',
    dataIndex: 'memoryUsed',
    align: 'right',
    render: (value: string, node) => (node.alive === false ? '—' : value || '—'),
  },
  { title: 'Version', dataIndex: 'version', render: (value: string) => value || '—' },
];

function NodesPage() {
  const navigate = Route.useNavigate();
  const { data, error, isPending, dataUpdatedAt, refetch } = useNodes();
  const nodes = data?.nodes ?? [];

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <div>
        <Typography.Title level={2}>Nodes</Typography.Title>
        <Typography.Paragraph type="secondary">
          Frontend, backend, and compute nodes. Live updates every 5 seconds. Select a node to see its details.
        </Typography.Paragraph>
      </div>
      {error && (
        <Alert
          type="error"
          showIcon
          title="Could not update nodes"
          description={error.message}
          action={<Button onClick={refetch}>Retry</Button>}
        />
      )}
      {error && data && (
        <Alert
          type="warning"
          showIcon
          title="Showing the last successful snapshot. Node status and usage may be out of date."
        />
      )}
      <Typography.Paragraph type="secondary">
        Frontend memory shows JVM heap usage. Frontend CPU metrics are not reported by this cluster’s SQL metadata.
      </Typography.Paragraph>
      <Row gutter={[16, 16]}>
        {(['frontend', 'compute', 'backend'] as const).map((type) => (
          <Col xs={24} sm={8} key={type}>
            <Card>
              <Statistic
                title={`${type[0].toUpperCase()}${type.slice(1)} nodes`}
                value={data ? nodes.filter((node) => node.type === type).length : '—'}
                loading={isPending}
              />
            </Card>
          </Col>
        ))}
      </Row>
      {(['frontend', 'compute', 'backend'] as const).map((type) => (
        <Card
          key={type}
          title={`${type[0].toUpperCase()}${type.slice(1)} nodes`}
          extra={
            <Typography.Text type="secondary">
              Last updated: {dataUpdatedAt ? formatTimestamp(dataUpdatedAt) : '—'}
            </Typography.Text>
          }
        >
          <Table<ClusterNode>
            size="small"
            rowKey={(node) => `${node.type}:${node.id}`}
            columns={columns(type)}
            dataSource={nodes.filter((node) => node.type === type)}
            loading={isPending}
            scroll={{ x: 1100 }}
            pagination={{ defaultPageSize: 20, showSizeChanger: true }}
            onRow={(node) => ({
              style: { cursor: 'pointer' },
              onClick: () => void navigate({ to: '/nodes/$type/$id', params: { type: node.type, id: node.id } }),
            })}
            locale={{
              emptyText: (
                <Empty description={error ? 'Nodes unavailable' : `No ${type} nodes registered in this cluster`} />
              ),
            }}
          />
        </Card>
      ))}
    </Space>
  );
}
