import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { LoadingSpinner } from '@/components/ui/loading-spinner';
import { safeRedirect } from '@/lib/auth/safe-redirect';
import { clearReturnTo, getReturnTo } from '@/lib/auth/return-to';

/**
 * AuthCallbackPage is the landing route for the Google OAuth roundtrip.
 *
 * The auth cookie is already set by the backend by the time this renders, so
 * it renders only a spinner and forwards the user to the path they originally
 * requested (carried across the roundtrip in sessionStorage via returnTo),
 * validated through safeRedirect to block open redirects.
 */
export const AuthCallbackPage: React.FC = () => {
  const navigate = useNavigate();

  // Resolve the target once, during render, before the effect clears returnTo.
  // The effect is not guaranteed to run only once (React StrictMode invokes it
  // twice in development, and a re-render could re-fire it); reading returnTo
  // inside the effect would see it already cleared on the second pass and fall
  // back to /dashboard, defeating the forwarding. Capturing it here keeps the
  // destination stable across every invocation.
  const [target] = useState(() => safeRedirect(getReturnTo()));

  useEffect(() => {
    clearReturnTo();
    navigate(target, { replace: true });
  }, [navigate, target]);

  return <LoadingSpinner fullScreen text="Signing you in..." />;
};
