import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route, useLocation } from 'react-router-dom';
import { LoginPage } from '@/pages/LoginPage';
import { useAuth } from '@/contexts/AuthContext';
import { getReturnTo, setReturnTo } from '@/lib/auth/return-to';

// Mock AuthContext so login resolves without hitting the network.
const login = vi.fn();
vi.mock('@/contexts/AuthContext', () => ({
  useAuth: vi.fn(),
}));

// Stub the animated background/text components to keep the test focused and fast.
vi.mock('@/components/ui/shooting-stars', () => ({ ShootingStars: () => null }));
vi.mock('@/components/ui/stars-background', () => ({ StarsBackground: () => null }));
vi.mock('@/components/ui/encrypted-text', () => ({
  EncryptedText: ({ text }: { text: string }) => <span>{text}</span>,
}));

// Renders the pathname so tests can assert where login navigated.
const LocationProbe = () => {
  const location = useLocation();
  return <span data-testid="pathname">{location.pathname}</span>;
};

const renderLoginAt = (entry: string | { pathname: string; state?: unknown }) =>
  render(
    <MemoryRouter initialEntries={[entry]}>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="*" element={<LocationProbe />} />
      </Routes>
    </MemoryRouter>,
  );

const submitLogin = async () => {
  const user = userEvent.setup();
  await user.type(screen.getByPlaceholderText('you@example.com'), 'user@example.com');
  await user.type(screen.getByPlaceholderText('Enter your password'), 'secret123');
  await user.click(screen.getByRole('button', { name: /sign in$/i }));
};

describe('LoginPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
    login.mockResolvedValue(undefined);
    (useAuth as ReturnType<typeof vi.fn>).mockReturnValue({
      login,
      initiateGoogleAuth: vi.fn(),
    });
  });

  it('navigates to the RR state "from" path after successful login', async () => {
    renderLoginAt({ pathname: '/login', state: { from: '/projects/42' } });

    await submitLogin();

    await waitFor(() => expect(screen.getByTestId('pathname')).toHaveTextContent('/projects/42'));
    expect(login).toHaveBeenCalledWith({ email: 'user@example.com', password: 'secret123' });
  });

  it('falls back to returnTo when RR state is absent', async () => {
    setReturnTo('/environments');
    renderLoginAt('/login');

    await submitLogin();

    await waitFor(() => expect(screen.getByTestId('pathname')).toHaveTextContent('/environments'));
    expect(getReturnTo()).toBeNull();
  });

  it('falls back to /dashboard when neither state nor returnTo is set', async () => {
    renderLoginAt('/login');

    await submitLogin();

    await waitFor(() => expect(screen.getByTestId('pathname')).toHaveTextContent('/dashboard'));
  });
});
