import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { components } from './types/openapi';
import { apiFetch } from './shared/api';

export interface AppEvent {
  type: string;
  data: Record<string, unknown>;
}

/**
 * Subscribe to server sent events and automatically reconnect on failure.
 * Returns a cleanup function to close the connection.
 */
export function subscribeEvents(onStatus?: (connected: boolean) => void) {
  const handlers = new Map<string, Set<(ev: AppEvent) => void>>();
  let es: EventSource | null = null;
  let timer: number | null = null;
  let backoff = 1000;
  let lastEventId: string | undefined;

  const connect = () => {
    const headers = lastEventId ? { 'Last-Event-ID': lastEventId } : undefined;
    es = new EventSource('/api/events', { withCredentials: true, headers });
    es.onopen = () => {
      onStatus?.(true);
      backoff = 1000;
    };
    es.onmessage = (e: MessageEvent<string>) => {
      lastEventId = e.lastEventId || lastEventId;
      try {
        const parsed = JSON.parse(e.data) as AppEvent;
        const cbs = handlers.get(parsed.type);
        cbs?.forEach((cb) => cb(parsed));
      } catch {
        // ignore malformed events
      }
    };
    es.onerror = () => {
      onStatus?.(false);
      if (timer) window.clearTimeout(timer);
      es?.close();
      timer = window.setTimeout(connect, backoff);
      backoff = Math.min(backoff * 2, 30000);
    };
  };

  connect();

  const on = (type: string, handler: (ev: AppEvent) => void) => {
    const set = handlers.get(type);
    if (set) set.add(handler);
    else handlers.set(type, new Set([handler]));
    return () => handlers.get(type)?.delete(handler);
  };

  return {
    on,
    close: () => {
      if (timer) window.clearTimeout(timer);
      es?.close();
    },
  };
}

export type Ticket = components['schemas']['Ticket'];

export function useTickets(
  opts: {
    cursor?: string;
    query?: Record<string, string>;
    refetchInterval?: number | false;
  } = {},
) {
  type Resp = { items: Ticket[]; next_cursor?: string };
  const { cursor, query, ...rest } = opts;
  return useQuery({
    queryKey: ['tickets', cursor, query],
    queryFn: () => {
      const params = new URLSearchParams(query || {});
      if (cursor) params.set('cursor', cursor);
      const qs = params.toString();
      return apiFetch<Resp>(`/tickets${qs ? `?${qs}` : ''}`);
    },
    ...rest,
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
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings'] }),
  });
}

export function useTestConnection() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiFetch('/test-connection', {
        method: 'POST',
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings'] }),
  });
}
