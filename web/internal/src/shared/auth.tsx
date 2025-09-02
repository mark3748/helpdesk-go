import { useQuery } from '@tanstack/react-query';
import { Navigate, Outlet } from 'react-router-dom';
async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`/api${path}`, { credentials: 'include', ...init });
  if (!res.ok) {
    const txt = await res.text().catch(() => '');
    throw new Error(`${res.status} ${txt}`);
  }
  if (res.status === 204) return undefined as unknown as T;
  return (await res.json()) as T;
}
import type { components } from '../types/openapi';

export type Me = components['schemas']['AuthUser'];

export function useMe() {
  return useQuery<Me | null>({
    queryKey: ['me'],
    queryFn: async () => {
      try {
        return await apiFetch<Me>('/me');
      } catch {
        return null;
      }
    },
  });
}

export function RequireRole({ role }: { role: string }) {
  const { data, isLoading } = useMe();
  if (isLoading) return null;
  const roles = data?.roles || [];
  const isSuper = roles.includes('admin');
  if (!data || !(isSuper || roles.includes(role))) {
    return <Navigate to="/" replace />;
  }
  return <Outlet />;
}
