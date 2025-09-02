import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SidebarLayout } from '../../shared/SidebarLayout';
import { RequireRole, useMe } from '../../shared/auth';
import QueueList from './components/agent/QueueList';
import TicketDetail from './components/agent/TicketDetail';
import MailSettings from './components/admin/MailSettings';
import OIDCSettings from './components/admin/OIDCSettings';
import StorageSettings from './components/admin/StorageSettings';
import QueueManager from './components/manager/QueueManager';
import Login from './components/Login';

const queryClient = new QueryClient();

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          {/* Public login route */}
          <Route path="/login" element={<Login />} />

          <Route element={<SidebarLayout />}>
            {/* Landing: show login if unauthenticated, else go to tickets */}
            <Route index element={<Landing />} />
            <Route element={<RequireRole role="agent" />}> 
              <Route path="/tickets" element={<QueueList />} />
              <Route path="/tickets/:id" element={<TicketDetail />} />
            </Route>
            <Route element={<RequireRole role="admin" />}>
              <Route path="/settings" element={<MailSettings />} />
              <Route path="/settings/oidc" element={<OIDCSettings />} />
              <Route path="/settings/storage" element={<StorageSettings />} />
            </Route>
            <Route element={<RequireRole role="manager" />}>
              <Route path="/manager" element={<QueueManager />} />
            </Route>
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

function Landing() {
  const { data, isLoading } = useMe();
  if (isLoading) return null;
  if (!data) return <Login />;
  return <Navigate to="/tickets" replace />;
}
