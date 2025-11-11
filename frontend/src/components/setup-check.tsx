import { useEffect, useState, type ReactNode } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { setupService } from '@/lib/api/services/setup';

interface SetupCheckProps {
  children: ReactNode;
}

/**
 * SetupCheck component checks if the system requires initial setup (first user registration).
 * If no users exist, it redirects to the register page.
 */
export function SetupCheck({ children }: SetupCheckProps) {
  const [isChecking, setIsChecking] = useState(true);
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    const checkSetup = async () => {
      try {
        const status = await setupService.getSetupStatus();

        // If setup is required and we're not already on the register page, redirect
        if (status.requires_setup && location.pathname !== '/register') {
          navigate('/register', {
            state: { isFirstUser: true },
            replace: true,
          });
        }
      } catch (error) {
        console.error('Failed to check setup status:', error);
        // On error, allow the app to continue normally
      } finally {
        setIsChecking(false);
      }
    };

    checkSetup();
  }, [navigate, location.pathname]);

  // Show nothing while checking (or a loading spinner if preferred)
  if (isChecking) {
    return null;
  }

  return <>{children}</>;
}
