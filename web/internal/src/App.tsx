import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SidebarLayout } from './shared/SidebarLayout';
import { RequireRole, useMe } from './shared/auth';
import QueueList from './components/agent/QueueList';
import TicketDetail from './components/agent/TicketDetail';
import AgentMetrics from './components/agent/AgentMetrics';
import MailSettings from './components/admin/MailSettings';
import OIDCSettings from './components/admin/OIDCSettings';
import StorageSettings from './components/admin/StorageSettings';
import AdminSettings from './components/admin/AdminSettings';
import AdminUsers from './components/admin/AdminUsers';
import QueueManager from './components/manager/QueueManager';
import ManagerAnalytics from './components/manager/ManagerAnalytics';
import Login from './components/Login';
import ComingSoon from './shared/ComingSoon';
import UserSettings from './components/UserSettings';

const queryClient = new QueryClient();

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          {/* Public login route */}
          <Route path="/login" element={<Login />} />

          <Route element={<SidebarLayout />}>
            {/* Landing: show login if unauthenticated, otherwise neutral page */}
            <Route index element={<Landing />} />
            <Route element={<RequireRole role="agent" />}>
              <Route path="/tickets" element={<QueueList />} />
              <Route path="/tickets/:id" element={<TicketDetail />} />
              <Route path="/metrics" element={<AgentMetrics />} />
            </Route>
            <Route element={<RequireRole role="admin" />}>
              <Route path="/settings" element={<AdminSettings />} />
              <Route path="/settings/mail" element={<MailSettings />} />
              <Route path="/settings/oidc" element={<OIDCSettings />} />
              <Route path="/settings/storage" element={<StorageSettings />} />
              <Route path="/settings/users" element={<AdminUsers />} />
              <Route path="/settings/*" element={<ComingSoon title="Settings area" detail="Additional admin settings will appear here." />} />
            </Route>
            {/* User account settings (any authenticated user) */}
            <Route path="/me/settings" element={<UserSettings />} />
            <Route element={<RequireRole role="manager" />}>
              <Route path="/manager" element={<QueueManager />} />
              <Route path="/manager/analytics" element={<ManagerAnalytics />} />
            </Route>
            {/* 404 catch-all inside the layout: redirect to a sensible default */}
            <Route path="*" element={<NotFoundRedirect />} />
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
  return (
    <div style={{ padding: 24 }}>
      <p>Select a page from the sidebar to get started.</p>
    </div>
  );
}

function NotFoundRedirect() {
  // Redirect to tickets if agent/admin; otherwise to landing
  const { data } = useMe();
  const roles = data?.roles || [];
  if (roles.includes('agent') || roles.includes('admin')) {
    return <Navigate to="/tickets" replace />;
  }
  if (roles.includes('manager')) {
    return <Navigate to="/manager" replace />;
  }
  return <Navigate to="/" replace />;
}
