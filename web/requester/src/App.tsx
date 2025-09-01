import { useAuth } from 'react-oidc-context';
import { Link, Navigate, Route, Routes } from 'react-router-dom';
import TicketList from './components/TicketList';
import TicketForm from './components/TicketForm';
import TicketDetail from './components/TicketDetail';
import KnowledgeBase from './components/KnowledgeBase';
import ServiceCatalog from './components/ServiceCatalog';

export default function App() {
  const auth = useAuth();
  if (auth.isLoading) return <p>Loading...</p>;
  if (!auth.isAuthenticated)
    return <button onClick={() => auth.signinRedirect()}>Login</button>;
  return (
    <div>
      <nav>
        <Link to="/tickets">Tickets</Link> |{' '}
        <Link to="/tickets/new">New Ticket</Link> |{' '}
        <Link to="/kb">Knowledge Base</Link> |{' '}
        <Link to="/catalog">Service Catalog</Link> |{' '}
        <button onClick={() => auth.signoutRedirect()}>Logout</button>
      </nav>
      <Routes>
        <Route path="/" element={<Navigate to="/tickets" />} />
        <Route path="/tickets" element={<TicketList />} />
        <Route path="/tickets/new" element={<TicketForm />} />
        <Route path="/tickets/:id" element={<TicketDetail />} />
        <Route path="/kb" element={<KnowledgeBase />} />
        <Route path="/catalog/*" element={<ServiceCatalog />} />
      </Routes>
    </div>
  );
}
