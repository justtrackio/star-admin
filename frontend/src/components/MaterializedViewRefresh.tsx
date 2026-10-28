import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { Alert, Button, DatePicker, Form, Modal, Space, Typography } from 'antd';
import { apiClient } from '../api/client';
import type { ApiError } from '../api/client';

export function MaterializedViewRefresh({ database, id, disabled, onSuccess }: {
  database: string;
  id: string;
  disabled: boolean;
  onSuccess: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const refresh = useMutation<{ status: string }, ApiError, { startDate: string; endDate: string }>({
    mutationFn: (range) => apiClient.post(
      `/api/materialized-views/${encodeURIComponent(database)}/${encodeURIComponent(id)}/refresh`, range,
    ),
    onSuccess: () => {
      setOpen(false);
      onSuccess();
    },
  });

  return (
    <>
      <Button disabled={disabled} onClick={() => { refresh.reset(); setOpen(true); }}>
        Force refresh…
      </Button>
      <Modal
        title="Force refresh partitions"
        open={open}
        okText="Force refresh"
        confirmLoading={refresh.isPending}
        okButtonProps={{ disabled: disabled || !startDate || !endDate || startDate > endDate }}
        cancelButtonProps={{ disabled: refresh.isPending }}
        closable={!refresh.isPending}
        maskClosable={!refresh.isPending}
        keyboard={!refresh.isPending}
        onCancel={() => setOpen(false)}
        onOk={() => refresh.mutate({ startDate, endDate })}
      >
        <Space orientation="vertical" style={{ width: '100%' }}>
          <Typography.Text>
            Dates refer to partition keys in cluster time. Both the start and end dates are included.
            Whole partitions are refreshed, even if their data has not changed.
          </Typography.Text>
          <Form layout="vertical">
            <Form.Item label="Date range (both dates included)" htmlFor="refresh-start">
              <DatePicker.RangePicker
                id={{ start: 'refresh-start', end: 'refresh-end' }}
                format="YYYY-MM-DD"
                placeholder={['Start date (included)', 'End date (included)']}
                style={{ width: '100%' }}
                disabled={refresh.isPending}
                onChange={(_, dates) => {
                  setStartDate(dates[0]);
                  setEndDate(dates[1]);
                }}
              />
            </Form.Item>
          </Form>
          {startDate && endDate && startDate > endDate && (
            <Alert type="error" title="End date must be on or after start date." />
          )}
          {refresh.error && <Alert type="error" showIcon title="Could not submit refresh" description={refresh.error.message} />}
        </Space>
      </Modal>
    </>
  );
}
