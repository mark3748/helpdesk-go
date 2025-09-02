import { useEffect, useState } from 'react';
import { Table, Button, Space, Tag } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTickets, subscribeEvents } from '../../api';
import type { AppEvent } from '../../api';
import CreateTicketModal from './CreateTicketModal';

export default function QueueList() {
  const navigate = useNavigate();
  const [connected, setConnected] = useState(true);
  const [showNew, setShowNew] = useState(false);
  const { data: tickets, refetch } = useTickets({
    refetchInterval: connected ? false : 5000,
  });

  useEffect(() => {
    const stop = subscribeEvents((ev: AppEvent) => {
      if (['ticket_created', 'ticket_updated', 'queue_changed'].includes(ev.type)) {
        refetch();
      }
    }, setConnected);
    return stop;
  }, [refetch]);

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
      title: 'Priority',
      dataIndex: 'priority',
      key: 'priority',
      width: 120,
      render: (p?: number) => {
        const map: Record<number, { text: string; color: string }> = {
          1: { text: 'Critical', color: 'red' },
          2: { text: 'High', color: 'volcano' },
          3: { text: 'Medium', color: 'gold' },
          4: { text: 'Low', color: 'green' },
        };
        const m = p ? map[p] : undefined;
        return m ? <Tag color={m.color}>{m.text}</Tag> : null;
      },
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
    </Space>
  );
}
