import { useState } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { Alert, Button, Card, Descriptions, Empty, Input, Space, Table, Typography } from 'antd';
import type { TableColumnsType } from 'antd';
import type { ResourceGroup } from '../api/schema';
import { useResourceGroups } from '../hooks/useResourceGroups';
import { formatTimestamp } from '../utils/formatTimestamp';

export const Route = createFileRoute('/resource-groups')({ component: ResourceGroupsPage });

const columns: TableColumnsType<ResourceGroup> = [
  { title: 'Name', dataIndex: 'name', sorter: (a, b) => a.name.localeCompare(b.name) },
  { title: 'ID', dataIndex: 'id' },
  { title: 'CPU weight', render: (_, group) => group.properties.cpu_weight_percent ? `${group.properties.cpu_weight_percent}%` : group.properties.cpu_weight || group.properties.cpu_core_limit || '—' },
  { title: 'Exclusive CPU', render: (_, group) => group.properties.exclusive_cpu_percent ? `${group.properties.exclusive_cpu_percent}%` : group.properties.exclusive_cpu_cores ? `${group.properties.exclusive_cpu_cores} cores` : '—' },
  { title: 'Memory limit', render: (_, group) => group.properties.mem_limit || '—' },
  { title: 'Concurrency limit', render: (_, group) => group.properties.concurrency_limit || '—' },
  { title: 'Spill threshold', render: (_, group) => group.properties.spill_mem_limit_threshold || '—' },
];

function ResourceGroupDetails({ group }: { group: ResourceGroup }) {
  return <Space orientation="vertical" style={{ width: '100%' }}>
    <Descriptions size="small" column={{ xs: 1, md: 2 }} items={Object.entries(group.properties).map(([key, value]) => ({
      key, label: key, children: value || '—',
    }))} />
    <Typography.Text strong>Classifier rules</Typography.Text>
    {group.classifiers.length ? group.classifiers.map(classifier =>
      <Typography.Paragraph key={classifier} code style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere', marginBottom: 0 }}>{classifier}</Typography.Paragraph>
    ) : <Typography.Text type="secondary">No classifier rules reported</Typography.Text>}
  </Space>;
}

const expandable = { expandedRowRender: (group: ResourceGroup) => <ResourceGroupDetails group={group} /> };

function ResourceGroupsPage() {
  const { data, error, isPending, dataUpdatedAt, refetch } = useResourceGroups();
  const [search, setSearch] = useState('');
  const groups = data?.groups ?? [];
  const filtered = groups.filter(group =>
    `${group.name} ${group.id} ${group.classifiers.join(' ')}`.toLowerCase().includes(search.trim().toLowerCase())
  );

  return <Space orientation="vertical" size="large" style={{ width: '100%' }}>
    <div>
      <Typography.Title level={2}>Resource groups</Typography.Title>
      <Typography.Paragraph type="secondary">Resource group configuration across the connected cluster. Live updates every 5 seconds. Expand a row for all limits and classifier rules.</Typography.Paragraph>
    </div>
    {error && <Alert type="error" showIcon title="Could not update resource groups" description={error.message} action={<Button onClick={refetch}>Retry</Button>} />}
    {error && data && <Alert type="warning" showIcon title="Showing the last successful snapshot. Resource group configuration may be out of date." />}
    <Card title={`Resource groups${data ? ` (${groups.length})` : ''}`} extra={<Typography.Text type="secondary">Last updated: {dataUpdatedAt ? formatTimestamp(dataUpdatedAt) : '—'}</Typography.Text>}>
      <Input placeholder="Search by name, ID or classifier" aria-label="Search resource groups" allowClear value={search} onChange={event => setSearch(event.target.value)} style={{ width: 320, maxWidth: '100%', marginBottom: 16 }} />
      <Table<ResourceGroup>
        size="small"
        rowKey="name"
        columns={columns}
        dataSource={filtered}
        loading={isPending}
        scroll={{ x: 1100 }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: total => `${total} resource groups` }}
        expandable={expandable}
        locale={{ emptyText: <Empty description={error ? 'Resource groups unavailable' : groups.length ? 'No resource groups match your search' : 'No resource groups in this cluster'} /> }}
      />
      <Typography.Paragraph type="secondary" style={{ marginTop: 16, marginBottom: 0 }}>Limits apply per execution node. These values show configured limits, not current resource usage. Missing values display a dash.</Typography.Paragraph>
    </Card>
  </Space>;
}
