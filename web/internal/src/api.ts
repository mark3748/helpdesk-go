import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { paths } from './types/openapi';
import { apiFetch } from '../../shared/api';

export type Ticket = paths['/tickets']['get']['responses']['200']['content']['application/json'][number];

export function useTickets() {
  type Resp = paths['/tickets']['get']['responses']['200']['content']['application/json'];
  return useQuery({
    queryKey: ['tickets'],
    queryFn: () => apiFetch<Resp>('/tickets'),
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
