import React from 'react';
import { Badge } from '@/components/ui/badge';
import type { ChangePlanStatus } from '@/lib/api/types';

interface ChangePlanStatusBadgeProps {
  status: ChangePlanStatus;
  className?: string;
}

/**
 * Reusable component for displaying change plan status badges.
 * Provides consistent styling and variant mapping across the application.
 */
export const ChangePlanStatusBadge: React.FC<ChangePlanStatusBadgeProps> = ({
  status,
  className,
}) => {
  // Get status badge variant
  const getStatusBadgeVariant = (
    status: ChangePlanStatus,
  ): 'default' | 'secondary' | 'destructive' | 'outline' => {
    switch (status) {
      case 'Open':
        return 'default';
      case 'Approved':
        return 'default';
      case 'Completed':
        return 'default';
      case 'Draft':
        return 'secondary';
      case 'InProgress':
        return 'outline';
      case 'Failed':
        return 'destructive';
      case 'Cancelled':
        return 'secondary';
      case 'Closed':
        return 'secondary';
      default:
        return 'secondary';
    }
  };

  // Get status badge color class
  const getStatusBadgeClass = (status: ChangePlanStatus): string => {
    switch (status) {
      case 'Open':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200';
      case 'Approved':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
      case 'Completed':
        return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-200';
      case 'Draft':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
      case 'InProgress':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
      case 'Failed':
        return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
      case 'Cancelled':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
      case 'Closed':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
    }
  };

  return (
    <Badge
      variant={getStatusBadgeVariant(status)}
      className={`${getStatusBadgeClass(status)} ${className || ''}`}
    >
      {status}
    </Badge>
  );
};
