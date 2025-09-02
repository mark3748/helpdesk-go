import { useEffect, useState } from 'react';
import Login from './components/Login';
import TicketWorkspace from './components/TicketWorkspace';
import { getMe } from './api';

function App() {
  const [loggedIn, setLoggedIn] = useState(false);
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    getMe()
      .then((m) => setLoggedIn(!!m))
      .catch(() => setLoggedIn(false))
      .finally(() => setChecking(false));
  }, []);

  if (checking) return null;

  return loggedIn ? (
    <TicketWorkspace />
  ) : (
    <Login onLoggedIn={() => setLoggedIn(true)} />
  );
}

export default App;
