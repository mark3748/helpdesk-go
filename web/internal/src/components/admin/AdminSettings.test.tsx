import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { vi, afterEach, test, expect } from 'vitest';
import AdminSettings from './AdminSettings';

import { MemoryRouter } from 'react-router-dom';

function renderWithClient(ui: React.ReactElement) {
  const client = new QueryClient();
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>
  );
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

test('shows settings data and Discord configuration', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    json: async () => ({
      version: 'test',
      uptime: 'running',
      database_status: 'connected',
      storage_status: 'configured',
      mail_status: 'configured',
      oidc_status: 'configured',
      discord_status: 'configured',
    }),
  }));

  renderWithClient(<AdminSettings />);
  await waitFor(() => {
    expect(screen.getByText(/System Version/)).toBeTruthy();
    expect(screen.getAllByText('Discord Bot').length).toBeGreaterThan(0);
  });
});
