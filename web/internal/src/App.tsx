import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SidebarLayout } from '../../shared/SidebarLayout';
import { RequireRole } from '../../shared/auth';
import TicketList from './components/TicketList';
import MailSettings from './components/admin/MailSettings';
import OIDCSettings from './components/admin/OIDCSettings';
import StorageSettings from './components/admin/StorageSettings';

const queryClient = new QueryClient();

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route element={<SidebarLayout />}>
            <Route element={<RequireRole role="agent" />}>
              <Route path="/tickets" element={<TicketList />} />
            </Route>
            <Route element={<RequireRole role="admin" />}>
              <Route path="/settings" element={<MailSettings />} />
              <Route path="/settings/oidc" element={<OIDCSettings />} />
              <Route path="/settings/storage" element={<StorageSettings />} />
            </Route>
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
