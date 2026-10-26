import { useState } from 'react';
import { createFileRoute, Link } from '@tanstack/react-router';
import { Alert, Button, Card, Empty, Input, Space, Table, Typography } from 'antd';
import type { TableColumnsType } from 'antd';
import type { RoutineLoad } from '../api/schema';
import { useRoutineLoads } from '../hooks/useRoutineLoads';
import { RoutineLoadStatus } from '../components/RoutineLoadStatus';
import { formatTimestamp } from '../utils/formatTimestamp';

export const Route = createFileRoute('/routine-loads')({ component: RoutineLoadsPage });

const jobColumns: TableColumnsType<RoutineLoad> = [
  {
    title: 'Job',
    dataIndex: 'name',
    sorter: (a, b) => a.name.localeCompare(b.name),
    render: (name: string, job) => (
      <Link
        to="/routine-loads/$database/$id"
        params={{ database: job.database, id: job.id }}
        onClick={(event) => event.stopPropagation()}
      >
        {name}
      </Link>
    ),
  },
  { title: 'Database', dataIndex: 'database' },
  { title: 'Table', dataIndex: 'table' },
  { title: 'Kafka topic', dataIndex: 'topic', render: (value: string) => value || '—' },
  { title: 'State', dataIndex: 'state', render: (value: string) => <RoutineLoadStatus state={value} /> },
  { title: 'Loaded rows', dataIndex: 'loadedRows', render: (value: string) => value || '—' },
  { title: 'Error rows', dataIndex: 'errorRows', render: (value: string) => value || '—' },
  { title: 'Rows/s', dataIndex: 'rowsPerSecond', render: (value: string) => value || '—' },
  { title: 'State change reason', dataIndex: 'reason', width: 300, render: (value: string) => value || '—' },
];

function RoutineLoadsPage() {
  const navigate = Route.useNavigate();
  const jobs = useRoutineLoads();
  const [search, setSearch] = useState('');
  const query = search.trim().toLowerCase();
  const filteredJobs = (jobs.data?.jobs ?? []).filter((job) =>
    `${job.database} ${job.table} ${job.name} ${job.topic}`.toLowerCase().includes(query),
  );

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <div>
        <Typography.Title level={2}>Kafka routines</Typography.Title>
        <Typography.Paragraph type="secondary">
          Kafka Routine Load ingestion across the connected cluster. Live updates every 5 seconds.
        </Typography.Paragraph>
      </div>
      <Input
        placeholder="Search by database, table, job or topic"
        aria-label="Search Kafka routines"
        allowClear
        value={search}
        onChange={(event) => setSearch(event.target.value)}
        style={{ width: 360, maxWidth: '100%' }}
      />
      {jobs.error && (
        <Alert
          type="error"
          showIcon
          title="Could not update Kafka ingestion"
          description={jobs.error.message}
          action={<Button onClick={jobs.refetch}>Retry</Button>}
        />
      )}
      {jobs.error && jobs.data && (
        <Alert
          type="warning"
          showIcon
          title="Showing the last successful ingestion snapshot. Job states may be out of date."
        />
      )}
      <Card
        title={`Kafka ingestion${jobs.data ? ` (${jobs.data.jobs.length})` : ''}`}
        extra={
          <Typography.Text type="secondary">
            Last updated: {jobs.dataUpdatedAt ? formatTimestamp(jobs.dataUpdatedAt) : '—'}
          </Typography.Text>
        }
      >
        <Table<RoutineLoad>
          size="small"
          rowKey="id"
          columns={jobColumns}
          dataSource={filteredJobs}
          loading={jobs.isPending}
          scroll={{ x: 1500 }}
          onRow={(job) => ({
            style: { cursor: 'pointer' },
            onClick: () => {
              void navigate({ to: '/routine-loads/$database/$id', params: { database: job.database, id: job.id } });
            },
          })}
          pagination={{ defaultPageSize: 20, showSizeChanger: true }}
          locale={{
            emptyText: (
              <Empty
                description={
                  jobs.error
                    ? 'Kafka ingestion metadata unavailable'
                    : query
                      ? 'No ingestion jobs match your search'
                      : 'No Kafka Routine Load jobs visible to this account'
                }
              />
            ),
          }}
        />
        <Typography.Text type="secondary">
          Row totals belong to each job. Click a routine to see its details and consumed partition offsets. External
          Kafka connectors are not reported by Routine Load metadata.
        </Typography.Text>
      </Card>
    </Space>
  );
}
