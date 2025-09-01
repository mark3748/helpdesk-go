import { useState } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from 'react-oidc-context';
import { createTicket, uploadAttachment } from '../api';
import type { Ticket } from '../api';

interface Props {
  initial?: Partial<Ticket>;
  hideTitle?: boolean;
  hideCategory?: boolean;
}

export default function TicketForm({ initial = {}, hideTitle, hideCategory }: Props) {
  const [title, setTitle] = useState(initial.title || '');
  const [description, setDescription] = useState(initial.description || '');
  const [category, setCategory] = useState(initial.category || '');
  const [subcategory, setSubcategory] = useState(initial.subcategory || '');
  const [priority, setPriority] = useState(initial.priority ?? 3);
  const [urgency, setUrgency] = useState(initial.urgency ?? 3);
  const [attachment, setAttachment] = useState<File | null>(null);
  const nav = useNavigate();
  const auth = useAuth();

  async function submit(e: FormEvent) {
    e.preventDefault();
    const t = await createTicket(
      {
        title,
        description,
        status: 'New',
        category,
        subcategory,
        requester_id: auth.user?.profile.sub || '',
        priority,
        urgency,
      },
      auth.user?.access_token || '',
    );
    if (attachment) {
      await uploadAttachment(t.id, attachment, auth.user?.access_token || '');
    }
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
      <div>
        <label>Subcategory</label>
        <input value={subcategory} onChange={(e) => setSubcategory(e.target.value)} />
      </div>
      <div>
        <label>Priority</label>
        <select value={priority} onChange={(e) => setPriority(Number(e.target.value))}>
          <option value={1}>1 - Critical</option>
          <option value={2}>2 - High</option>
          <option value={3}>3 - Medium</option>
          <option value={4}>4 - Low</option>
        </select>
      </div>
      <div>
        <label>Urgency</label>
        <select value={urgency} onChange={(e) => setUrgency(Number(e.target.value))}>
          <option value={1}>1 - Critical</option>
          <option value={2}>2 - High</option>
          <option value={3}>3 - Medium</option>
          <option value={4}>4 - Low</option>
        </select>
      </div>
      <div>
        <label>Attachment</label>
        <input type="file" onChange={(e) => setAttachment(e.target.files?.[0] || null)} />
      </div>
      <button type="submit">Submit</button>
    </form>
  );
}
