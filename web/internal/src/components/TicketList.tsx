import { useEffect, useState } from 'react';
import { List } from 'antd';
import { useTickets, subscribeEvents, AppEvent } from '../api';

export default function TicketList() {
  const [connected, setConnected] = useState(true);
  const { data, refetch } = useTickets({
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

  return (
    <div>
      <div style={{ marginBottom: 8 }}>Events: {connected ? 'connected' : 'disconnected'}</div>
      <List
        dataSource={data || []}
        renderItem={(t) => (
          <List.Item key={String((t as any).id)}>
            {String((t as any).title || (t as any).number)}
          </List.Item>
        )}
      />
    </div>
  );
}
