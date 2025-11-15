import { Navigate } from 'react-router-dom';
import { useAuth } from '@/contexts/AuthContext';
import { LoadingSpinner } from '@/components/ui/loading-spinner';

export function Home() {
  const { isAuthenticated, isLoading } = useAuth();

  // Show loading spinner while checking authentication
  if (isLoading) {
    return <LoadingSpinner fullScreen text="Loading..." />;
  }

  // Redirect authenticated users to dashboard, unauthenticated to login
  if (isAuthenticated) {
    return <Navigate to="/dashboard" replace />;
  }

  // Redirect unauthenticated users to login page
  return <Navigate to="/login" replace />;
}
