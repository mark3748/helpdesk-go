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
});

test('shows settings data', async () => {
  renderWithClient(<AdminSettings />);
  await waitFor(() => {
    expect(screen.getByText(/System Version/)).toBeTruthy();
  });
});
