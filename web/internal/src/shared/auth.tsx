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

export function RequireAnyRole({ roles: allowedRoles }: { roles: string[] }) {
  const { data, isLoading } = useMe();
  if (isLoading) return null;
  const userRoles = data?.roles || [];
  const isSuper = userRoles.includes('admin');
  const hasAnyRole = allowedRoles.some(role => userRoles.includes(role));
  if (!data || !(isSuper || hasAnyRole)) {
    return <Navigate to="/" replace />;
  }
  return <Outlet />;
}
