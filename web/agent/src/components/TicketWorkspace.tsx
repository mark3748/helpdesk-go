import { useEffect, useState } from 'react';
import TicketQueue from './TicketQueue';
import type { Ticket, Comment } from '../api';
import { fetchComments, uploadAttachment } from '../api';

export default function TicketWorkspace() {
  const [tabs, setTabs] = useState<Ticket[]>([]);
  const [activeId, setActiveId] = useState<number | null>(null);

  function openTicket(ticket: Ticket) {
    setTabs((prev) => {
      if (prev.find((t) => t.id === ticket.id)) return prev;
      return [...prev, ticket];
    });
    setActiveId(ticket.id);
  }

  function closeTicket(id: number) {
    setTabs((prev) => prev.filter((t) => t.id !== id));
    if (activeId === id) {
      const remaining = tabs.filter((t) => t.id !== id);
      setActiveId(remaining.length ? remaining[0].id : null);
    }
  }

  function newTicket() {
    // placeholder for creating a new ticket
    alert('new ticket');
  }

  return (
    <div>
      <TicketQueue onOpen={openTicket} onNewTicket={newTicket} />
      <div className="tabs">
        {tabs.map((t) => (
          <button key={t.id} onClick={() => setActiveId(t.id)}>
            {t.subject}{' '}
            <span onClick={() => closeTicket(t.id)}>×</span>
          </button>
        ))}
      </div>
      <div className="tab-content">
        {tabs.map((t) =>
          t.id === activeId ? <TicketDetail key={t.id} ticket={t} /> : null
        )}
      </div>
    </div>
  );
}

function TicketDetail({ ticket }: { ticket: Ticket }) {
  const [comments, setComments] = useState<Comment[]>([]);
  useEffect(() => {
    fetchComments(ticket.id).then(setComments).catch(console.error);
  }, [ticket.id]);

  async function handleUpload(e: React.ChangeEvent<HTMLInputElement>) {
    if (e.target.files && e.target.files[0]) {
      await uploadAttachment(ticket.id, e.target.files[0]);
    }
  }

  return (
    <div>
      <h4>{ticket.subject}</h4>
      <input type="file" onChange={handleUpload} />
      <ul>
        {comments.map((c) => (
          <li key={c.id}>{c.body}</li>
        ))}
      </ul>
    </div>
  );
}
