import { useEffect, type ReactNode } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { useSetupSWR } from '@/hooks/api/useSetupSWR';

interface SetupCheckProps {
  children: ReactNode;
}

/**
 * SetupCheck component checks if the system requires initial setup (first user registration).
 * If no users exist, it redirects to the register page.
 *
 * Uses SWR caching to avoid duplicate API calls across route changes.
 */
export function SetupCheck({ children }: SetupCheckProps) {
  const navigate = useNavigate();
  const location = useLocation();

  // Use cached setup status - SWR will deduplicate requests automatically
  const { requiresSetup, isLoading } = useSetupSWR();

  useEffect(() => {
    // If setup is required and we're not already on the register page, redirect
    if (!isLoading && requiresSetup && location.pathname !== '/register') {
      navigate('/register', {
        state: { isFirstUser: true },
        replace: true,
      });
    }
  }, [requiresSetup, isLoading, navigate, location.pathname]);

  // Show nothing while checking (or a loading spinner if preferred)
  if (isLoading) {
    return null;
  }

  return <>{children}</>;
}
