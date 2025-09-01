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
    <div className="min-h-screen flex flex-col">
      <nav className="flex flex-col items-start gap-2 bg-blue-600 p-4 text-white sm:flex-row sm:items-center sm:gap-4">
        <Link className="hover:underline" to="/tickets">
          Tickets
        </Link>
        <Link className="hover:underline" to="/tickets/new">
          New Ticket
        </Link>
        <Link className="hover:underline" to="/kb">
          Knowledge Base
        </Link>
        <Link className="hover:underline" to="/catalog">
          Service Catalog
        </Link>
        <button
          className="sm:ml-auto hover:underline"
          onClick={() => auth.signoutRedirect()}
        >
          Logout
        </button>
      </nav>
      <main className="flex-1 p-4">
        <Routes>
          <Route path="/" element={<Navigate to="/tickets" />} />
          <Route path="/tickets" element={<TicketList />} />
          <Route path="/tickets/new" element={<TicketForm />} />
          <Route path="/tickets/:id" element={<TicketDetail />} />
          <Route path="/kb" element={<KnowledgeBase />} />
          <Route path="/catalog/*" element={<ServiceCatalog />} />
        </Routes>
      </main>
    </div>
  );
}
