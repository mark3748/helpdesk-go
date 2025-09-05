import type { components } from '../types/openapi';

const API_BASE = '/api';

export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, { credentials: 'include', ...init });
  if (!res.ok) {
    const txt = await res.text().catch(() => '');
    throw new Error(`${res.status} ${txt}`);
  }
  if (res.status === 204) return undefined as unknown as T;
  return (await res.json()) as T;
}

export type Ticket = components['schemas']['Ticket'];
export type Comment = components['schemas']['Comment'];
export type Attachment = components['schemas']['Attachment'];
export type Requester = components['schemas']['Requester'];

export async function fetchRequester(id: string): Promise<Requester> {
  return apiFetch<Requester>(`/requesters/${id}`);
}

export async function createRequester(data: {
  email: string;
  display_name: string;
}): Promise<Requester> {
  return apiFetch<Requester>('/requesters', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
}

export async function updateRequester(
  id: string,
  data: { email?: string; display_name?: string },
): Promise<Requester> {
  return apiFetch<Requester>(`/requesters/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
}


export async function createTicket(data: {
  title: string;
  description?: string;
  requester_id: string;
  priority: number;
  assignee_id?: string;
}): Promise<Ticket> {
  return apiFetch<Ticket>('/tickets', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      // Help the API deduplicate accidental double-submits via idempotency keys.
      'Idempotency-Key': (typeof globalThis.crypto?.randomUUID === 'function')
        ? globalThis.crypto.randomUUID()
        : String(Date.now()) + '-' + Math.random().toString(36).slice(2),
    },
    body: JSON.stringify({
      title: data.title,
      description: data.description || '',
      requester_id: data.requester_id,
      priority: data.priority,
      assignee_id: data.assignee_id,
    }),
  });
}

export async function updateTicketStatus(
  id: string,
  status: string,
): Promise<void> {
  await apiFetch(`/tickets/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status }),
  });
}

export async function updateTicket(
  id: string,
  data: { assignee_id?: string | null; priority?: number },
): Promise<void> {
  await apiFetch(`/tickets/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
}

export async function fetchComments(ticketId: string): Promise<Comment[]> {
  try {
    return await apiFetch<Comment[]>(`/tickets/${ticketId}/comments`);
  } catch {
    return [];
  }
}

export async function addComment(
  ticketId: string,
  body: string,
  isInternal = false,
): Promise<void> {
  await apiFetch(`/tickets/${ticketId}/comments`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ body_md: body, is_internal: isInternal }),
  });
}

export async function fetchAttachments(ticketId: string): Promise<Attachment[]> {
  try {
    return await apiFetch<Attachment[]>(`/tickets/${ticketId}/attachments`);
  } catch {
    return [];
  }
}

export interface UploadCallbacks {
  onProgress?: (evt: { percent: number }) => void;
}

export function uploadAttachment(
  ticketId: string,
  file: File,
  cb: UploadCallbacks = {},
): Promise<void> {
  return (async () => {
    const presign = await apiFetch<{ upload_url: string; headers: Record<string, string>; attachment_id: string }>(
      `/tickets/${ticketId}/attachments/presign`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ filename: file.name, bytes: file.size, mime: file.type }),
      },
    );

    await new Promise<void>((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      xhr.open('PUT', presign.upload_url);
      Object.entries(presign.headers || {}).forEach(([k, v]) => xhr.setRequestHeader(k, v));
      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable) cb.onProgress?.({ percent: (e.loaded / e.total) * 100 });
      };
      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) resolve();
        else reject(new Error(`failed to upload attachment: ${xhr.status}`));
      };
      xhr.onerror = () => reject(new Error('failed to upload attachment'));
      xhr.send(file);
    });

    for (let i = 0; i < 5; i++) {
      try {
        await apiFetch(`/tickets/${ticketId}/attachments`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            attachment_id: presign.attachment_id,
            filename: file.name,
            bytes: file.size,
            mime: file.type,
          }),
        });
        return;
      } catch {
        await new Promise((r) => setTimeout(r, 1000));
      }
    }
    throw new Error('failed to finalize attachment');
  })();
}

export async function deleteAttachment(
  ticketId: string,
  attId: string,
): Promise<void> {
  await apiFetch(`/tickets/${ticketId}/attachments/${attId}`, {
    method: 'DELETE',
  });
}

export async function downloadAttachment(
  ticketId: string,
  attId: string,
): Promise<void> {
  const res = await fetch(`${API_BASE}/tickets/${ticketId}/attachments/${attId}`, {
    credentials: 'include',
    redirect: 'follow',
  });
  if (!res.ok) throw new Error(await res.text());
  const blob = await res.blob();
  const cd = res.headers.get('Content-Disposition') || '';
  const m = /filename\*=UTF-8''([^;]+)|filename=\"?([^\";]+)\"?/i.exec(cd);
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
