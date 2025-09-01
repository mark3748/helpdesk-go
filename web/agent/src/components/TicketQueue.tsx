import { useEffect, useState } from 'react';
import type { Ticket } from '../api';
import { fetchTickets, bulkUpdate } from '../api';
import { Table, Button, Space, Tag } from 'antd';

interface Props {
  onOpen(ticket: Ticket): void;
  onNewTicket(): void;
}

export default function TicketQueue({ onOpen, onNewTicket }: Props) {
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  async function load() {
    const data = await fetchTickets();
    setTickets(data);
  }

  useEffect(() => {
    load().catch(console.error);
  }, []);

  const columns = [
    {
      title: 'Number',
      dataIndex: 'number',
      key: 'number',
      width: 140,
    },
    {
      title: 'Subject',
      dataIndex: 'subject',
      key: 'subject',
      render: (_: any, record: Ticket) => (
        <Button type="link" onClick={() => onOpen(record)}>{record.subject}</Button>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 140,
      render: (v?: string) => v ? <Tag>{v}</Tag> : null,
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
      sorter: (a: Ticket, b: Ticket) => (a.priority || 0) - (b.priority || 0),
    },
  ];

  async function handleBulkEdit() {
    if (selectedRowKeys.length === 0) return;
    await bulkUpdate(selectedRowKeys as string[], { status: 'open' });
  }

  return (
    <Space direction="vertical" style={{ width: '100%' }}>
      <Space>
        <Button type="primary" onClick={onNewTicket}>New Ticket</Button>
        <Button onClick={handleBulkEdit} disabled={!selectedRowKeys.length}>Bulk Edit</Button>
        <Button onClick={load}>Refresh</Button>
      </Space>
      <Table
        rowKey={(r) => r.id}
        columns={columns as any}
        dataSource={tickets}
        pagination={{ pageSize: 10 }}
        rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }}
        style={{ background: '#fff' }}
      />
    </Space>
  );
}
