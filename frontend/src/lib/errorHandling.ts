import { APIError } from '@/lib/api/types';
import { toast } from 'sonner';

/**
 * Display a user-friendly error message based on the error type
 */
export function getErrorMessage(error: unknown): string {
  if (error instanceof APIError) {
    // Return API error message directly
    return error.message;
  }

  if (error instanceof Error) {
    // Return error message
    return error.message;
  }

  if (typeof error === 'string') {
    // Return string directly
    return error;
  }

  // Default fallback
  return 'An unexpected error occurred. Please try again.';
}

/**
 * Show error toast notification
 */
export function showErrorToast(
  error: unknown,
  title?: string,
  options?: {
    duration?: number;
    action?: {
      label: string;
      onClick: () => void;
    };
  },
): void {
  const message = getErrorMessage(error);
  const errorTitle = title || 'Error';

  toast.error(errorTitle, {
    description: message,
    duration: options?.duration || 5000,
    action: options?.action,
  });
}

/**
 * Show success toast notification
 */
export function showSuccessToast(message: string, title?: string, duration?: number): void {
  toast.success(title || 'Success', {
    description: message,
    duration: duration || 3000,
  });
}

/**
 * Show info toast notification
 */
export function showInfoToast(message: string, title?: string, duration?: number): void {
  toast.info(title || 'Info', {
    description: message,
    duration: duration || 3000,
  });
}

/**
 * Show warning toast notification
 */
export function showWarningToast(message: string, title?: string, duration?: number): void {
  toast.warning(title || 'Warning', {
    description: message,
    duration: duration || 4000,
  });
}

/**
 * Handle API errors with toast notification and optional retry
 */
export function handleApiError(
  error: unknown,
  options?: {
    showToast?: boolean;
    toastTitle?: string;
    onRetry?: () => void;
    fallbackMessage?: string;
  },
): string {
  const message = error instanceof APIError ? error.message : getErrorMessage(error);
  const displayMessage = options?.fallbackMessage || message;

  if (options?.showToast !== false) {
    showErrorToast(error, options?.toastTitle, {
      action: options?.onRetry
        ? {
            label: 'Retry',
            onClick: options.onRetry,
          }
        : undefined,
    });
  }

  return displayMessage;
}

/**
 * Check if error is retryable (network errors, 5xx errors)
 */
export function isRetryableError(error: unknown): boolean {
  if (error instanceof APIError) {
    // Network errors and server errors are retryable
    if (error.statusCode >= 500) {
      return true;
    }
    // Timeout errors
    if (error.statusCode === 408 || error.statusCode === 504) {
      return true;
    }
  }

  // Network errors (no response)
  if (error instanceof Error) {
    if (error.message.includes('network') || error.message.includes('Network')) {
      return true;
    }
    if (error.message.includes('timeout') || error.message.includes('Timeout')) {
      return true;
    }
  }

  return false;
}

/**
 * Get user-friendly error message based on status code
 */
export function getUserFriendlyErrorMessage(error: unknown): string {
  if (error instanceof APIError) {
    switch (error.statusCode) {
      case 400:
        return 'Invalid request. Please check your input and try again.';
      case 401:
        return 'You are not authenticated. Please log in again.';
      case 403:
        return 'You do not have permission to perform this action.';
      case 404:
        return 'The requested resource was not found.';
      case 409:
        return 'This resource already exists. Please check and try again.';
      case 422:
        return 'The request could not be processed. Please check your input.';
      case 429:
        return 'Too many requests. Please wait a moment and try again.';
      case 500:
        return 'An internal server error occurred. Please try again later.';
      case 502:
      case 503:
      case 504:
        return 'The service is temporarily unavailable. Please try again later.';
      default:
        return error.message || 'An unexpected error occurred. Please try again.';
    }
  }

  return getErrorMessage(error);
}
