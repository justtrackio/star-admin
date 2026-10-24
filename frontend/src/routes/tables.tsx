import { useState } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { Alert, Button, Card, Empty, Input, Space, Table, Typography } from 'antd';
import type { TableColumnsType } from 'antd';
import type { ClusterTable } from '../api/schema';
import { useTables } from '../hooks/useTables';
import { formatTimestamp } from '../utils/formatTimestamp';

export const Route = createFileRoute('/tables')({ component: TablesPage });

const tableColumns: TableColumnsType<ClusterTable> = [
  { title: 'Name', dataIndex: 'name', sorter: (a, b) => a.name.localeCompare(b.name) },
  { title: 'Database', dataIndex: 'database', sorter: (a, b) => a.database.localeCompare(b.database) },
  { title: 'Engine', dataIndex: 'engine' },
  { title: 'Approximate rows', dataIndex: 'rows', render: (value: number | null) => value?.toLocaleString() ?? '—' },
  { title: 'Data size (bytes, all replicas)', dataIndex: 'dataBytes', render: (value: number | null) => value?.toLocaleString() ?? '—' },
];

const tableKey = (table: ClusterTable) => JSON.stringify([table.database, table.name]);

function TablesPage() {
  const tables = useTables();
  const [search, setSearch] = useState('');
  const query = search.trim().toLowerCase();
  const filteredTables = (tables.data?.tables ?? []).filter(table => `${table.database} ${table.name}`.toLowerCase().includes(query));

  return <Space orientation="vertical" size="large" style={{ width: '100%' }}>
    <div>
      <Typography.Title level={2}>Tables</Typography.Title>
      <Typography.Paragraph type="secondary">Regular tables across the connected cluster. Live updates every 5 seconds.</Typography.Paragraph>
    </div>
    <Input placeholder="Search by database or table" aria-label="Search tables" allowClear value={search} onChange={event => setSearch(event.target.value)} style={{ width: 360, maxWidth: '100%' }} />
    {tables.error && <Alert type="error" showIcon title="Could not update tables" description={tables.error.message} action={<Button onClick={tables.refetch}>Retry</Button>} />}
    {tables.error && tables.data && <Alert type="warning" showIcon title="Showing the last successful tables snapshot. Statistics may be out of date." />}
    <Card title={`Tables${tables.data ? ` (${tables.data.tables.length})` : ''}`} extra={<Space wrap>
      <Typography.Text type="secondary">Last updated: {tables.dataUpdatedAt ? formatTimestamp(tables.dataUpdatedAt) : '—'}</Typography.Text>
      <Button loading={tables.isFetching} onClick={tables.refetch}>Reload tables</Button>
    </Space>}>
      <Table<ClusterTable> size="small" rowKey={tableKey} columns={tableColumns} dataSource={filteredTables} loading={tables.isPending} scroll={{ x: 900 }} pagination={{ defaultPageSize: 20, showSizeChanger: true }} locale={{ emptyText: <Empty description={tables.error ? 'Tables unavailable' : query ? 'No tables match your search' : 'No regular tables visible to this account'} /> }} />
      <Typography.Text type="secondary">Counts are metadata estimates. Materialized views and system databases are excluded. Reload reads metadata only.</Typography.Text>
    </Card>
  </Space>;
}
