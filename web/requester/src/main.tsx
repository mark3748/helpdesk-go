import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { AuthProvider } from 'react-oidc-context';
import { BrowserRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import './index.css';
import App from './App';

const authority = import.meta.env.VITE_OIDC_AUTHORITY as string | undefined;
const clientId = import.meta.env.VITE_OIDC_CLIENT_ID as string | undefined;
const localMode = !authority;
const oidcConfig = {
  authority,
  client_id: clientId,
  redirect_uri: window.location.origin,
  onSigninCallback: () => {
    window.history.replaceState({}, document.title, window.location.pathname);
  },
};

const queryClient = new QueryClient();

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      {localMode ? (
        <BrowserRouter>
          <App />
        </BrowserRouter>
      ) : (
        <AuthProvider {...oidcConfig}>
          <BrowserRouter>
            <App />
          </BrowserRouter>
        </AuthProvider>
      )}
    </QueryClientProvider>
  </StrictMode>,
);
