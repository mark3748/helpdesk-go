import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { vi, afterEach } from 'vitest';
import UserRoleManagement from './UserRoleManagement';

function renderWithClient(ui: React.ReactElement) {
  const client = new QueryClient();
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

afterEach(() => {
  vi.restoreAllMocks();
});

test('loads roles for a user', async () => {
  vi.spyOn(global, 'fetch').mockResolvedValue({
    ok: true,
    status: 200,
    json: async () => ['admin'],
  } as any);

  renderWithClient(<UserRoleManagement />);
  fireEvent.change(screen.getByPlaceholderText('User ID'), { target: { value: '1' } });
  fireEvent.click(screen.getByText('Load Roles'));
  await waitFor(() => {
    expect(screen.getByText('admin')).toBeTruthy();
  });
});
