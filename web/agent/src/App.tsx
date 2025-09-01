import { useState } from 'react';
import Login from './components/Login';
import TicketWorkspace from './components/TicketWorkspace';

function App() {
  const [loggedIn, setLoggedIn] = useState(false);

  return loggedIn ? (
    <TicketWorkspace />
  ) : (
    <Login onLoggedIn={() => setLoggedIn(true)} />
  );
}

export default App;
