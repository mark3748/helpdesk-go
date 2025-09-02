import type { Ticket, Comment, Attachment } from '../types/api';

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

export type { Ticket, Comment, Attachment };

export async function createTicket(data: {
  title: string;
  description?: string;
  requester_id: string;
  priority: number;
}): Promise<Ticket> {
  return apiFetch<Ticket>('/tickets', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      title: data.title,
      description: data.description || '',
      requester_id: data.requester_id,
      priority: data.priority,
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
        resolve();
      } else {
        reject(new Error(`failed to upload attachment: ${xhr.status}`));
      }
    };

    xhr.onerror = () => reject(new Error('failed to upload attachment'));

    xhr.send(form);
  });
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
