import { useAuth } from 'react-oidc-context';
import { useEffect, useState } from 'react';
import { Link, Navigate, Route, Routes } from 'react-router-dom';
import TicketList from './components/TicketList';
import TicketForm from './components/TicketForm';
import TicketDetail from './components/TicketDetail';
import KnowledgeBase from './components/KnowledgeBase';
import ServiceCatalog from './components/ServiceCatalog';

export default function App() {
  const auth = useAuth();
  const localMode = !import.meta.env.VITE_OIDC_AUTHORITY;
  const [localReady, setLocalReady] = useState(!localMode);
  const [localAuthed, setLocalAuthed] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (!localMode) return;
    (async () => {
      try {
        const res = await fetch('/api/me', { credentials: 'include' });
        setLocalAuthed(res.ok);
      } finally {
        setLocalReady(true);
      }
    })();
  }, [localMode]);

  if (!localMode) {
    if (auth.isLoading) return <p>Loading...</p>;
    if (!auth.isAuthenticated)
      return <button onClick={() => auth.signinRedirect()}>Login</button>;
  } else {
    if (!localReady) return <p>Loading...</p>;
    if (!localAuthed)
      return (
        <div className="mx-auto mt-32 max-w-sm rounded border p-4">
          <h2 className="mb-4 text-xl font-semibold">Sign in</h2>
          {err && <p className="mb-2 text-red-600">{err}</p>}
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              setErr(null);
              const fd = new FormData(e.currentTarget as HTMLFormElement);
              const body = {
                username: String(fd.get('username') || ''),
                password: String(fd.get('password') || ''),
              };
              const res = await fetch('/api/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'include',
                body: JSON.stringify(body),
              });
              if (!res.ok) {
                setErr('Invalid credentials');
              } else {
                setLocalAuthed(true);
              }
            }}
          >
            <div className="mb-2 flex flex-col">
              <label className="mb-1">Username</label>
              <input className="rounded border p-2" name="username" defaultValue="admin" />
            </div>
            <div className="mb-4 flex flex-col">
              <label className="mb-1">Password</label>
              <input className="rounded border p-2" name="password" type="password" defaultValue="admin" />
            </div>
            <button className="rounded bg-blue-600 px-4 py-2 text-white" type="submit">Login</button>
          </form>
        </div>
      );
  }
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
        {!localMode ? (
          <button className="sm:ml-auto hover:underline" onClick={() => auth.signoutRedirect()}>Logout</button>
        ) : (
          <button
            className="sm:ml-auto hover:underline"
            onClick={async () => { await fetch('/api/logout', { method: 'POST', credentials: 'include' }); setLocalAuthed(false); }}
          >
            Logout
          </button>
        )}
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
