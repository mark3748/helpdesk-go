import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from 'react-oidc-context';
import { listTickets } from '../api';
import type { Ticket } from '../api';

export default function TicketList() {
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const auth = useAuth();

  useEffect(() => {
    if (auth.user) {
      listTickets(auth.user.access_token).then(setTickets).catch(console.error);
    }
  }, [auth.user]);

  return (
    <div className="mx-auto max-w-2xl space-y-4 p-4">
      <h2 className="text-2xl font-semibold">My Tickets</h2>
      <ul className="space-y-2">
        {tickets.map(t => (
          <li key={t.id} className="border-b pb-2 last:border-b-0">
            <Link className="text-blue-600 hover:underline" to={`/tickets/${t.id}`}>
              {t.title}
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
