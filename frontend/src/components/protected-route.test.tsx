import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { BrowserRouter, MemoryRouter, Routes, Route, useLocation } from 'react-router-dom';
import { ProtectedRoute } from '@/components/protected-route';
import { useAuth } from '@/contexts/AuthContext';
import { getReturnTo } from '@/lib/auth/return-to';

// Mock AuthContext
vi.mock('@/contexts/AuthContext', () => ({
  useAuth: vi.fn(),
}));

// Mock LoadingSpinner
vi.mock('@/components/ui/loading-spinner', () => ({
  LoadingSpinner: ({ text }: { text?: string }) => <div>Loading: {text}</div>,
}));

// Renders the current router state so tests can assert where Navigate landed
// and what location state it carried.
const LocationProbe = () => {
  const location = useLocation();
  return (
    <div>
      <span data-testid="pathname">{location.pathname}</span>
      <span data-testid="from">{(location.state as { from?: string } | null)?.from ?? ''}</span>
    </div>
  );
};

describe('ProtectedRoute', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
  });

  it('should render children when authenticated', () => {
    (useAuth as ReturnType<typeof vi.fn>).mockReturnValue({
      isAuthenticated: true,
      isLoading: false,
    });

    render(
      <BrowserRouter>
        <ProtectedRoute>
          <div>Protected Content</div>
        </ProtectedRoute>
      </BrowserRouter>,
    );

    expect(screen.getByText('Protected Content')).toBeInTheDocument();
  });

  it('should show loading spinner when loading', () => {
    (useAuth as ReturnType<typeof vi.fn>).mockReturnValue({
      isAuthenticated: false,
      isLoading: true,
    });

    render(
      <BrowserRouter>
        <ProtectedRoute>
          <div>Protected Content</div>
        </ProtectedRoute>
      </BrowserRouter>,
    );

    expect(screen.getByText(/Loading:/)).toBeInTheDocument();
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
  });

  it('should redirect to login when not authenticated', () => {
    (useAuth as ReturnType<typeof vi.fn>).mockReturnValue({
      isAuthenticated: false,
      isLoading: false,
    });

    render(
      <MemoryRouter initialEntries={['/protected']}>
        <Routes>
          <Route
            path="/protected"
            element={
              <ProtectedRoute>
                <div>Protected Content</div>
              </ProtectedRoute>
            }
          />
          <Route path="/login" element={<LocationProbe />} />
        </Routes>
      </MemoryRouter>,
    );

    // Redirected to /login; the protected content is never shown.
    expect(screen.getByTestId('pathname')).toHaveTextContent('/login');
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
  });

  it('should capture the attempted path into RR state and returnTo on redirect', () => {
    (useAuth as ReturnType<typeof vi.fn>).mockReturnValue({
      isAuthenticated: false,
      isLoading: false,
    });

    render(
      <MemoryRouter initialEntries={['/projects/42?tab=history']}>
        <Routes>
          <Route
            path="/projects/:id"
            element={
              <ProtectedRoute>
                <div>Protected Content</div>
              </ProtectedRoute>
            }
          />
          <Route path="/login" element={<LocationProbe />} />
        </Routes>
      </MemoryRouter>,
    );

    // Redirected to /login carrying the attempted path in RR location state.
    expect(screen.getByTestId('pathname')).toHaveTextContent('/login');
    expect(screen.getByTestId('from')).toHaveTextContent('/projects/42?tab=history');

    // ...and the same path is persisted to returnTo for the OAuth roundtrip.
    expect(getReturnTo()).toBe('/projects/42?tab=history');
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
  });
});
