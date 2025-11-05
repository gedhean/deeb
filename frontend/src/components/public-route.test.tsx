import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { PublicRoute } from '@/components/public-route';
import { useAuth } from '@/contexts/AuthContext';

// Mock AuthContext
vi.mock('@/contexts/AuthContext', () => ({
  useAuth: vi.fn(),
}));

// Mock LoadingSpinner
vi.mock('@/components/ui/loading-spinner', () => ({
  LoadingSpinner: ({ text }: { text?: string }) => <div>Loading: {text}</div>,
}));

describe('PublicRoute', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render children when not authenticated', () => {
    (useAuth as ReturnType<typeof vi.fn>).mockReturnValue({
      isAuthenticated: false,
      isLoading: false,
    });

    render(
      <BrowserRouter>
        <PublicRoute>
          <div>Public Content</div>
        </PublicRoute>
      </BrowserRouter>,
    );

    expect(screen.getByText('Public Content')).toBeInTheDocument();
  });

  it('should show loading spinner when loading', () => {
    (useAuth as ReturnType<typeof vi.fn>).mockReturnValue({
      isAuthenticated: false,
      isLoading: true,
    });

    render(
      <BrowserRouter>
        <PublicRoute>
          <div>Public Content</div>
        </PublicRoute>
      </BrowserRouter>,
    );

    expect(screen.getByText(/Loading:/)).toBeInTheDocument();
    expect(screen.queryByText('Public Content')).not.toBeInTheDocument();
  });

  it('should redirect to dashboard when authenticated', () => {
    (useAuth as ReturnType<typeof vi.fn>).mockReturnValue({
      isAuthenticated: true,
      isLoading: false,
    });

    render(
      <BrowserRouter>
        <PublicRoute>
          <div>Public Content</div>
        </PublicRoute>
      </BrowserRouter>,
    );

    // Navigate component renders but doesn't show content
    // We verify by checking that public content is not shown
    expect(screen.queryByText('Public Content')).not.toBeInTheDocument();
  });
});
