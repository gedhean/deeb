/* eslint-disable @typescript-eslint/no-explicit-any */
import axios, { type AxiosInstance, AxiosError } from 'axios';
import { APIError, type ErrorResponse } from './types';

// API Client Configuration
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

class APIClient {
  private client: AxiosInstance;

  constructor(baseURL: string = API_BASE_URL) {
    this.client = axios.create({
      baseURL,
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
      withCredentials: true, // Enable sending cookies with requests
    });

    this.setupInterceptors();
  }

  private setupInterceptors(): void {
    // Response Interceptor - Handle errors consistently
    this.client.interceptors.response.use(
      (response) => response,
      (error: AxiosError) => {
        // Handle 401 Unauthorized - dispatch event for auth context
        if (error.response?.status === 401) {
          // Dispatch custom event for auth context to handle
          window.dispatchEvent(new CustomEvent('auth:unauthorized'));
        }

        // Transform error into APIError
        const statusCode = error.response?.status || 500;
        const errorResponse = error.response?.data as ErrorResponse;
        const message = errorResponse?.error || error.message || 'An unexpected error occurred';

        return Promise.reject(new APIError(message, statusCode, errorResponse));
      },
    );
  }

  // Expose axios instance for direct use if needed
  public getInstance(): AxiosInstance {
    return this.client;
  }

  // Helper methods for common HTTP methods
  public async get<T>(url: string, config?: any): Promise<T> {
    const response = await this.client.get<T>(url, config);
    return response.data;
  }

  public async post<T>(url: string, data?: any, config?: any): Promise<T> {
    const response = await this.client.post<T>(url, data, config);
    return response.data;
  }

  public async put<T>(url: string, data?: any, config?: any): Promise<T> {
    const response = await this.client.put<T>(url, data, config);
    return response.data;
  }

  public async patch<T>(url: string, data?: any, config?: any): Promise<T> {
    const response = await this.client.patch<T>(url, data, config);
    return response.data;
  }

  public async delete<T>(url: string, config?: any): Promise<T> {
    const response = await this.client.delete<T>(url, config);
    return response.data;
  }
}

// Export singleton instance
export const apiClient = new APIClient();

// Export API client class for testing
export default APIClient;
