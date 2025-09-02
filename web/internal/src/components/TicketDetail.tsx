import { useEffect, useState } from 'react';
import { useTicket, subscribeEvents, AppEvent } from '../api';

interface Props {
  id: string;
}

export default function TicketDetail({ id }: Props) {
  const [connected, setConnected] = useState(true);
  const { data, refetch } = useTicket(id, { refetchInterval: connected ? false : 5000 });

  useEffect(() => {
    const stop = subscribeEvents((ev: AppEvent) => {
      if (ev.type === 'ticket_updated' && String((ev.data as any)?.id) === id) {
        refetch();
      }
    }, setConnected);
    return stop;
  }, [id, refetch]);

  if (!data) return <div>Loading…</div>;

  return (
    <div>
      <div style={{ marginBottom: 8 }}>Events: {connected ? 'connected' : 'disconnected'}</div>
      <h3>{String((data as any).title || (data as any).number)}</h3>
      <pre>{JSON.stringify(data, null, 2)}</pre>
    </div>
  );
}
