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
    <div>
      <h2>{ticket.title}</h2>
      <p>{ticket.description}</p>
      <h3>Comments</h3>
      <ul>
        {comments.map(c => (
          <li key={c.id}>{c.body_md}</li>
        ))}
      </ul>
      <form onSubmit={submit}>
        <textarea value={body} onChange={(e) => setBody(e.target.value)} required />
        <button type="submit">Add Comment</button>
      </form>
    </div>
  );
}
