// Frontend-facing shapes simplified for UI
export interface Ticket {
  id: string; // UUID from API
  subject: string; // maps to API 'title'
  number?: string;
  status?: string;
  priority?: number;
}

export interface Comment {
  id: string;
  body: string;
}

const API_BASE = '/api';

// Local auth with HttpOnly cookie via /login
export async function login(username: string, password: string): Promise<boolean> {
  const res = await fetch(`${API_BASE}/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ username, password }),
  });
  return res.ok;
}

// API returns rich ticket objects; map down to the UI shape
export async function fetchTickets(): Promise<Ticket[]> {
  const res = await fetch(`${API_BASE}/tickets`, { credentials: 'include' });
  if (!res.ok) throw new Error('failed to load tickets');
  const data = await res.json();
  return (data as Array<Record<string, unknown>>).map((t) => ({
    id: t.id,
    subject: t.title ?? t.number ?? 'Ticket',
    number: t.number,
    status: t.status,
    priority: t.priority,
  }));
}

// No GET comments endpoint exists yet; return empty for now
export async function fetchComments(ticketId: string): Promise<Comment[]> {
  const res = await fetch(`${API_BASE}/tickets/${ticketId}/comments`, {
    credentials: 'include',
  });
  if (!res.ok) {
    if (res.status === 404) {
      // Endpoint or comments not found; treat as no comments
      return [];
    }
    const txt = await res.text().catch(() => '');
    throw new Error(`failed to load comments: ${res.status} ${txt}`);
  }
  const data = await res.json();
  return (data as Array<Record<string, unknown>>).map((c) => ({
    id: String(c.id),
    body: String(c.body),
  }));
}

export interface UploadCallbacks {
  onProgress?: (evt: { percent: number }) => void;
  onSuccess?: () => void;
  onError?: (err: Error) => void;
}

export function uploadAttachment(
  ticketId: string,
  file: File,
  cb: UploadCallbacks = {},
): Promise<void> {
  return new Promise((resolve, reject) => {
    const form = new FormData();
    form.append('file', file);

    const xhr = new XMLHttpRequest();
    xhr.open('POST', `${API_BASE}/tickets/${ticketId}/attachments`);
    xhr.withCredentials = true;

    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) {
        cb.onProgress?.({ percent: (e.loaded / e.total) * 100 });
      }
    };

    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        cb.onSuccess?.();
        resolve();
      } else {
        const err = new Error(
          `failed to upload attachment: ${xhr.status} ${xhr.responseText}`,
        );
        cb.onError?.(err);
        reject(err);
      }
    };

    xhr.onerror = () => {
      const err = new Error('failed to upload attachment');
      cb.onError?.(err);
      reject(err);
    };

    xhr.send(form);
  });
}

// Bulk update endpoint not implemented server-side; noop for now
/* eslint-disable @typescript-eslint/no-unused-vars */
export async function bulkUpdate(
  _ticketIds: string[],
  _data: Record<string, unknown>,
): Promise<void> {
  return;
}
/* eslint-enable @typescript-eslint/no-unused-vars */

export interface Me {
  id: string;
  email?: string;
  display_name?: string;
  roles?: string[];
}

export async function getMe(): Promise<Me | null> {
  const res = await fetch(`${API_BASE}/me`, { credentials: 'include' });
  if (!res.ok) return null;
  return res.json();
}

export async function createTicket(params: {
  title: string;
  description?: string;
  requesterId: string;
  priority: number;
}): Promise<{ id: string; number: string } | null> {
  const res = await fetch(`${API_BASE}/tickets`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      title: params.title,
      description: params.description || '',
      requester_id: params.requesterId,
      priority: params.priority,
    }),
  });
  if (!res.ok) return null;
  return res.json();
}

export async function addComment(ticketId: string, authorId: string, body: string): Promise<boolean> {
  const res = await fetch(`${API_BASE}/tickets/${ticketId}/comments`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ body_md: body, is_internal: false, author_id: authorId }),
  });
  return res.ok;
}

export async function logout(): Promise<void> {
  await fetch(`${API_BASE}/logout`, { method: 'POST', credentials: 'include' });
}
