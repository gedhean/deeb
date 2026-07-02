import React, { useEffect, useState } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import { LoadingSpinner } from '@/components/ui/loading-spinner';
import { setReturnTo } from '@/lib/auth/return-to';

interface ProtectedRouteProps {
  children: React.ReactNode;
}

/**
 * Redirects an unauthenticated visitor to /login while remembering where they
 * were headed.
 *
 * The attempted path is captured once, at first render, via useState. This is
 * deliberate: <Navigate> changes the location, and because this component reads
 * useLocation it would otherwise re-render, recompute `from` as "/login", and
 * emit a fresh <Navigate> — a self-sustaining render loop. Freezing `from`
 * keeps the destination (and the RR state object's payload) stable so Navigate
 * settles after a single redirect.
 *
 * setReturnTo runs in an effect, never during render: writing to storage while
 * rendering is a side effect React does not allow. The value carries the path
 * through the cross-origin Google OAuth roundtrip (RR location state only
 * survives the in-SPA email/password flow).
 */
const RedirectToLogin: React.FC = () => {
  const location = useLocation();
  const [from] = useState(() => location.pathname + location.search);

  useEffect(() => {
    setReturnTo(from);
  }, [from]);

  return <Navigate to="/login" state={{ from }} replace />;
};

export const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ children }) => {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    // Show loading spinner while checking authentication
    return <LoadingSpinner fullScreen text="Checking authentication..." />;
  }

  if (!isAuthenticated) {
    return <RedirectToLogin />;
  }

  return <>{children}</>;
};
