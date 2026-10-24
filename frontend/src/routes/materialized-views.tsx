import { useState } from 'react';
import { createFileRoute, Link } from '@tanstack/react-router';
import { Alert, Button, Card, Empty, Input, Select, Space, Table, Tag, Tooltip, Typography } from 'antd';
import type { TableColumnsType } from 'antd';
import type { MaterializedView } from '../api/schema';
import { useMaterializedViews } from '../hooks/useMaterializedViews';
import { MaterializedViewRefreshStatus, MaterializedViewStatus } from '../components/MaterializedViewStatus';
import { formatRefreshSchedule } from '../utils/formatRefreshSchedule';
import { formatTimestamp } from '../utils/formatTimestamp';

export const Route = createFileRoute('/materialized-views')({ component: MaterializedViewsPage });

const columns: TableColumnsType<MaterializedView> = [
  { title: 'Name', dataIndex: 'name', sorter: (a, b) => a.name.localeCompare(b.name), render: (name: string, view) => <Link to="/materialized-views/$database/$id" params={{ database: view.database, id: view.id }}>{name}</Link> },
  { title: 'Database', dataIndex: 'database', sorter: (a, b) => a.database.localeCompare(b.database) },
  { title: 'Refresh type', dataIndex: 'refreshType', render: (value: string) => value ? <Tag>{value}</Tag> : '—' },
  { title: 'Refresh schedule', render: (_, view) => <Tooltip title={view.refreshPolicy || undefined}>{formatRefreshSchedule(view)}</Tooltip> },
  { title: 'Status', render: (_, view) => <MaterializedViewStatus active={view.active} reason={view.inactiveReason} /> },
  { title: 'Last refresh', dataIndex: 'lastRefreshState', render: (value: string, view) => <MaterializedViewRefreshStatus state={value} error={view.lastRefreshError} /> },
  { title: 'Last successful refresh at', dataIndex: 'lastSuccessfulRefreshFinished', sorter: (a, b) => a.lastSuccessfulRefreshFinished.localeCompare(b.lastSuccessfulRefreshFinished), render: (value: string) => <Tooltip title={value ? 'Completion time of the most recent refresh with SUCCESS status. Pending and skipped attempts do not replace it.' : 'No successful refresh was found in the available metadata or retained refresh history.'}>{formatTimestamp(value)}</Tooltip> },
  { title: 'Last refresh duration', dataIndex: 'lastRefreshDuration', align: 'right', sorter: (a, b) => (a.lastRefreshDuration ?? -1) - (b.lastRefreshDuration ?? -1), render: (value: number | null) => value == null ? '—' : `${value.toLocaleString(undefined, { maximumFractionDigits: 3 })} s` },
  { title: 'Rows (approx.)', dataIndex: 'rows', align: 'right', sorter: (a, b) => (a.rows ?? -1) - (b.rows ?? -1), render: (value: number | null) => value?.toLocaleString() ?? '—' },
];

function MaterializedViewsPage() {
  const { data, error, isPending, dataUpdatedAt, refetch } = useMaterializedViews();
  const [search, setSearch] = useState('');
  const [database, setDatabase] = useState<string>();
  const views = data?.views ?? [];
  // Sort a fresh array; toSorted is outside this project's ES2022 target.
  // oxlint-disable-next-line unicorn/no-array-sort
  const databases = [...new Set(views.map(view => view.database))].sort();
  const filteredViews = views.filter(view =>
    (!database || view.database === database) &&
    `${view.database}.${view.name}`.toLowerCase().includes(search.trim().toLowerCase())
  );

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <div>
        <Typography.Title level={2}>Materialized views</Typography.Title>
        <Typography.Paragraph type="secondary">Views across all databases in the connected cluster. Updates automatically every 5 seconds. Click a view name to open its details.</Typography.Paragraph>
      </div>
      {error && <Alert type="error" showIcon title="Could not load materialized views" description={error.message} action={<Button onClick={refetch}>Retry</Button>} />}
      <Card title={`Materialized views${data ? ` (${views.length})` : ''}`} extra={
        <Typography.Text type="secondary">
          Last updated: {dataUpdatedAt ? <time dateTime={new Date(dataUpdatedAt).toISOString()}>{formatTimestamp(dataUpdatedAt)}</time> : '—'}
        </Typography.Text>
      }>
        <Space wrap style={{ marginBottom: 16 }}>
          <Input placeholder="Search by name or database" aria-label="Search materialized views" allowClear value={search} onChange={event => setSearch(event.target.value)} style={{ width: 300, maxWidth: '100%' }} />
          <Select placeholder="All databases" aria-label="Filter by database" allowClear value={database} onChange={setDatabase} options={databases.map(value => ({ label: value, value }))} style={{ minWidth: 200 }} />
        </Space>
        <Table<MaterializedView>
          size="small"
          rowKey={view => JSON.stringify([view.database, view.id, view.name])}
          columns={columns}
          dataSource={filteredViews}
          loading={isPending}
          scroll={{ x: 1400 }}
          pagination={{ defaultPageSize: 20, showSizeChanger: true, showTotal: total => `${total} views` }}
          locale={{ emptyText: <Empty description={error ? 'Materialized views unavailable' : views.length ? 'No views match your filters' : 'No materialized views in this cluster'} /> }}
        />
      </Card>
    </Space>
  );
}
