import { useQuery } from '@tanstack/react-query';

export type Me = {
  id?: string;
  external_id?: string;
  email?: string;
  display_name?: string;
  roles?: string[];
};

async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`/api${path}`, { credentials: 'include', ...init });
  if (!res.ok) {
    const txt = await res.text().catch(() => '');
    throw new Error(`${res.status} ${txt}`);
  }
  if (res.status === 204) return undefined as unknown as T;
  return (await res.json()) as T;
}

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

export { apiFetch };
