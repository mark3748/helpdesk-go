import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../../shared/api';

export default function ManagerAnalytics() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['manager-analytics'],
    queryFn: () => apiFetch<Record<string, number>>('/metrics/manager'),
  });

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Failed to load analytics</div>;

  return (
    <div>
      <h2>Manager Analytics</h2>
      <pre>{JSON.stringify(data, null, 2)}</pre>
    </div>
  );
}
