export interface Ticket {
  id: number;
  subject: string;
}

export interface Comment {
  id: number;
  ticketId: number;
  body: string;
}

const API_BASE = '/api';

export async function login(username: string, password: string): Promise<boolean> {
  const res = await fetch(`${API_BASE}/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password })
  });
  return res.ok;
}

export async function fetchTickets(): Promise<Ticket[]> {
  const res = await fetch(`${API_BASE}/tickets`);
  if (!res.ok) throw new Error('failed to load tickets');
  return res.json();
}

export async function fetchComments(ticketId: number): Promise<Comment[]> {
  const res = await fetch(`${API_BASE}/tickets/${ticketId}/comments`);
  if (!res.ok) throw new Error('failed to load comments');
  return res.json();
}

export async function uploadAttachment(ticketId: number, file: File): Promise<void> {
  const form = new FormData();
  form.append('file', file);
  const res = await fetch(`${API_BASE}/tickets/${ticketId}/attachments`, {
    method: 'POST',
    body: form
  });
  if (!res.ok) throw new Error('failed to upload attachment');
}

export async function bulkUpdate(ticketIds: number[], data: Record<string, unknown>): Promise<void> {
  await fetch(`${API_BASE}/tickets/bulk`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ids: ticketIds, data })
  });
}
