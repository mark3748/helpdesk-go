import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { vi, afterEach } from 'vitest';
import AdminSettings from './AdminSettings';

function renderWithClient(ui: React.ReactElement) {
  const client = new QueryClient();
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

afterEach(() => {
  vi.restoreAllMocks();
});

test('shows settings data', async () => {
  vi.spyOn(global, 'fetch').mockResolvedValue({
    ok: true,
    status: 200,
    json: async () => ({ site_name: 'Helpdesk' }),
  } as any);

  renderWithClient(<AdminSettings />);
  await waitFor(() => {
    expect(screen.getByText(/site_name/)).toBeTruthy();
  });
});
