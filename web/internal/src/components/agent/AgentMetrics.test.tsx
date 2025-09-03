import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { vi, afterEach } from 'vitest';
import AgentMetrics from './AgentMetrics';

function renderWithClient(ui: React.ReactElement) {
  const client = new QueryClient();
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

afterEach(() => {
  vi.restoreAllMocks();
});

test('displays agent metrics', async () => {
  vi.spyOn(global, 'fetch').mockResolvedValue({
    ok: true,
    status: 200,
    json: async () => ({ tickets_closed: 3 }),
  } as any);

  renderWithClient(<AgentMetrics />);
  await waitFor(() => {
    expect(screen.getByText(/tickets_closed/)).toBeTruthy();
  });
});
