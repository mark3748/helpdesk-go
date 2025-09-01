import { useState } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from 'react-oidc-context';
import { createTicket } from '../api';
import type { Ticket } from '../api';

interface Props {
  initial?: Partial<Ticket>;
  hideTitle?: boolean;
  hideCategory?: boolean;
}

export default function TicketForm({ initial = {}, hideTitle, hideCategory }: Props) {
  const [title, setTitle] = useState(initial.title || '');
  const [description, setDescription] = useState(initial.description || '');
  const [category, setCategory] = useState((initial as any).category || '');
  const nav = useNavigate();
  const auth = useAuth();

  async function submit(e: FormEvent) {
    e.preventDefault();
    const t = await createTicket({ title, description, status: 'New', category }, auth.user?.access_token || '');
    nav(`/tickets/${t.id}`);
  }

  return (
    <form onSubmit={submit}>
      {!hideTitle && (
        <div>
          <label>Title</label>
          <input value={title} onChange={(e) => setTitle(e.target.value)} required />
        </div>
      )}
      <div>
        <label>Description</label>
        <textarea value={description} onChange={(e) => setDescription(e.target.value)} required />
      </div>
      {!hideCategory && (
        <div>
          <label>Category</label>
          <input value={category} onChange={(e) => setCategory(e.target.value)} />
        </div>
      )}
      <button type="submit">Submit</button>
    </form>
  );
}
