import { useEffect, useState } from 'react';
import { Table, Button, Space, Tag } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTickets, subscribeEvents } from '../../api';
import type { Ticket } from '../../api';
import CreateTicketModal from './CreateTicketModal';

export default function QueueList() {
  const navigate = useNavigate();
  const [connected, setConnected] = useState(true);
  const [showNew, setShowNew] = useState(false);
  const [items, setItems] = useState<Ticket[]>([]);
  const [cursor, setCursor] = useState<string | undefined>(undefined);
  const [nextCursor, setNextCursor] = useState<string | undefined>(undefined);
  const [filters, setFilters] = useState<Record<string, string>>({});
  const { data, refetch, isFetching } = useTickets({
    cursor,
    query: filters,
    refetchInterval: connected ? false : 5000,
  });

  useEffect(() => {
    if (data) {
      setItems((prev) => (cursor ? [...prev, ...data.items] : data.items));
      setNextCursor(data.next_cursor);
    }
  }, [data, cursor]);

  useEffect(() => {
    const sub = subscribeEvents(setConnected);
    const handler = () => {
      setItems([]);
      setCursor(undefined);
      refetch();
    };
    const offs = ['ticket_created', 'ticket_updated', 'queue_changed'].map((t) =>
      sub.on(t, handler),
    );
    return () => {
      offs.forEach((off) => off());
      sub.close();
    };
  }, [refetch]);

  const loadMore = () => {
    if (nextCursor) setCursor(nextCursor);
  };

  const applyFilter = (params: Record<string, string>) => {
    setFilters(params);
    setItems([]);
    setCursor(undefined);
    refetch();
  };

  const truncate = (s: string, n: number) => (s && s.length > n ? s.slice(0, n - 1) + '…' : s);

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
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true as any,
      render: (v?: string) => (v ? <span title={v}>{truncate(String(v), 120)}</span> : null),
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
      <Space>
        <Button onClick={() => applyFilter({ status: 'open|pending' })}>
          Open|Pending
        </Button>
        <Button onClick={() => applyFilter({ assignee: 'me' })}>Mine</Button>
        <Button onClick={() => applyFilter({ aging: '3d' })}>Aging&gt;3d</Button>
      </Space>
      <Button type="primary" onClick={() => setShowNew(true)}>
        New Ticket
      </Button>
      <Table
        rowKey={(r: any) => String((r as any).id)}
        columns={columns as any}
        dataSource={items}
        pagination={false}
        style={{ background: '#fff' }}
      />
      {nextCursor && (
        <Button onClick={loadMore} disabled={isFetching}>
          Load more
        </Button>
      )}
      <CreateTicketModal
        open={showNew}
        onClose={() => setShowNew(false)}
        onCreated={() => {
          setShowNew(false);
          setItems([]);
          setCursor(undefined);
          refetch();
        }}
      />
    </Space>
  );
}
