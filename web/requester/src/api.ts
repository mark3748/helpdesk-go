import type { components } from './types/openapi';

export type Ticket = components['schemas']['Ticket'];
export type Comment = components['schemas']['Comment'];
export type Attachment = components['schemas']['Attachment'];

export type { Ticket, Comment, Attachment };

const API_BASE = import.meta.env.VITE_API_BASE || '/api';

async function apiFetch<T = unknown>(path: string, opts: RequestInit = {}, token?: string): Promise<T> {
  const headers: Record<string, string> = {
    ...(opts.headers as Record<string, string>),
  };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const res = await fetch(`${API_BASE}${path}`, { ...opts, headers });
  if (!res.ok) throw new Error(await res.text());
  if (res.status === 204) return null as T;
  return (await res.json()) as T;
}

export async function listTickets(
  token: string,
  cursor?: string,
  query: Record<string, string> = {},
): Promise<{ items: Ticket[]; next_cursor?: string }> {
  const params = new URLSearchParams(query);
  if (cursor) params.set('cursor', cursor);
  const qs = params.toString();
  return apiFetch<{ items: Ticket[]; next_cursor?: string }>(
    `/tickets${qs ? `?${qs}` : ''}`,
    {},
    token,
  );
}

export async function getTicket(id: string, token: string): Promise<Ticket> {
  return apiFetch<Ticket>(`/tickets/${id}`, {}, token);
}

export async function createTicket(data: Partial<Ticket>, token: string): Promise<Ticket> {
  return apiFetch<Ticket>('/tickets', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  }, token);
}

export async function listComments(id: string, token: string): Promise<Comment[]> {
  return apiFetch<Comment[]>(`/tickets/${id}/comments`, {}, token);
}

export async function addComment(id: string | number, content: string, token: string): Promise<{ id: string }> {
  const body = { body_md: content, is_internal: false } as any;
  return apiFetch<{ id: string }>(`/tickets/${String(id)}/comments`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  }, token);
}

export async function listAttachments(id: string, token: string): Promise<Attachment[]> {
  return apiFetch<Attachment[]>(`/tickets/${id}/attachments`, {}, token);
}

export async function deleteAttachment(id: string | number, attID: string | number, token: string): Promise<void> {
  await apiFetch(`/tickets/${String(id)}/attachments/${String(attID)}`, { method: 'DELETE' }, token);
}

export async function downloadAttachment(id: string | number, attID: string | number, token: string): Promise<void> {
  const res = await fetch(`${API_BASE}/tickets/${String(id)}/attachments/${String(attID)}`, {
    headers: { Authorization: `Bearer ${token}` },
    redirect: 'follow',
  });
  if (!res.ok) throw new Error(await res.text());
  const blob = await res.blob();
  // Try to extract filename from Content-Disposition
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

export interface UploadOptions {
  onProgress?: (evt: { percent: number }) => void;
}

export function uploadAttachment(
  id: string | number,
  file: File,
  token: string,
  opts: UploadOptions = {},
): Promise<void> {
  return new Promise((resolve, reject) => {
    const form = new FormData();
    form.append('file', file);

    const xhr = new XMLHttpRequest();
    xhr.open('POST', `${API_BASE}/tickets/${String(id)}/attachments`);
    xhr.setRequestHeader('Authorization', `Bearer ${token}`);

    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) {
        opts.onProgress?.({ percent: (e.loaded / e.total) * 100 });
      }
    };

    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve();
      } else {
        reject(new Error(xhr.responseText || `failed to upload attachment: ${xhr.status}`));
      }
    };

    xhr.onerror = () => reject(new Error('failed to upload attachment'));

    xhr.send(form);
  });
}
