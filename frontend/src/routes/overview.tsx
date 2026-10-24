import { createFileRoute } from '@tanstack/react-router';
import { Alert, Button, Card, Col, Descriptions, Empty, Row, Space, Statistic, Table, Tag, Tooltip, Typography } from 'antd';
import type { TableColumnsType } from 'antd';
import type { ComputeNode } from '../api/schema';
import { useOverview } from '../hooks/useOverview';
import { formatTimestamp } from '../utils/formatTimestamp';

export const Route = createFileRoute('/overview')({ component: OverviewPage });

function NodeStatus({ node }: { node: ComputeNode }) {
  const color = node.alive === false ? 'red' : node.decommissioned ? 'orange' : node.alive === true ? 'green' : 'default';
  const label = node.alive === false ? 'Offline' : node.decommissioned ? 'Decommissioned' : node.alive === true ? 'Online' : 'Unknown';
  return <Tooltip title={node.error || node.status || undefined}><Tag color={color}>{label}</Tag></Tooltip>;
}

const columns: TableColumnsType<ComputeNode> = [
  { title: 'Node', dataIndex: 'host', width: 180, fixed: 'left', sorter: (a, b) => a.host.localeCompare(b.host), render: (host: string, node) => <Tooltip title={host}><Space orientation="vertical" size={0}><Typography.Text strong>{host.split('.')[0] || node.id}</Typography.Text><Typography.Text type="secondary">ID {node.id}</Typography.Text></Space></Tooltip> },
  { title: 'Status', render: (_, node) => <NodeStatus node={node} /> },
  { title: 'CPU cores', dataIndex: 'cpuCores', align: 'right', render: (value: number | null) => value?.toLocaleString() ?? '—' },
  { title: 'CPU used', dataIndex: 'cpuUsed', align: 'right', render: (value: string, node) => node.alive === false ? '—' : value || '—' },
  { title: 'Memory limit', dataIndex: 'memoryLimit', align: 'right', render: (value: string) => value || '—' },
  { title: 'Memory used', dataIndex: 'memoryUsed', align: 'right', render: (value: string, node) => node.alive === false ? '—' : value || '—' },
  { title: 'Running queries', dataIndex: 'runningQueries', align: 'right', render: (value: number | null, node) => node.alive === false ? '—' : value?.toLocaleString() ?? '—' },
  { title: 'Data cache', dataIndex: 'dataCache', width: 250, render: (value: string, node) => node.alive === false || !value ? '—' : <Space orientation="vertical" size={0}>{value.split(', ').map(part => <Typography.Text key={part}>{part.replace('DiskUsage:', 'Disk used:').replace('MemUsage:', 'Memory used:')}</Typography.Text>)}</Space> },
  { title: 'Tablets', dataIndex: 'tablets', align: 'right', render: (value: number | null) => value?.toLocaleString() ?? '—' },
];

function NodeDetails({ node }: { node: ComputeNode }) {
  return <Space orientation="vertical" style={{ width: '100%' }}>
    {node.error && <Alert type="error" showIcon title="Node error" description={node.error} />}
    <Descriptions column={{ xs: 1, md: 2 }} items={[
      { key: 'host', label: 'Host', children: node.host || '—' },
      { key: 'status', label: 'Reported status', children: node.status || '—' },
      { key: 'heartbeat', label: 'Last heartbeat', children: formatTimestamp(node.lastHeartbeat) },
      { key: 'start', label: 'Last started', children: formatTimestamp(node.lastStartTime) },
      { key: 'version', label: 'Version', children: node.version || '—' },
    ]} />
  </Space>;
}

const expandable = { expandedRowRender: (node: ComputeNode) => <NodeDetails node={node} /> };

function OverviewPage() {
  const { data, error, isPending, isFetching, dataUpdatedAt, refetch } = useOverview();
  const nodes = data?.nodes ?? [];
  const online = nodes.filter(node => node.alive === true);
  const cores = online.every(node => node.cpuCores !== null) ? online.reduce((sum, node) => sum + (node.cpuCores ?? 0), 0) : '—';
  const queries = online.every(node => node.runningQueries !== null) ? online.reduce((sum, node) => sum + (node.runningQueries ?? 0), 0) : '—';

  return <Space orientation="vertical" size="large" style={{ width: '100%' }}>
    <div>
      <Typography.Title level={2}>Cluster overview</Typography.Title>
      <Typography.Paragraph type="secondary">Compute node availability and resource usage. Updates every 15 seconds. Expand a node for heartbeat and version details.</Typography.Paragraph>
    </div>
    {error && <Alert type="error" showIcon title="Could not update cluster overview" description={error.message} action={<Button onClick={() => void refetch()}>Retry</Button>} />}
    {error && data && <Alert type="warning" showIcon title="Showing the last successful snapshot. Node status and usage may be out of date." />}
    <Row gutter={[16, 16]}>
      {[
        { title: 'Compute nodes', value: data ? nodes.length : '—' },
        { title: 'Online nodes', value: data ? `${online.length} / ${nodes.length}` : '—' },
        { title: 'Online CPU cores', value: data ? cores : '—' },
        { title: 'Queries across online nodes', value: data ? queries : '—' },
      ].map(item => <Col xs={24} sm={12} xl={6} key={item.title}><Card><Statistic title={item.title} value={item.value} loading={isPending} /></Card></Col>)}
    </Row>
    {data && nodes.some(node => node.alive === false || node.decommissioned || node.error) && <Alert type="warning" showIcon title="Some compute nodes need attention" description="Review node status and expand affected rows for details." />}
    <Card title="Compute nodes" extra={<Space wrap><Typography.Text type="secondary">Last updated: {dataUpdatedAt ? formatTimestamp(dataUpdatedAt) : '—'}</Typography.Text><Button loading={isFetching} onClick={() => void refetch()}>Refresh</Button></Space>}>
      <Table<ComputeNode>
        size="small"
        rowKey="id"
        columns={columns}
        dataSource={nodes}
        loading={isPending}
        scroll={{ x: 1400 }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true }}
        expandable={expandable}
        locale={{ emptyText: <Empty description={error ? 'Compute nodes unavailable' : 'No compute nodes registered in this cluster'} /> }}
      />
      <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>Usage is reported by the latest node heartbeat. Query counts are summed across nodes; one distributed query can run on several nodes.</Typography.Paragraph>
    </Card>
  </Space>;
}
