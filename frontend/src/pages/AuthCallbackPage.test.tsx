import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Routes, Route, useLocation } from 'react-router-dom';
import { AuthCallbackPage } from '@/pages/AuthCallbackPage';
import { getReturnTo, setReturnTo } from '@/lib/auth/return-to';

// Renders the pathname so tests can assert where the callback forwarded to.
const LocationProbe = () => {
  const location = useLocation();
  return <span data-testid="pathname">{location.pathname}</span>;
};

const renderCallback = () =>
  render(
    <MemoryRouter initialEntries={['/auth/callback']}>
      <Routes>
        <Route path="/auth/callback" element={<AuthCallbackPage />} />
        <Route path="*" element={<LocationProbe />} />
      </Routes>
    </MemoryRouter>,
  );

describe('AuthCallbackPage', () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it('forwards to the stored returnTo path and clears it', () => {
    setReturnTo('/projects/42?tab=history');

    renderCallback();

    expect(screen.getByTestId('pathname')).toHaveTextContent('/projects/42');
    expect(getReturnTo()).toBeNull();
  });

  it('falls back to /dashboard when nothing is stored', () => {
    renderCallback();

    expect(screen.getByTestId('pathname')).toHaveTextContent('/dashboard');
  });

  it('falls back to /dashboard for an unsafe stored path', () => {
    setReturnTo('//evil.com');

    renderCallback();

    expect(screen.getByTestId('pathname')).toHaveTextContent('/dashboard');
    expect(getReturnTo()).toBeNull();
  });
});
