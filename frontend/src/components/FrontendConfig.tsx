import { useState } from 'react';
import { Alert, Button, Card, Input, Space, Table, Typography } from 'antd';
import { isFrontendConfigResponse } from '../api/snapshots';
import { useSnapshotStream } from '../hooks/useSnapshotStream';
import { formatTimestamp } from '../utils/formatTimestamp';

export function FrontendConfig() {
  const config = useSnapshotStream('/api/frontend-config', isFrontendConfigResponse);
  const [search, setSearch] = useState('');
  const term = search.toLowerCase();
  const settings = (config.data?.settings ?? []).filter(setting => Object.values(setting).some(value => value.toLowerCase().includes(term)));

  return <Card title={`Frontend configuration${config.data ? ` (${config.data.settings.length})` : ''}`} extra={<Typography.Text type="secondary">Last updated: {config.dataUpdatedAt ? formatTimestamp(config.dataUpdatedAt) : '—'}</Typography.Text>}>
    <Space orientation="vertical" style={{ width: '100%' }}>
      <Alert type="info" showIcon title="Source: shared SQL connection" description="These values come from the frontend serving the shared SQL connection. They can differ from the selected node and can come from another frontend between updates. Updates every 5 seconds. Credential values are masked. Configuration is read-only." />
      {config.error && <Alert type="error" showIcon title="Could not update frontend configuration" description={config.error.message} action={<Button onClick={config.refetch}>Retry</Button>} />}
      {config.error && config.data && <Alert type="warning" showIcon title="Showing the last successful configuration snapshot. Values may be out of date." />}
      <Input.Search aria-label="Search frontend configuration" placeholder="Search configuration" allowClear value={search} onChange={event => setSearch(event.target.value)} />
      <Table<Record<string, string>>
        size="small"
        rowKey="Key"
        columns={(config.data?.columns ?? []).map(column => ({
          title: column === 'Key' ? 'Name' : column,
          dataIndex: column,
          key: column,
          sorter: (a: Record<string, string>, b: Record<string, string>) => (a[column] ?? '').localeCompare(b[column] ?? ''),
          render: (value: string) => <span style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{value === '' ? <Typography.Text type="secondary">(empty)</Typography.Text> : value}</span>,
        }))}
        dataSource={settings}
        loading={config.isPending}
        scroll={{ x: 1200 }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true, hideOnSinglePage: true }}
        locale={{ emptyText: config.error ? 'Frontend configuration unavailable' : search ? 'No matching configuration values' : 'No frontend configuration values reported' }}
      />
    </Space>
  </Card>;
}
