import { useEffect, useState } from 'react';
import { Table, Button, Tag, Typography, Select } from 'antd';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useTickets, subscribeEvents } from '../../api';
import type { Ticket } from '../../api';
import CreateTicketModal from './CreateTicketModal';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

const { Text } = Typography;

export default function QueueList() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [connected, setConnected] = useState(true);
  const [showNew, setShowNew] = useState(false);
  const [items, setItems] = useState<Ticket[]>([]);
  const [cursor, setCursor] = useState<string | undefined>(undefined);
  const [nextCursor, setNextCursor] = useState<string | undefined>(undefined);

  // Sync filters from URL
  const filters: Record<string, string> = {};
  searchParams.forEach((value, key) => { filters[key] = value; });

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
    // Reset cursor when filters change. The data sync effect will handle clearing items.
    setCursor(undefined);
  }, [searchParams]);

  useEffect(() => {
    // connect
    const sub = subscribeEvents(setConnected);
    const handler = () => {
      // When a relevant event occurs, refresh the list
      setCursor(undefined);
      refetch();
    };

    // Subscribe to both creation and updates
    const unsubs = [
      sub.on('ticket_created', handler),
      sub.on('ticket_updated', handler),
      sub.on('queue_changed', handler),
    ];

    return () => {
      unsubs.forEach(u => u());
      sub.close();
    };
  }, [refetch]);

  const loadMore = () => {
    if (nextCursor) setCursor(nextCursor);
  };

  const getStatusColor = (status?: string) => {
    switch (status?.toLowerCase()) {
      case 'open': return { bg: '#FFF4E5', color: '#FF9500' }; // Orange
      case 'closed': return { bg: '#E6FFFA', color: '#00B894' }; // Green
      case 'in progress': return { bg: '#E6F7FF', color: '#0984E3' }; // Blue
      case 'on hold': return { bg: '#FFF0F0', color: '#FF7675' }; // Red
      default: return { bg: '#F3F4F8', color: '#636E72' }; // Gray
    }
  };

  const getPriorityColor = (p?: number) => {
    // 1: Critical (Red), 2: High (Volcano), 3: Medium (Gold), 4: Low (Green)
    // Mapping to screenshot styles: generic High/Normal/Low text?
    switch (p) {
      case 1: return { text: 'High', color: 'red' }; // Critical
      case 2: return { text: 'High', color: '#FF7675' };
      case 3: return { text: 'Medium', color: '#FDCB6E' };
      case 4: return { text: 'Normal', color: '#00B894' };
      default: return { text: 'Normal', color: '#00B894' };
    }
  };

  const columns = [
    {
      title: 'S.No',
      key: 'sno',
      width: 80,
      render: (_: any, __: any, index: number) => (
        <Text style={{ color: '#999' }}>{String(index + 1).padStart(2, '0')}</Text>
      ),
    },
    {
      title: 'Ticket ID',
      dataIndex: 'number',
      key: 'number',
      render: (text: string, record: Ticket) => (
        <a onClick={() => navigate(`/tickets/${record.id}`)} style={{ color: '#6B4EFF', fontWeight: 500 }}>
          #{text}
        </a>
      ),
    },
    {
      title: 'Name',
      key: 'name',
      render: (_: any, record: Ticket) => (
        <Text strong>{record.requester_id ? 'Requester ' + record.requester_id.slice(0, 4) : 'Unknown'}</Text>
      ),
    },
    {
      title: 'Subject',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
      render: (text: string) => <Text>{text}</Text>,
    },
    {
      title: 'Category',
      dataIndex: 'category',
      key: 'category',
      render: (v?: string) => <Text>{v || 'General'}</Text>,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (v: string) => {
        const style = getStatusColor(v);
        return (
          <Tag style={{
            color: style.color,
            background: style.bg,
            border: 0,
            borderRadius: 12,
            padding: '2px 12px',
            fontWeight: 600
          }}>
            {v?.toUpperCase()}
          </Tag>
        );
      },
    },
    {
      title: 'Priority',
      dataIndex: 'priority',
      key: 'priority',
      render: (v: number) => {
        const p = getPriorityColor(v);
        // Just text in the screenshot for priority, but let's keep it bold/colored text
        return <Text strong style={{ color: p.color }}>{p.text}</Text>;
      },
    },
    {
      title: 'Created',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (v: string) => <Text style={{ color: '#999' }}>{dayjs(v).fromNow()}</Text>,
    },
  ];

  return (
    <div style={{ background: '#fff', borderRadius: 12, padding: 24, boxShadow: '0 2px 8px rgba(0,0,0,0.02)' }}>
      {/* Table Top Toolbar */}
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 24 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Select defaultValue="10" style={{ width: 70 }} variant="borderless" options={[{ value: '10', label: '10' }]} />
          <Text style={{ color: '#999' }}>Entries Per Page</Text>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <Select
            defaultValue="all"
            style={{ width: 120 }}
            options={[
              { value: 'all', label: 'All Tickets' },
              { value: 'open', label: 'Open' },
              { value: 'closed', label: 'Closed' }
            ]}
            onChange={(v) => {
              if (v === 'all') navigate('/tickets');
              else navigate(`/tickets?status=${v}`);
            }}
          />
          <Button type="primary" onClick={() => setShowNew(true)} style={{ background: '#6B4EFF' }}>
            New Ticket
          </Button>
        </div>
      </div>

      <Table
        rowKey="id"
        columns={columns as any}
        dataSource={items}
        pagination={false}
        rowSelection={{ type: 'checkbox' }}
        components={{
          header: {
            cell: (props: any) => <th {...props} style={{ background: '#F9FAFB', color: '#666', fontWeight: 600, borderBottom: '1px solid #f0f0f0' }} />
          }
        }}
        onRow={() => ({
          style: { borderBottom: '1px solid #f3f3f3' }
        })}
      />

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 24 }}>
        <Text style={{ color: '#999' }}>Showing 1 to {items.length} of {items.length}</Text>
        <div style={{ display: 'flex', gap: 8 }}>
          {nextCursor && (
            <Button onClick={loadMore} loading={isFetching}>Load More</Button>
          )}
        </div>
      </div>

      <CreateTicketModal
        open={showNew}
        onClose={() => setShowNew(false)}
        onCreated={() => {
          setShowNew(false);
          setCursor(undefined);
          refetch();
        }}
      />
    </div>
  );
}
