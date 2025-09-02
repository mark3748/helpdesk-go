// Frontend-facing shapes simplified for UI
import type { paths, components } from './types/openapi';

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

export interface Attachment {
  id: string;
  filename: string;
  bytes: number;
  mime?: string | null;
  created_at?: string;
}

const API_BASE = '/api';

// Simple typed fetch helper using OpenAPI types
async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, { credentials: 'include', ...init });
  if (!res.ok) {
    const txt = await res.text().catch(() => '');
    throw new Error(`${res.status} ${txt}`);
  }
  // Some endpoints may return 204 No Content
  if (res.status === 204) return undefined as unknown as T;
  return (await res.json()) as T;
}

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
  type APITicket = paths['/tickets']['get']['responses']['200']['content']['application/json'][number];
  const data = await apiFetch<APITicket[]>('/tickets');
  return data.map((t) => ({
    id: String(t.id),
    subject: (t as any).title ?? (t as any).number ?? 'Ticket',
    number: (t as any).number,
    status: (t as any).status,
    priority: (t as any).priority as number | undefined,
  }));
}

// Fetch comments for a ticket
export async function fetchComments(ticketId: string): Promise<Comment[]> {
  type APIComment = paths['/tickets/{id}/comments']['get']['responses']['200']['content']['application/json'][number];
  const res = await fetch(`${API_BASE}/tickets/${ticketId}/comments`, { credentials: 'include' });
  if (!res.ok) {
    if (res.status === 404) return [];
    const txt = await res.text().catch(() => '');
    throw new Error(`failed to load comments: ${res.status} ${txt}`);
  }
  const data = (await res.json()) as APIComment[];
  return data.map((c) => ({ id: String(c.id), body: String((c as any).body_md) }));
}

// Fetch attachments for a ticket
export async function fetchAttachments(ticketId: string): Promise<Attachment[]> {
  type APIAttachment = components['schemas']['Attachment'];
  try {
    const data = await apiFetch<APIAttachment[]>(`/tickets/${ticketId}/attachments`);
    return data.map((a) => ({
      id: String(a.id),
      filename: String((a as any).filename),
      bytes: Number((a as any).bytes ?? 0),
      mime: (a as any).mime as string | null | undefined,
      created_at: (a as any).created_at as string | undefined,
    }));
  } catch {
    return [];
  }
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

export async function deleteAttachment(ticketId: string, attId: string): Promise<boolean> {
  try {
    await apiFetch(`/tickets/${ticketId}/attachments/${attId}`, { method: 'DELETE' });
    return true;
  } catch {
    return false;
  }
}

export async function downloadAttachment(ticketId: string, attId: string): Promise<void> {
  const res = await fetch(`${API_BASE}/tickets/${ticketId}/attachments/${attId}`, {
    credentials: 'include',
    redirect: 'follow',
  });
  if (!res.ok) throw new Error(await res.text());
  const blob = await res.blob();
  const cd = res.headers.get('Content-Disposition') || '';
  const m = /filename\*=UTF-8''([^;]+)|filename="?([^";]+)"?/i.exec(cd);
  const fname = m ? decodeURIComponent(m[1] || m[2] || 'attachment') : 'attachment';
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = fname;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
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

export type Me = components['schemas']['AuthUser'];

export async function getMe(): Promise<Me | null> {
  try {
    return await apiFetch<Me>('/me');
  } catch {
    return null;
  }
}

export async function createTicket(params: {
  title: string;
  description?: string;
  requesterId: string;
  priority: number;
}): Promise<{ id: string; number: string } | null> {
  type CreateResp = paths['/tickets']['post']['responses']['201']['content']['application/json'];
  try {
    const data = await apiFetch<CreateResp>('/tickets', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        title: params.title,
        description: params.description || '',
        requester_id: params.requesterId,
        priority: params.priority,
      }),
    });
    return { id: String((data as any).id), number: String((data as any).number) };
  } catch {
    return null;
  }
}

export async function addComment(ticketId: string, _authorId: string, body: string): Promise<boolean> {
  // author is derived server-side from the authenticated user
  try {
    await apiFetch<paths['/tickets/{id}/comments']['post']['responses']['201']['content']['application/json']>(
      `/tickets/${ticketId}/comments`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ body_md: body, is_internal: false }),
      },
    );
    return true;
  } catch {
    return false;
  }
}

export async function logout(): Promise<void> {
  await apiFetch('/logout', { method: 'POST' });
}
