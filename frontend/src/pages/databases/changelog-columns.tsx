import type { ColumnDef } from '@tanstack/react-table';
import { formatDistanceToNow } from 'date-fns';
import { Link } from 'react-router-dom';
import { AlertTriangle } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import type { Changelog } from '@/lib/api/types';
import { SNAPSHOT_STATUS } from '@/lib/api/types';

export const changelogColumns: ColumnDef<Changelog>[] = [
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => {
      const status = row.getValue('status') as string;
      const metadata = row.original.metadata;
      const hasSnapshotFailure = metadata?.snapshot_status === SNAPSHOT_STATUS.FAILED;

      let variant: 'default' | 'secondary' | 'destructive' | 'outline' = 'default';
      let colorClass = '';

      if (status === 'done') {
        variant = 'default';
        colorClass = 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
      } else if (status === 'failed') {
        variant = 'destructive';
        colorClass = 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
      } else if (status === 'pending') {
        variant = 'secondary';
        colorClass = 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200';
      }

      return (
        <div className="flex items-center gap-2">
          <Badge variant={variant} className={`capitalize ${colorClass}`}>
            {status}
          </Badge>
          {hasSnapshotFailure && (
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <AlertTriangle className="h-4 w-4 text-amber-500 dark:text-amber-400" />
                </TooltipTrigger>
                <TooltipContent>
                  <p className="text-sm">
                    Schema snapshot failed: {metadata.snapshot_error || 'Unknown error'}
                  </p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          )}
        </div>
      );
    },
  },
  {
    accessorKey: 'change_plan_title',
    header: 'Change Plan',
    cell: ({ row }) => {
      const title = row.getValue('change_plan_title') as string;
      const changePlanId = row.original.change_plan_id;

      return (
        <Link
          to={`/change-plans/${changePlanId}`}
          className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-200 hover:underline"
          onClick={(e) => e.stopPropagation()}
        >
          {title || 'Unknown Change Plan'}
        </Link>
      );
    },
  },
  {
    accessorKey: 'created_at',
    header: 'Created',
    cell: ({ row }) => {
      const createdAt = row.getValue('created_at') as string;
      return (
        <span className="text-sm text-muted-foreground">
          {formatDistanceToNow(new Date(createdAt), { addSuffix: true })}
        </span>
      );
    },
  },
];
