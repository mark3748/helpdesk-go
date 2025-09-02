import { useState } from 'react';
import type { FormEvent } from 'react';
import { useParams } from 'react-router-dom';
import { useAuth } from 'react-oidc-context';
import { addComment, getTicket, listComments, uploadAttachment, listAttachments, downloadAttachment, deleteAttachment } from '../api';
import type { Comment, Ticket, Attachment } from '../api';
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

  const attachmentsQuery = useQuery<Attachment[]>({
    queryKey: ['attachments', id],
    queryFn: () => listAttachments(id!, auth.user!.access_token),
    enabled: !!id && !!auth.user,
  });

  const addCommentMutation = useMutation({
    mutationFn: (content: string) =>
      addComment(id!, content, auth.user!.access_token),
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
      qc.invalidateQueries({ queryKey: ['attachments', id] });
    } catch {
      alert('Upload failed');
    } finally {
      setUploading(false);
      setProgress(0);
      e.target.value = '';
    }
  }

  if (ticketQuery.isLoading || !ticketQuery.data) return <p>Loading...</p>;
  const ticket = ticketQuery.data;
  const comments = commentsQuery.data || [];
  const commentsLoading = commentsQuery.isLoading;
  const attachments = attachmentsQuery.data || [];
  const attachmentsLoading = attachmentsQuery.isLoading;

  return (
    <div className="mx-auto max-w-2xl space-y-4 p-4">
      <h2 className="text-2xl font-semibold">{ticket.title}</h2>
      <p>{ticket.description}</p>
      <h3 className="text-xl font-semibold">Comments</h3>
      <input type="file" onChange={handleUpload} />
      {uploading && <progress value={progress} max={100} className="w-full" />}
      <div>
        <h4 className="text-lg font-semibold">Attachments</h4>
        {attachmentsLoading ? (
          <p>Loading…</p>
        ) : attachments.length === 0 ? (
          <p className="text-sm text-gray-500">No attachments yet</p>
        ) : (
          <ul className="space-y-1">
            {attachments.map(a => (
              <li key={a.id} className="flex items-center justify-between">
                <span>
                  {a.filename} <span className="text-gray-500 text-sm">({Math.round(((a.bytes || 0) / 1024))} KB)</span>
                </span>
                <span className="space-x-2">
                  <button className="rounded bg-gray-200 px-2 py-1" onClick={() => downloadAttachment(id!, a.id, auth.user!.access_token)}>Download</button>
                  <button
                    className="rounded bg-red-600 px-2 py-1 text-white"
                    onClick={async () => {
                      await deleteAttachment(id!, a.id, auth.user!.access_token);
                      qc.invalidateQueries({ queryKey: ['attachments', id] });
                    }}
                  >
                    Delete
                  </button>
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
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
