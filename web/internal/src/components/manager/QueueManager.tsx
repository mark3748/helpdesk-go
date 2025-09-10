import { useEffect, useMemo, useState } from 'react';
import { Table, Button, Space, Tag, Select, Typography } from 'antd';
import { apiFetch } from '../../shared/api';
import { useNavigate } from 'react-router-dom';
import { useMutation } from '@tanstack/react-query';
import { useTickets, subscribeEvents } from '../../api';
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
    const sub = subscribeEvents(setConnected);
    const handler = () => refetch();
    const offs = ['ticket_created', 'ticket_updated', 'queue_changed'].map((t) =>
      sub.on(t, handler),
    );
    return () => {
      offs.forEach((off) => off());
      sub.close();
    };
  }, [refetch]);

  const mutate = useMutation({
    mutationFn: ({ id, data }: { id: string; data: { assignee_id?: string | null; priority?: number } }) =>
      updateTicket(id, data),
    onSuccess: () => refetch(),
  });

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
      title: 'Assignee',
      dataIndex: 'assignee_id',
      key: 'assignee_id',
      render: (v: string | undefined, record: any) => <AssigneePicker value={v} onChange={(val) => mutate.mutate({ id: String(record.id), data: { assignee_id: val || null } })} />,
    },
    {
      title: 'Requester',
      dataIndex: 'requester',
      key: 'requester',
      width: 220,
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

function AssigneePicker({ value, onChange }: { value?: string; onChange: (v?: string) => void }) {
  const [opts, setOpts] = useState<{ value: string; label: string }[]>([]);
  const [fetching, setFetching] = useState(false);
  useEffect(() => {
    (async () => {
      if (value) {
        try {
          const u = await apiFetch<any>(`/users/${encodeURIComponent(value)}`);
          const label = u.display_name || u.email || u.username || u.id;
          setOpts([{ value: String(u.id), label }]);
        } catch {
          // Error fetching user, ignore
        }
      }
    })();
  }, [value]);
  const search = useMemo(() => {
    let t: number | undefined;
    const run = async (q: string) => {
      if (!q) { setOpts([]); return; }
      setFetching(true);
      try {
        const users = await apiFetch<any[]>(`/users?q=${encodeURIComponent(q)}`);
        setOpts(users.map(u => ({ value: String(u.id), label: u.display_name || u.email || u.username || u.id })));
      } finally { setFetching(false); }
    };
    const deb = (q: string) => { if (t) window.clearTimeout(t); t = window.setTimeout(() => run(q), 300) as unknown as number; };
    return deb;
  }, []);
  return (
    <Select
      showSearch
      allowClear
      value={value}
      filterOption={false}
      onSearch={search}
      options={opts}
      loading={fetching}
      placeholder="Search user"
      style={{ minWidth: 220 }}
      onChange={(v) => onChange(v || undefined)}
    />
  );
}
