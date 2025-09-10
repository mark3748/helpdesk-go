import { Navigate, Outlet } from 'react-router-dom';
import { useMe } from './auth-utils';

export { useMe };

export function RequireRole({ role }: { role: string }) {
  const { data, isLoading } = useMe();
  if (isLoading) return null;
  const roles = data?.roles || [];
  const isSuper = roles.includes('admin');
  if (!data || !(isSuper || roles.includes(role))) {
    return <Navigate to="/" replace />;
  }
  return <Outlet />;
}
