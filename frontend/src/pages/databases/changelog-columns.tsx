import type { ColumnDef } from '@tanstack/react-table';
import { formatDistanceToNow } from 'date-fns';
import { Link } from 'react-router-dom';
import { Badge } from '@/components/ui/badge';
import type { Changelog } from '@/lib/api/types';

export const changelogColumns: ColumnDef<Changelog>[] = [
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => {
      const status = row.getValue('status') as string;
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
        <Badge variant={variant} className={`capitalize ${colorClass}`}>
          {status}
        </Badge>
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
