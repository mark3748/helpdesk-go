export interface Ticket {
  id: string;
  title: string;
  description: string;
  status: string;
  requester_id?: string;
  priority?: number;
  created_at?: string;
  category?: string;
}

export interface Comment {
  id: string;
  ticket_id: string;
  author_id?: string;
  body_md: string;
  created_at?: string;
  is_internal?: boolean;
}

const API_BASE = import.meta.env.VITE_API_BASE || '/api';

async function apiFetch(path: string, opts: RequestInit = {}, token?: string) {
  const headers: Record<string, string> = {
    ...(opts.headers as Record<string, string>),
  };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const res = await fetch(`${API_BASE}${path}`, { ...opts, headers });
  if (!res.ok) throw new Error(await res.text());
  if (res.status === 204) return null;
  return res.json();
}

export async function listTickets(token: string): Promise<Ticket[]> {
  return apiFetch('/tickets', {}, token);
}

export async function getTicket(id: string, token: string): Promise<Ticket> {
  return apiFetch(`/tickets/${id}`, {}, token);
}

export async function createTicket(data: Partial<Ticket>, token: string): Promise<Ticket> {
  return apiFetch('/tickets', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  }, token);
}

export async function listComments(id: string, token: string): Promise<Comment[]> {
  return apiFetch(`/tickets/${id}/comments`, {}, token);
}

export interface CommentInput {
  body_md: string;
  author_id: string;
  is_internal: boolean;
}

export async function addComment(id: string, data: CommentInput, token: string): Promise<Comment> {
  return apiFetch(`/tickets/${id}/comments`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  }, token);
}
