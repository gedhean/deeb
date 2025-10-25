import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { authService } from '@/lib/api/services/auth';
import { getToken, clearToken } from '@/lib/api/client';
import type { User, LoginRequest, RegisterRequest } from '@/lib/api/types';

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (credentials: LoginRequest) => Promise<void>;
  register: (data: RegisterRequest) => Promise<void>;
  logout: () => void;
  initiateGoogleAuth: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);

  // Load user from token on mount
  useEffect(() => {
    const loadUser = async () => {
      const token = getToken();
      if (token) {
        try {
          // Decode JWT to get user info (basic implementation)
          // In production, you might want to validate the token with the backend
          const payload = JSON.parse(atob(token.split('.')[1]));

          // Check if token is expired
          const now = Date.now() / 1000;
          if (payload.exp && payload.exp < now) {
            clearToken();
            setUser(null);
          } else {
            // Set user from token payload
            setUser({
              id: payload.user_id || '',
              email: payload.email || '',
              organization_role: payload.role || '',
              is_active: true,
              created_at: '',
              updated_at: '',
            });
          }
        } catch (error) {
          console.error('Failed to load user from token:', error);
          clearToken();
          setUser(null);
        }
      }
      setIsLoading(false);
    };

    loadUser();
  }, []);

  // Listen for unauthorized events from API client
  useEffect(() => {
    const handleUnauthorized = () => {
      setUser(null);
    };

    const handleLogout = () => {
      setUser(null);
    };

    window.addEventListener('auth:unauthorized', handleUnauthorized);
    window.addEventListener('auth:logout', handleLogout);

    return () => {
      window.removeEventListener('auth:unauthorized', handleUnauthorized);
      window.removeEventListener('auth:logout', handleLogout);
    };
  }, []);

  // Token expiration check - runs every minute
  useEffect(() => {
    if (!user) return;

    const checkTokenExpiration = () => {
      const token = getToken();
      if (!token) {
        setUser(null);
        return;
      }

      try {
        const payload = JSON.parse(atob(token.split('.')[1]));
        const now = Date.now() / 1000;

        // Check if token is expired or will expire in the next 5 minutes
        if (payload.exp && payload.exp < now + 300) {
          // Token expired or about to expire
          // In a real app, you would refresh the token here
          // For now, we'll log the user out
          console.warn('Token expired or about to expire');
          clearToken();
          setUser(null);
        }
      } catch (error) {
        console.error('Failed to check token expiration:', error);
        clearToken();
        setUser(null);
      }
    };

    // Check immediately
    checkTokenExpiration();

    // Check every minute
    const interval = setInterval(checkTokenExpiration, 60000);

    return () => clearInterval(interval);
  }, [user]);

  const login = useCallback(async (credentials: LoginRequest) => {
    try {
      const response = await authService.login(credentials);
      setUser(response.user);
    } catch (error) {
      console.error('Login failed:', error);
      throw error;
    }
  }, []);

  const register = useCallback(async (data: RegisterRequest) => {
    try {
      await authService.register(data);
      // After registration, user needs to login
    } catch (error) {
      console.error('Registration failed:', error);
      throw error;
    }
  }, []);

  const logout = useCallback(() => {
    authService.logout();
    setUser(null);
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
