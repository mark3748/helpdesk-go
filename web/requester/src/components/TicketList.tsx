import { Link } from 'react-router-dom';
import { useAuth } from 'react-oidc-context';
import { useQuery } from '@tanstack/react-query';
import { listTickets } from '../api';
import type { Ticket } from '../api';

export default function TicketList() {
  const auth = useAuth();
  const { data: tickets = [], isLoading } = useQuery<Ticket[]>({
    queryKey: ['tickets'],
    queryFn: () => listTickets(auth.user!.access_token),
    enabled: !!auth.user,
  });

  if (isLoading) return <p>Loading...</p>;

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
