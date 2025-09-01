import { useState } from 'react';
import { login } from '../api';

interface Props {
  onLoggedIn(): void;
}

export default function Login({ onLoggedIn }: Props) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const ok = await login(username, password);
    if (ok) {
      onLoggedIn();
    } else {
      setError('Login failed');
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <h2>Agent Login</h2>
      {error && <p>{error}</p>}
      <div>
        <input
          placeholder="Username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
        />
      </div>
      <div>
        <input
          type="password"
          placeholder="Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
      </div>
      <button type="submit">Login</button>
    </form>
  );
}
