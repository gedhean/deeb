import { apiClient } from '../client';
import type {
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  RegisterResponse,
  User,
} from '../types';

export const authService = {
  /**
   * Login with email and password
   * JWT token is set in HttpOnly cookie by backend
   */
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await apiClient.post<LoginResponse>('/api/v1/auth/login', credentials);
    return response;
  },

  /**
   * Register a new user
   */
  async register(data: RegisterRequest): Promise<RegisterResponse> {
    const response = await apiClient.post<RegisterResponse>('/api/v1/auth/register', data);
    return response;
  },

  /**
   * Get current user info from backend
   * This validates the HttpOnly cookie and returns user data
   */
  async getCurrentUser(): Promise<User> {
    const response = await apiClient.get<{ user: User }>('/api/v1/auth/me');
    return response.user;
  },

  /**
   * Initiate Google OAuth flow
   */
  initiateGoogleAuth(): void {
    // Redirect to backend OAuth endpoint
    const baseURL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';
    window.location.href = `${baseURL}/api/v1/auth/google`;
  },

  /**
   * Logout - clear HttpOnly cookie
   */
  async logout(): Promise<void> {
    try {
      await apiClient.post('/api/v1/auth/logout');
    } catch (error) {
      // Even if logout fails, we should still clear local state
      console.error('Logout request failed:', error);
    }
    // Dispatch custom event for auth context to handle
    window.dispatchEvent(new CustomEvent('auth:logout'));
  },
};
