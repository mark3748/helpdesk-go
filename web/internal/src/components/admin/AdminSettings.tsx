import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../../shared/api';

export default function AdminSettings() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['admin-settings'],
    queryFn: () => apiFetch<Record<string, unknown>>('/settings'),
  });

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Failed to load settings</div>;

  return (
    <div>
      <h2>Admin Settings</h2>
      <pre>{JSON.stringify(data, null, 2)}</pre>
    </div>
  );
}
