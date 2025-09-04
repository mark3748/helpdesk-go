import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { vi, afterEach } from 'vitest';
import ManagerAnalytics from './ManagerAnalytics';

function renderWithClient(ui: React.ReactElement) {
  const client = new QueryClient();
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

afterEach(() => {
  vi.restoreAllMocks();
});

test('renders manager analytics', async () => {
  vi.spyOn(global, 'fetch').mockResolvedValue({
    ok: true,
    status: 200,
    json: async () => ({ queues: 2 }),
  } as any);

  renderWithClient(<ManagerAnalytics />);
  await waitFor(() => {
    expect(screen.getByText(/queues/)).toBeTruthy();
  });
});
