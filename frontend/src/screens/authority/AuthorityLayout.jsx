import { Outlet, Navigate, useLocation } from 'react-router-dom';
import { getAuthorityToken } from '../../services/authorityApi';

// Redirect /authority to dashboard if logged in, else to login.
export function AuthorityIndexRedirect() {
  return getAuthorityToken() ? <Navigate to="/authority/dashboard" replace /> : <Navigate to="/authority/login" replace />;
}

// Wrapper for /authority: redirect to login if no token (except when already on login).
export default function AuthorityLayout() {
  const token = getAuthorityToken();
  const location = useLocation();
  const onLogin = location.pathname === '/authority/login' || location.pathname === '/authority';

  if (!token && !onLogin) {
    return <Navigate to="/authority/login" replace />;
  }
  return <Outlet />;
}
