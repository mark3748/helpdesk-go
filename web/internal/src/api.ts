import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { Ticket } from './types/api';
import { apiFetch } from './shared/api';

export interface AppEvent {
  type: string;
  data: Record<string, unknown>;
}

/**
 * Subscribe to server sent events and automatically reconnect on failure.
 * Returns a cleanup function to close the connection.
 */
export function subscribeEvents(
  onEvent: (ev: AppEvent) => void,
  onStatus?: (connected: boolean) => void,
): () => void {
  let es: EventSource | null = null;
  let timer: number | null = null;

  const connect = () => {
    es = new EventSource('/api/events', { withCredentials: true });
    es.onopen = () => onStatus?.(true);
    const handler = (e: MessageEvent<string>) => {
      try {
        const parsed = JSON.parse(e.data) as AppEvent;
        onEvent(parsed);
      } catch {
        // ignore malformed events
      }
    };
    ['ticket_created', 'ticket_updated', 'queue_changed'].forEach((evt) => {
      es?.addEventListener(evt, handler);
    });
    es.onerror = () => {
      onStatus?.(false);
      if (timer) window.clearTimeout(timer);
      es?.close();
      timer = window.setTimeout(connect, 3000);
    };
  };

  connect();

  return () => {
    if (timer) window.clearTimeout(timer);
    es?.close();
  };
}

export type { Ticket };

export function useTickets(opts: { refetchInterval?: number | false; params?: Record<string, string> } = {}) {
  type Resp = Ticket[];
  return useQuery({
    queryKey: ['tickets', opts.params || null],
    queryFn: () => {
      const qs = opts.params ? '?' + new URLSearchParams(opts.params).toString() : '';
      return apiFetch<Resp>(`/tickets${qs}`);
    },
    ...opts,
  });
}

export function useTicket(id: string, opts: { refetchInterval?: number | false } = {}) {
  type Resp = Ticket;
  return useQuery({
    queryKey: ['ticket', id],
    queryFn: () => apiFetch<Resp>(`/tickets/${id}`),
    enabled: Boolean(id),
    ...opts,
  });
}

export function useSettings() {
  return useQuery({
    queryKey: ['settings'],
    queryFn: () => apiFetch<Record<string, unknown>>('/settings'),
  });
}

export function useSaveMailSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Record<string, unknown>) =>
      apiFetch('/settings/mail', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings'] }),
  });
}

export function useMailTest() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiFetch('/settings/mail/test', {
        method: 'POST',
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings'] }),
  });
}
