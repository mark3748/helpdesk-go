import { useEffect, useState } from 'react';
import { Table, Button, Space, Tag, Select, Input, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useMutation } from '@tanstack/react-query';
import { useTickets, subscribeEvents } from '../../api';
import type { AppEvent } from '../../api';
import { updateTicket } from '../../shared/api';
import CreateTicketModal from './CreateTicketModal';

export default function QueueManager() {
  const navigate = useNavigate();
  const [connected, setConnected] = useState(true);
  const [showNew, setShowNew] = useState(false);
  const { data, refetch } = useTickets({
    refetchInterval: connected ? false : 5000,
  });
  const tickets = data?.items || [];

  useEffect(() => {
    const stop = subscribeEvents((ev: AppEvent) => {
      if (['ticket_created', 'ticket_updated', 'queue_changed'].includes(ev.type)) {
        refetch();
      }
    }, setConnected);
    return stop;
  }, [refetch]);

  const mutate = useMutation({
    mutationFn: ({ id, data }: { id: string; data: { assignee_id?: string | null; priority?: number } }) =>
      updateTicket(id, data),
    onSuccess: () => refetch(),
  });

  const columns = [
    {
      title: 'Number',
      dataIndex: 'number',
      key: 'number',
      width: 140,
    },
    {
      title: 'Subject',
      dataIndex: 'title',
      key: 'title',
      render: (_: any, record: any) => (
        <Button type="link" onClick={() => navigate(`/tickets/${record.id}`)}>
          {(record as any).title || (record as any).number}
        </Button>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 140,
      render: (v?: string) => (v ? <Tag>{v}</Tag> : null),
    },
    {
      title: 'Assignee',
      dataIndex: 'assignee_id',
      key: 'assignee_id',
      render: (v: string, record: any) => (
        <Input
          defaultValue={v || ''}
          placeholder="user id"
          onBlur={(e) =>
            mutate.mutate({
              id: String(record.id),
              data: { assignee_id: e.target.value || null },
            })
          }
        />
      ),
    },
    {
      title: 'Priority',
      dataIndex: 'priority',
      key: 'priority',
      render: (p: number | undefined, record: any) => (
        <Select
          value={p}
          style={{ width: 120 }}
          onChange={(val) =>
            mutate.mutate({ id: String(record.id), data: { priority: val } })
          }
          options={[
            { value: 1, label: 'Critical' },
            { value: 2, label: 'High' },
            { value: 3, label: 'Medium' },
            { value: 4, label: 'Low' },
          ]}
        />
      ),
    },
  ];

  return (
    <Space direction="vertical" style={{ width: '100%' }}>
      <div>Events: {connected ? 'connected' : 'disconnected'}</div>
      <Button type="primary" onClick={() => setShowNew(true)}>
        New Ticket
      </Button>
      <Table
        rowKey={(r: any) => String((r as any).id)}
        columns={columns as any}
        dataSource={tickets || []}
        pagination={{ pageSize: 10 }}
        style={{ background: '#fff' }}
      />
      <CreateTicketModal
        open={showNew}
        onClose={() => setShowNew(false)}
        onCreated={() => {
          setShowNew(false);
          refetch();
        }}
      />
      <div style={{ padding: 24, background: '#fff' }}>
        <Typography.Title level={5}>Reporting</Typography.Title>
        <div
          style={{
            border: '1px dashed #ccc',
            padding: 32,
            textAlign: 'center',
          }}
        >
          SLA metrics and other widgets coming soon...
        </div>
      </div>
    </Space>
  );
}
