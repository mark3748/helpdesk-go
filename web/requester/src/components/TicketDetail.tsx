import { useState } from 'react';
import type { FormEvent } from 'react';
import { useParams } from 'react-router-dom';
import { useAuth } from 'react-oidc-context';
import { addComment, getTicket, listComments, uploadAttachment } from '../api';
import type { Comment, Ticket } from '../api';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

export default function TicketDetail() {
  const { id } = useParams<{ id: string }>();
  const [body, setBody] = useState('');
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const auth = useAuth();
  const qc = useQueryClient();

  const ticketQuery = useQuery<Ticket>({
    queryKey: ['ticket', id],
    queryFn: () => getTicket(id!, auth.user!.access_token),
    enabled: !!id && !!auth.user,
  });

  const commentsQuery = useQuery<Comment[]>({
    queryKey: ['comments', id],
    queryFn: () => listComments(id!, auth.user!.access_token),
    enabled: !!id && !!auth.user,
  });

  const addCommentMutation = useMutation({
    mutationFn: (content: string) =>
      addComment(
        id!,
        {
          body_md: content,
          author_id: auth.user?.profile.sub || '',
          is_internal: false,
        },
        auth.user!.access_token,
      ),
    onSuccess: () => {
      setBody('');
      qc.invalidateQueries({ queryKey: ['comments', id] });
    },
  });

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (id && auth.user) {
      addCommentMutation.mutate(body);
    }
  }

  async function handleUpload(e: React.ChangeEvent<HTMLInputElement>) {
    if (!e.target.files || !e.target.files[0] || !id || !auth.user) return;
    setUploading(true);
    try {
      await uploadAttachment(id, e.target.files[0], auth.user.access_token, {
        onProgress: (evt) => setProgress(evt.percent),
      });
      alert('Uploaded');
    } catch {
      alert('Upload failed');
    } finally {
      setUploading(false);
      setProgress(0);
      e.target.value = '';
    }
  }

  if (ticketQuery.isLoading) return <p>Loading...</p>;
  const ticket = ticketQuery.data!;
  const comments = commentsQuery.data || [];
  const commentsLoading = commentsQuery.isLoading;

  return (
    <div className="mx-auto max-w-2xl space-y-4 p-4">
      <h2 className="text-2xl font-semibold">{ticket.title}</h2>
      <p>{ticket.description}</p>
      <h3 className="text-xl font-semibold">Comments</h3>
      <input type="file" onChange={handleUpload} />
      {uploading && <progress value={progress} max={100} className="w-full" />}
      {commentsLoading ? (
        <p>Loading...</p>
      ) : (
        <ul className="space-y-2">
          {comments.map(c => (
            <li key={c.id} className="rounded border p-2">
              {c.body_md}
            </li>
          ))}
        </ul>
      )}
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
          disabled={addCommentMutation.isPending}
        >
          {addCommentMutation.isPending ? 'Adding…' : 'Add Comment'}
        </button>
      </form>
    </div>
  );
}
