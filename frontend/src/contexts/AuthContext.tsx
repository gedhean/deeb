import React, { createContext, useContext, useCallback, useEffect } from 'react';
import { authService } from '@/lib/api/services/auth';
import type { User, LoginRequest, RegisterRequest } from '@/lib/api/types';
import { useCurrentUserSWR } from '@/hooks/api/useAuthSWR';

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (credentials: LoginRequest) => Promise<void>;
  register: (data: RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
  initiateGoogleAuth: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  // Use SWR hook for automatic caching and deduplication
  const { user, isLoading, refetch } = useCurrentUserSWR();

  // Listen for auth events from API client
  useEffect(() => {
    const handleUnauthorized = () => {
      // DO NOT call refetch() here - it causes infinite loop!
      // The 401 response from /api/auth/me triggers this event,
      // and calling refetch() would make another request that gets 401,
      // triggering the event again, creating an infinite loop.
      // The user state will naturally be null/undefined when not authenticated.
    };

    const handleLogout = () => {
      // Clear user data by triggering a refetch (which will get 401 and clear state)
      // Navigation happens in the logout callback function
      refetch();
    };

    window.addEventListener('auth:unauthorized', handleUnauthorized);
    window.addEventListener('auth:logout', handleLogout);

    return () => {
      window.removeEventListener('auth:unauthorized', handleUnauthorized);
      window.removeEventListener('auth:logout', handleLogout);
    };
  }, [refetch]);

  const login = useCallback(
    async (credentials: LoginRequest) => {
      try {
        await authService.login(credentials);
        // Refetch user data after successful login
        await refetch();
      } catch (error) {
        console.error('Login failed:', error);
        throw error;
      }
    },
    [refetch],
  );

  const register = useCallback(async (data: RegisterRequest) => {
    try {
      await authService.register(data);
      // After registration, user needs to login
    } catch (error) {
      console.error('Registration failed:', error);
      throw error;
    }
  }, []);

  const logout = useCallback(async () => {
    // authService.logout() dispatches 'auth:logout' event which triggers refetch
    await authService.logout();
    // Hard redirect to login page after logout
    window.location.replace('/login');
  }, []);

  const initiateGoogleAuth = useCallback(() => {
    authService.initiateGoogleAuth();
  }, []);

  const value: AuthContextType = {
    user,
    isAuthenticated: !!user,
    isLoading,
    login,
    register,
    logout,
    initiateGoogleAuth,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
