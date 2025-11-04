import { describe, it, expect, vi, beforeEach } from 'vitest';
import axios, { type AxiosInstance } from 'axios';
import APIClient, { apiClient } from './client';
import { checkAuthStatus } from './auth';

// Mock axios.create to return a mock instance
type InterceptorHandler = {
  fulfilled?: (value: unknown) => unknown;
  rejected?: (error: unknown) => unknown;
};

const mockAxiosInstance = {
  interceptors: {
    request: {
      use: vi.fn(),
    },
    response: {
      use: vi.fn(),
      handlers: [] as InterceptorHandler[],
    },
  },
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
} as unknown as AxiosInstance;

vi.mock('axios', async () => {
  const actual = await vi.importActual('axios');
  return {
    ...actual,
    default: {
      ...actual.default,
      create: vi.fn(() => mockAxiosInstance),
    },
  };
});

// Mock fetch for auth utilities
globalThis.fetch = vi.fn() as typeof fetch;

describe('APIClient', () => {
  const mockBaseURL = 'http://localhost:8080';

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('constructor', () => {
    it('should create axios instance with correct base URL', () => {
      new APIClient(mockBaseURL);
      expect(axios.create).toHaveBeenCalledWith(
        expect.objectContaining({
          baseURL: mockBaseURL,
          timeout: 30000,
          headers: {
            'Content-Type': 'application/json',
          },
          withCredentials: true,
        }),
      );
    });

    it('should use default base URL if not provided', () => {
      new APIClient();
      expect(axios.create).toHaveBeenCalledWith(
        expect.objectContaining({
          baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
        }),
      );
    });
  });

  describe('withCredentials', () => {
    it('should enable withCredentials for HttpOnly cookies', () => {
      new APIClient(mockBaseURL);
      expect(axios.create).toHaveBeenCalledWith(
        expect.objectContaining({
          withCredentials: true,
        }),
      );
    });
  });

  describe('response interceptor', () => {
    it('should set up response interceptor', () => {
      new APIClient(mockBaseURL);
      // Verify interceptor was set up
      expect(mockAxiosInstance.interceptors.response.use).toHaveBeenCalled();
    });
  });

  describe('HTTP methods', () => {
    it('should expose get method', () => {
      const client = new APIClient(mockBaseURL);
      expect(typeof client.get).toBe('function');
    });

    it('should expose post method', () => {
      const client = new APIClient(mockBaseURL);
      expect(typeof client.post).toBe('function');
    });

    it('should expose put method', () => {
      const client = new APIClient(mockBaseURL);
      expect(typeof client.put).toBe('function');
    });

    it('should expose patch method', () => {
      const client = new APIClient(mockBaseURL);
      expect(typeof client.patch).toBe('function');
    });

    it('should expose delete method', () => {
      const client = new APIClient(mockBaseURL);
      expect(typeof client.delete).toBe('function');
    });
  });

  describe('singleton instance', () => {
    it('should export singleton apiClient instance', () => {
      expect(apiClient).toBeInstanceOf(APIClient);
    });
  });
});

describe('Auth Utilities', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('checkAuthStatus', () => {
    it('should return true when auth check succeeds', async () => {
      (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: true,
      });

      const result = await checkAuthStatus();
      expect(result).toBe(true);
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/auth/me'),
        expect.objectContaining({
          credentials: 'include',
        }),
      );
    });

    it('should return false when auth check fails', async () => {
      (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
        ok: false,
      });

      const result = await checkAuthStatus();
      expect(result).toBe(false);
    });

    it('should return false when fetch throws error', async () => {
      (globalThis.fetch as ReturnType<typeof vi.fn>).mockRejectedValueOnce(
        new Error('Network error'),
      );

      const result = await checkAuthStatus();
      expect(result).toBe(false);
    });
  });
});
