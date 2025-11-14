import React from 'react';
import { Badge } from '@/components/ui/badge';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import type { DatabaseStatus, DatabaseInstanceStatus } from '@/lib/api/types';

type Status = DatabaseStatus | DatabaseInstanceStatus;

interface DatabaseStatusBadgeProps {
  status: Status;
  className?: string;
  variant?: 'database' | 'instance';
}

/**
 * Reusable component for displaying database status badges with tooltips.
 * Provides consistent styling and helpful explanations for each status.
 */
export const DatabaseStatusBadge: React.FC<DatabaseStatusBadgeProps> = ({
  status,
  className,
  variant = 'database',
}) => {
  // Get status badge color class
  const getStatusBadgeClass = (status: Status): string => {
    switch (status) {
      case 'Active':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
      case 'Inactive':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
      case 'Pending':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200';
      case 'Maintenance':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
    }
  };

  // Get tooltip description for status
  const getStatusTooltip = (status: Status, variant: 'database' | 'instance'): string => {
    switch (status) {
      case 'Active':
        return variant === 'database'
          ? 'This database is active and available for use. Change plans can be executed against it.'
          : 'This database instance is active and available. Connections can be established.';
      case 'Inactive':
        return variant === 'database'
          ? 'This database is inactive and not available for use. Change plans cannot be executed against it.'
          : 'This database instance is inactive. Connections cannot be established.';
      case 'Pending':
        return 'This database is pending and awaiting creation. Change plans can be executed against it after it is created.';
      case 'Maintenance':
        return 'This database instance is in maintenance mode. It may be temporarily unavailable for connections or operations.';
      default:
        return 'Unknown status';
    }
  };

  const badgeContent = (
    <Badge className={`${getStatusBadgeClass(status)} ${className || ''}`}>{status}</Badge>
  );

  return (
    <Tooltip>
      <TooltipTrigger asChild>{badgeContent}</TooltipTrigger>
      <TooltipContent>
        <p>{getStatusTooltip(status, variant)}</p>
      </TooltipContent>
    </Tooltip>
  );
};
