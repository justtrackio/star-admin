import { useState } from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { Alert, Button, Card, Descriptions, Empty, Input, Select, Space, Table, Tag, Typography } from 'antd';
import type { TableColumnsType } from 'antd';
import type { Dictionary } from '../api/schema';
import { useDictionaries } from '../hooks/useDictionaries';
import { formatTimestamp } from '../utils/formatTimestamp';

export const Route = createFileRoute('/dictionaries')({ component: DictionariesPage });

const statusColors: Record<string, string> = {
  FINISHED: 'green', REFRESHING: 'blue', COMMITTING: 'blue', CANCELLED: 'red', UNINITIALIZED: 'default',
};

const columns: TableColumnsType<Dictionary> = [
  { title: 'Name', dataIndex: 'name', sorter: (a, b) => a.name.localeCompare(b.name) },
  { title: 'Database', dataIndex: 'database', sorter: (a, b) => a.database.localeCompare(b.database) },
  { title: 'Source', dataIndex: 'source' },
  { title: 'Status', dataIndex: 'status', render: (status: string) => <Tag color={statusColors[status] ?? 'default'}>{status || 'Unknown'}</Tag> },
  { title: 'Last successful refresh finished', dataIndex: 'lastRefreshFinished', sorter: (a, b) => a.lastRefreshFinished.localeCompare(b.lastRefreshFinished), render: (value: string) => formatTimestamp(value) },
  { title: 'Next refresh', dataIndex: 'nextRefresh', render: (value: string) => value || '—' },
  { title: 'Refresh error', dataIndex: 'error', width: 300, render: (value: string) => value ? <Typography.Text type="danger">{value}</Typography.Text> : '—' },
];

function DictionaryDetails({ dictionary }: { dictionary: Dictionary }) {
  return <Descriptions size="small" column={{ xs: 1, md: 2 }} items={[
    { key: 'id', label: 'ID', children: dictionary.id || '—' },
    { key: 'catalog', label: 'Source catalog', children: dictionary.catalog || '—' },
    { key: 'keys', label: 'Key columns', children: dictionary.keys || '—' },
    { key: 'values', label: 'Value columns', children: dictionary.values || '—' },
    { key: 'started', label: 'Reported last successful refresh start', children: formatTimestamp(dictionary.lastRefreshStarted) },
    { key: 'memory', label: 'Approximate memory per node (bytes)', children: <span style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{dictionary.memoryUsage || '—'}</span> },
  ]} />;
}

const expandable = { expandedRowRender: (dictionary: Dictionary) => <DictionaryDetails dictionary={dictionary} /> };

function DictionariesPage() {
  const { data, error, isPending, isFetching, dataUpdatedAt, refetch } = useDictionaries();
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState<string>();
  const dictionaries = data?.dictionaries ?? [];
  // oxlint-disable-next-line unicorn/no-array-sort
  const statuses = [...new Set(dictionaries.map(dictionary => dictionary.status).filter(Boolean))].sort();
  const filtered = dictionaries.filter(dictionary =>
    (!status || dictionary.status === status) &&
    `${dictionary.name} ${dictionary.database} ${dictionary.source}`.toLowerCase().includes(search.trim().toLowerCase())
  );

  return <Space orientation="vertical" size="large" style={{ width: '100%' }}>
    <div>
      <Typography.Title level={2}>Dictionaries</Typography.Title>
      <Typography.Paragraph type="secondary">Dictionary refresh status across the connected cluster. Live updates every 5 seconds. Expand a row for source columns and memory details.</Typography.Paragraph>
    </div>
    {error && <Alert type="error" showIcon title="Could not update dictionaries" description={error.message} action={<Button onClick={refetch}>Retry</Button>} />}
    {error && data && <Alert type="warning" showIcon title="Showing the last successful snapshot. Dictionary status may be out of date." />}
    <Card title={`Dictionaries${data ? ` (${dictionaries.length})` : ''}`} extra={<Space wrap>
      <Typography.Text type="secondary">Last updated: {dataUpdatedAt ? formatTimestamp(dataUpdatedAt) : '—'}</Typography.Text>
      <Button loading={isFetching} onClick={refetch}>Reload</Button>
    </Space>}>
      <Space wrap style={{ marginBottom: 16 }}>
        <Input placeholder="Search by name, database or source" aria-label="Search dictionaries" allowClear value={search} onChange={event => setSearch(event.target.value)} style={{ width: 300, maxWidth: '100%' }} />
        <Select placeholder="All statuses" aria-label="Filter dictionaries by status" allowClear value={status} onChange={setStatus} options={statuses.map(value => ({ label: value, value }))} style={{ minWidth: 180 }} />
      </Space>
      <Table<Dictionary>
        size="small"
        rowKey="id"
        columns={columns}
        dataSource={filtered}
        loading={isPending}
        scroll={{ x: 1500 }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: total => `${total} dictionaries` }}
        expandable={expandable}
        locale={{ emptyText: <Empty description={error ? 'Dictionaries unavailable' : dictionaries.length ? 'No dictionaries match your filters' : 'No dictionaries in this cluster'} /> }}
      />
      <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>Status and errors are reported separately: a finished dictionary can still report a failed refresh that rolled back to previously cached data. Reload reads metadata only.</Typography.Paragraph>
    </Card>
  </Space>;
}
