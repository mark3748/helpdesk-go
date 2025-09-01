import { useEffect, useState } from 'react';
import type { Ticket } from '../api';
import { fetchTickets, bulkUpdate } from '../api';

interface Props {
  onOpen(ticket: Ticket): void;
  onNewTicket(): void;
}

export default function TicketQueue({ onOpen, onNewTicket }: Props) {
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [checked, setChecked] = useState<Record<number, boolean>>({});

  useEffect(() => {
    fetchTickets().then(setTickets).catch(console.error);
  }, []);

  useEffect(() => {
    function handler(e: KeyboardEvent) {
      if (e.key === 'j') {
        setSelectedIndex((i) => Math.min(i + 1, tickets.length - 1));
      }
      if (e.key === 'k') {
        setSelectedIndex((i) => Math.max(i - 1, 0));
      }
      if (e.key === 'n') {
        onNewTicket();
      }
      if (e.key === 'Enter') {
        const t = tickets[selectedIndex];
        if (t) onOpen(t);
      }
    }
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [tickets, selectedIndex, onNewTicket, onOpen]);

  function toggleCheck(id: number) {
    setChecked((prev) => ({ ...prev, [id]: !prev[id] }));
  }

  async function handleBulkEdit() {
    const ids = Object.keys(checked)
      .filter((id) => checked[Number(id)])
      .map(Number);
    if (ids.length) {
      await bulkUpdate(ids, { status: 'open' });
    }
  }

  return (
    <div>
      <h3>Tickets</h3>
      <button onClick={handleBulkEdit}>Bulk Edit</button>
      <ul>
        {tickets.map((t, idx) => (
          <li
            key={t.id}
            style={{ background: idx === selectedIndex ? '#eee' : 'transparent' }}
          >
            <input
              type="checkbox"
              checked={!!checked[t.id]}
              onChange={() => toggleCheck(t.id)}
            />
            <span onClick={() => onOpen(t)}>{t.subject}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
