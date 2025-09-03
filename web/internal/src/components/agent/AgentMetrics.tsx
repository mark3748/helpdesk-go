import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../../shared/api';

export default function AgentMetrics() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['agent-metrics'],
    queryFn: () => apiFetch<Record<string, number>>('/metrics/agent'),
  });

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Failed to load metrics</div>;

  return (
    <div>
      <h2>Agent Metrics</h2>
      <pre>{JSON.stringify(data, null, 2)}</pre>
    </div>
  );
}
