import { apiClient, setToken, clearToken } from '../client';
import type { LoginRequest, LoginResponse, RegisterRequest, RegisterResponse } from '../types';

export const authService = {
  /**
   * Login with email and password
   */
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await apiClient.post<LoginResponse>('/api/v1/auth/login', credentials);

    // Store the JWT token
    if (response.token) {
      setToken(response.token);
    }

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
   * Initiate Google OAuth flow
   */
  initiateGoogleAuth(): void {
    // Redirect to backend OAuth endpoint
    const baseURL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';
    window.location.href = `${baseURL}/api/v1/auth/google`;
  },

  /**
   * Logout - clear token
   */
  logout(): void {
    clearToken();
    // Dispatch custom event for auth context to handle
    window.dispatchEvent(new CustomEvent('auth:logout'));
  },
};
