import React from 'react';
import { AlertTriangle, XCircle, Info, AlertCircle } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { APIError } from '@/lib/api/types';

export interface ErrorDisplayProps {
  error: string | Error | APIError | null | undefined;
  title?: string;
  onRetry?: () => void;
  onDismiss?: () => void;
  className?: string;
  variant?: 'default' | 'destructive' | 'warning' | 'info';
}

/**
 * ErrorDisplay component for displaying user-friendly error messages
 */
export const ErrorDisplay: React.FC<ErrorDisplayProps> = ({
  error,
  title,
  onRetry,
  onDismiss,
  className = '',
  variant = 'destructive',
}) => {
  if (!error) {
    return null;
  }

  // Extract error message
  const errorMessage =
    error instanceof APIError
      ? error.message
      : error instanceof Error
        ? error.message
        : typeof error === 'string'
          ? error
          : 'An unexpected error occurred';

  // Determine variant styling
  const variantStyles = {
    destructive: {
      container: 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950/20',
      icon: 'text-red-600 dark:text-red-400',
      title: 'text-red-800 dark:text-red-200',
      message: 'text-red-700 dark:text-red-300',
      iconComponent: XCircle,
    },
    warning: {
      container: 'border-yellow-200 bg-yellow-50 dark:border-yellow-800 dark:bg-yellow-950/20',
      icon: 'text-yellow-600 dark:text-yellow-400',
      title: 'text-yellow-800 dark:text-yellow-200',
      message: 'text-yellow-700 dark:text-yellow-300',
      iconComponent: AlertTriangle,
    },
    info: {
      container: 'border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-950/20',
      icon: 'text-blue-600 dark:text-blue-400',
      title: 'text-blue-800 dark:text-blue-200',
      message: 'text-blue-700 dark:text-blue-300',
      iconComponent: Info,
    },
    default: {
      container: 'border-gray-200 bg-gray-50 dark:border-gray-800 dark:bg-gray-950/20',
      icon: 'text-gray-600 dark:text-gray-400',
      title: 'text-gray-800 dark:text-gray-200',
      message: 'text-gray-700 dark:text-gray-300',
      iconComponent: AlertCircle,
    },
  };

  const styles = variantStyles[variant];
  const IconComponent = styles.iconComponent;

  return (
    <Card className={`border ${styles.container} ${className}`}>
      <CardContent className="pt-6">
        <div className="flex items-start gap-3">
          <IconComponent className={`h-5 w-5 ${styles.icon} mt-0.5 flex-shrink-0`} />
          <div className="flex-1 min-w-0">
            {title && <p className={`text-sm font-semibold ${styles.title} mb-1`}>{title}</p>}
            <p className={`text-sm ${styles.message} whitespace-pre-wrap break-words`}>
              {errorMessage}
            </p>
            {(onRetry || onDismiss) && (
              <div className="flex items-center gap-2 mt-3">
                {onRetry && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={onRetry}
                    className={
                      variant === 'destructive'
                        ? 'border-red-300 text-red-700 hover:bg-red-100 dark:border-red-700 dark:text-red-300 dark:hover:bg-red-900/30'
                        : ''
                    }
                  >
                    Try Again
                  </Button>
                )}
                {onDismiss && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={onDismiss}
                    className={
                      variant === 'destructive'
                        ? 'text-red-700 hover:bg-red-100 dark:text-red-300 dark:hover:bg-red-900/30'
                        : ''
                    }
                  >
                    Dismiss
                  </Button>
                )}
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
};
