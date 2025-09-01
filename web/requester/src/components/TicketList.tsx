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
    <div>
      <h2>My Tickets</h2>
      <ul>
        {tickets.map(t => (
          <li key={t.id}>
            <Link to={`/tickets/${t.id}`}>{t.title}</Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
