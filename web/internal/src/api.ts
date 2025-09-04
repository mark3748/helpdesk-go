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
  const listeners = new Map<string, (e: MessageEvent<string>) => void>();
  let es: EventSource | null = null;
  let timer: number | null = null;
  let backoff = 1000;
  let lastEventId: string | undefined;

  const attachListeners = () => {
    listeners.forEach((listener, type) => es?.addEventListener(type, listener));
  };

  const connect = () => {
    // EventSource does not support custom headers in browsers; pass last id via query
    const url = lastEventId ? `/api/events?last_event_id=${encodeURIComponent(lastEventId)}` : '/api/events';
    es = new EventSource(url, { withCredentials: true });
    es.onopen = () => {
      onStatus?.(true);
      backoff = 1000;
    };
    es.onerror = () => {
      onStatus?.(false);
      if (timer) window.clearTimeout(timer);
      es?.close();
      timer = window.setTimeout(connect, backoff);
      backoff = Math.min(backoff * 2, 30000);
    };
    attachListeners();
  };

  connect();

  const addListener = (type: string) => {
    if (listeners.has(type)) return;
    const listener = (e: MessageEvent<string>) => {
      lastEventId = e.lastEventId || lastEventId;
      try {
        const parsed = JSON.parse(e.data) as AppEvent;
        const cbs = handlers.get(type);
        cbs?.forEach((cb) => cb(parsed));
      } catch {
        // ignore malformed events
      }
    };
    listeners.set(type, listener);
    es?.addEventListener(type, listener);
  };

  const removeListener = (type: string) => {
    const listener = listeners.get(type);
    if (!listener) return;
    es?.removeEventListener(type, listener);
    listeners.delete(type);
  };

  const on = (type: string, handler: (ev: AppEvent) => void) => {
    let set = handlers.get(type);
    if (!set) {
      set = new Set();
      handlers.set(type, set);
      addListener(type);
    }
    set.add(handler);
    return () => {
      const set = handlers.get(type);
      if (!set) return;
      set.delete(handler);
      if (set.size === 0) {
        handlers.delete(type);
        removeListener(type);
      }
    };
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
export type Requester = components['schemas']['Requester'];

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

export function useRequester(id: string, opts: { refetchInterval?: number | false } = {}) {
  type Resp = Requester;
  return useQuery({
    queryKey: ['requester', id],
    queryFn: () => apiFetch<Resp>(`/requesters/${id}`),
    enabled: Boolean(id),
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
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings'] }),
  });
}

export function useSaveOIDCSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Record<string, unknown>) =>
      apiFetch('/settings/oidc', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings'] }),
  });
}

export function useSendTestEmail() {
  return useMutation({
    mutationFn: () =>
      apiFetch('/settings/mail/send-test', {
        method: 'POST',
      }),
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
