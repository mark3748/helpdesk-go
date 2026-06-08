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
  let ws: WebSocket | null = null;
  let timer: number | null = null;
  let backoff = 1000;
  let shouldReconnect = true;

  const connect = () => {
    // Delay connection to handle potential immediate cleanup (React Strict Mode)
    timer = window.setTimeout(() => {
      let url = '';
      if (import.meta.env && import.meta.env.VITE_WS_URL) {
        url = import.meta.env.VITE_WS_URL;
      } else {
        const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = window.location.host; // includes port
        url = `${proto}//${host}/api/events`;
      }

      if (ws) {
        ws.close();
      }
      ws = new WebSocket(url);

      ws.onopen = () => {
        onStatus?.(true);
        backoff = 1000;
      };

      ws.onmessage = (event) => {
        try {
          const parsed = JSON.parse(event.data) as AppEvent;
          const cbs = handlers.get(parsed.type);
          cbs?.forEach((cb) => cb(parsed));
        } catch (e) {
          console.error('Failed to parse event', e);
        }
      };

      ws.onclose = () => {
        onStatus?.(false);
        if (shouldReconnect) {
          timer = window.setTimeout(connect, backoff);
          backoff = Math.min(backoff * 2, 30000);
        }
      };

      ws.onerror = (err) => {
        if (!shouldReconnect) return;
        console.error('WebSocket error:', err);
      };
    }, 50);
  };

  connect();

  const on = (type: string, handler: (ev: AppEvent) => void) => {
    let set = handlers.get(type);
    if (!set) {
      set = new Set();
      handlers.set(type, set);
    }
    set.add(handler);
    return () => {
      const set = handlers.get(type);
      if (!set) return;
      set.delete(handler);
      if (set.size === 0) {
        handlers.delete(type);
      }
    };
  };

  return {
    on,
    close: () => {
      shouldReconnect = false;
      if (timer) window.clearTimeout(timer);
      if (ws) {
        ws.onclose = null;
        ws.onerror = null;
        ws.onopen = null;
        ws.onmessage = null;
        ws.close();
      }
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

export function useSystemInfo() {
  return useQuery({
    queryKey: ['system-info'],
    queryFn: () =>
      apiFetch<{
        version: string;
        uptime: string;
        database_status: 'connected' | 'disconnected';
        storage_status: 'configured' | 'not_configured';
        mail_status: 'configured' | 'not_configured';
        oidc_status: 'configured' | 'not_configured';
      }>('/system/info'),
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
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (to?: string) =>
      apiFetch(`/settings/mail/send-test${to ? `?to=${encodeURIComponent(to)}` : ''}`, {
        method: 'POST',
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
