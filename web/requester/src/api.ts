import type { paths, components } from './types/openapi';

export type Ticket = components['schemas']['Ticket'];
export type Comment = components['schemas']['Comment'];
export type Attachment = components['schemas']['Attachment'];

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

export async function listTickets(token: string): Promise<Ticket[]> {
  type Resp = paths['/tickets']['get']['responses']['200']['content']['application/json'];
  return apiFetch<Resp>('/tickets', {}, token);
}

export async function getTicket(id: string, token: string): Promise<Ticket> {
  type Resp = paths['/tickets/{id}']['get']['responses']['200']['content']['application/json'];
  return apiFetch<Resp>(`/tickets/${id}`, {}, token);
}

export async function createTicket(data: Partial<Ticket>, token: string): Promise<Ticket> {
  type Req = paths['/tickets']['post']['requestBody']['content']['application/json'];
  type Resp = paths['/tickets']['post']['responses']['201']['content']['application/json'];
  return apiFetch<Resp>('/tickets', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data as Req),
  }, token);
}

export async function listComments(id: string, token: string): Promise<Comment[]> {
  type Resp = paths['/tickets/{id}/comments']['get']['responses']['200']['content']['application/json'];
  return apiFetch<Resp>(`/tickets/${id}/comments`, {}, token);
}

export async function addComment(id: string, content: string, token: string): Promise<{ id: string }> {
  type Req = paths['/tickets/{id}/comments']['post']['requestBody']['content']['application/json'];
  type Resp = paths['/tickets/{id}/comments']['post']['responses']['201']['content']['application/json'];
  const body: Req = { body_md: content, is_internal: false } as Req;
  return apiFetch<Resp>(`/tickets/${id}/comments`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  }, token);
}

export async function listAttachments(id: string, token: string): Promise<Attachment[]> {
  type Resp = paths['/tickets/{id}/attachments']['get']['responses']['200']['content']['application/json'];
  return apiFetch<Resp>(`/tickets/${id}/attachments`, {}, token);
}

export async function deleteAttachment(id: string, attID: string, token: string): Promise<void> {
  await apiFetch(`/tickets/${id}/attachments/${attID}`, { method: 'DELETE' }, token);
}

export async function downloadAttachment(id: string, attID: string, token: string): Promise<void> {
  const res = await fetch(`${API_BASE}/tickets/${id}/attachments/${attID}`, {
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
  id: string,
  file: File,
  token: string,
  opts: UploadOptions = {},
): Promise<void> {
  return new Promise((resolve, reject) => {
    const form = new FormData();
    form.append('file', file);

    const xhr = new XMLHttpRequest();
    xhr.open('POST', `${API_BASE}/tickets/${id}/attachments`);
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
