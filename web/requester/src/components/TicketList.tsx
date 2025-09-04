import { Link } from 'react-router-dom';
import { useAuth } from 'react-oidc-context';
import { useState, useEffect } from 'react';
import { listTickets } from '../api';
import type { Ticket } from '../api';

export default function TicketList() {
  const auth = useAuth();
  const [items, setItems] = useState<Ticket[]>([]);
  const [nextCursor, setNextCursor] = useState<string | undefined>();
  const [loading, setLoading] = useState(false);

  const load = async (c?: string, reset = false) => {
    if (!auth.user) return;
    setLoading(true);
    try {
      const data = await listTickets(auth.user.access_token, c);
      setItems((prev) => (reset ? data.items : [...prev, ...data.items]));
      setNextCursor(data.next_cursor);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (auth.user) {
      load(undefined, true);
    }
  }, [auth.user]);

  return (
    <div className="mx-auto max-w-2xl space-y-4 p-4">
      <h2 className="text-2xl font-semibold">My Tickets</h2>
      <ul className="space-y-2">
        {items.map(t => (
          <li key={t.id} className="border-b pb-2 last:border-b-0">
            <Link className="text-blue-600 hover:underline" to={`/tickets/${t.id}`}>
              {t.title}
            </Link>
          </li>
        ))}
      </ul>
      {nextCursor && (
        <button
          className="rounded bg-gray-200 px-4 py-2"
          onClick={() => load(nextCursor)}
          disabled={loading}
        >
          Load more
        </button>
      )}
    </div>
  );
}
