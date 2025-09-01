import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import { useParams } from 'react-router-dom';
import { useAuth } from 'react-oidc-context';
import { addComment, getTicket, listComments } from '../api';
import type { Comment, Ticket } from '../api';

export default function TicketDetail() {
  const { id } = useParams<{ id: string }>();
  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [comments, setComments] = useState<Comment[]>([]);
  const [body, setBody] = useState('');
  const auth = useAuth();

  useEffect(() => {
    if (id && auth.user) {
      getTicket(id, auth.user.access_token).then(setTicket).catch(console.error);
      listComments(id, auth.user.access_token).then(setComments).catch(console.error);
    }
  }, [id, auth.user]);

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (id && auth.user) {
      await addComment(
        id,
        {
          body_md: body,
          author_id: auth.user.profile.sub || '',
          is_internal: false,
        },
        auth.user.access_token,
      );
      setBody('');
      const c = await listComments(id, auth.user.access_token);
      setComments(c);
    }
  }

  if (!ticket) return <p>Loading...</p>;

  return (
    <div className="mx-auto max-w-2xl space-y-4 p-4">
      <h2 className="text-2xl font-semibold">{ticket.title}</h2>
      <p>{ticket.description}</p>
      <h3 className="text-xl font-semibold">Comments</h3>
      <ul className="space-y-2">
        {comments.map(c => (
          <li key={c.id} className="rounded border p-2">
            {c.body_md}
          </li>
        ))}
      </ul>
      <form onSubmit={submit} className="space-y-2">
        <textarea
          className="w-full rounded border p-2"
          value={body}
          onChange={(e) => setBody(e.target.value)}
          required
        />
        <button
          className="rounded bg-blue-600 px-4 py-2 font-medium text-white"
          type="submit"
        >
          Add Comment
        </button>
      </form>
    </div>
  );
}
