import { useState } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation, useQueryClient } from '@tanstack/react-query';
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
  const qc = useQueryClient();

  const createMutation = useMutation({
    mutationFn: async () => {
      const t = await createTicket(
        {
          title,
          description,
          status: 'New',
          category,
          subcategory,
          requester_id: '',
          priority,
          urgency,
        },
        undefined,
      );
      if (attachment) {
        try {
          await uploadAttachment(t.id!, attachment, undefined);
        } catch {
          alert('Failed to upload attachment');
        }
      }
      return t;
    },
    onSuccess: (t) => {
      alert('Ticket submitted');
      setTitle('');
      setDescription('');
      setCategory('');
      setSubcategory('');
      setPriority(3);
      setUrgency(3);
      setAttachment(null);
      qc.invalidateQueries({ queryKey: ['tickets'] });
      nav(`/tickets/${t.id}`);
    },
    onError: () => {
      alert('Failed to create ticket');
    },
  });

  function submit(e: FormEvent) {
    e.preventDefault();
    createMutation.mutate();
  }

  return (
    <form onSubmit={submit} className="mx-auto max-w-2xl space-y-4 p-4">
      {!hideTitle && (
        <div className="flex flex-col">
          <label className="mb-1 font-medium">Title</label>
          <input
            className="rounded border p-2"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            required
          />
        </div>
      )}
      <div className="flex flex-col">
        <label className="mb-1 font-medium">Description</label>
        <textarea
          className="rounded border p-2"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          required
        />
      </div>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {!hideCategory && (
          <div className="flex flex-col">
            <label className="mb-1 font-medium">Category</label>
            <input
              className="rounded border p-2"
              value={category}
              onChange={(e) => setCategory(e.target.value)}
            />
          </div>
        )}
        <div className="flex flex-col">
          <label className="mb-1 font-medium">Subcategory</label>
          <input
            className="rounded border p-2"
            value={subcategory}
            onChange={(e) => setSubcategory(e.target.value)}
          />
        </div>
      </div>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="flex flex-col">
          <label className="mb-1 font-medium">Priority</label>
          <select
            className="rounded border p-2"
            value={priority}
            onChange={(e) => setPriority(Number(e.target.value))}
          >
            <option value={1}>1 - Critical</option>
            <option value={2}>2 - High</option>
            <option value={3}>3 - Medium</option>
            <option value={4}>4 - Low</option>
          </select>
        </div>
        <div className="flex flex-col">
          <label className="mb-1 font-medium">Urgency</label>
          <select
            className="rounded border p-2"
            value={urgency}
            onChange={(e) => setUrgency(Number(e.target.value))}
          >
            <option value={1}>1 - Critical</option>
            <option value={2}>2 - High</option>
            <option value={3}>3 - Medium</option>
            <option value={4}>4 - Low</option>
          </select>
        </div>
      </div>
      <div className="flex flex-col">
        <label className="mb-1 font-medium">Attachment</label>
        <input
          className="rounded border p-2"
          type="file"
          onChange={(e) => setAttachment(e.target.files?.[0] || null)}
        />
      </div>
      <button
        className="rounded bg-blue-600 px-4 py-2 font-medium text-white"
        type="submit"
        disabled={createMutation.isPending}
      >
        {createMutation.isPending ? 'Submitting…' : 'Submit'}
      </button>
    </form>
  );
}
