import { useQuery } from '@tanstack/react-query';
import { Navigate, Outlet } from 'react-router-dom';
import { apiFetch } from './api';
import type { components } from '../agent/src/types/openapi';

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
  if (!data || !data.roles?.includes(role)) {
    return <Navigate to="/" replace />;
  }
  return <Outlet />;
}
