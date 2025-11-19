import type { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '@/components/data-table/data-table-column-header';
import { Badge } from '@/components/ui/badge';
import { Calendar } from 'lucide-react';
import type { AuditLog } from '@/lib/api/types';

// Format date
const formatDate = (dateString: string) => {
  try {
    return new Date(dateString).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  } catch {
    return dateString;
  }
};

// Format details as JSON
const formatDetails = (details?: Record<string, unknown>): string => {
  if (!details) return '-';
  try {
    return JSON.stringify(details, null, 2);
  } catch {
    return String(details);
  }
};

export interface AuditLogsColumnsProps {
  getUserById: (userId: string) => { email?: string; id: string } | undefined;
}

export const getAuditLogsColumns = ({
  getUserById,
}: AuditLogsColumnsProps): ColumnDef<AuditLog>[] => [
  {
    accessorKey: 'timestamp',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Timestamp" />,
    cell: ({ row }) => {
      return (
        <div className="flex items-center gap-2">
          <Calendar className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm">{formatDate(row.getValue('timestamp'))}</span>
        </div>
      );
    },
  },
  {
    id: 'user',
    header: 'User',
    cell: ({ row }) => {
      const log = row.original;
      if (!log.user_id) {
        return <span className="text-sm text-muted-foreground italic">Unknown</span>;
      }
      const logUser = getUserById(log.user_id);
      return <span className="text-sm">{logUser?.email || log.user_id.slice(0, 8)}</span>;
    },
    enableSorting: false,
  },
  {
    accessorKey: 'action',
    header: 'Action',
    cell: ({ row }) => <Badge variant="outline">{row.getValue('action')}</Badge>,
    enableSorting: false,
  },
  {
    accessorKey: 'resource_type',
    header: 'Resource Type',
    cell: ({ row }) => <Badge variant="secondary">{row.getValue('resource_type')}</Badge>,
    enableSorting: false,
  },
  {
    accessorKey: 'resource_id',
    header: 'Resource ID',
    cell: ({ row }) => {
      const resourceId = row.getValue('resource_id') as string | null | undefined;
      const displayValue =
        typeof resourceId === 'string' && resourceId.length > 0
          ? `${resourceId.slice(0, 8)}...`
          : '—';
      return <code className="text-xs">{displayValue}</code>;
    },
    enableSorting: false,
  },
  {
    accessorKey: 'ip_address',
    header: 'IP Address',
    cell: ({ row }) => {
      const ipAddress = row.getValue('ip_address') as string | null;
      return <span className="text-sm text-muted-foreground">{ipAddress || '-'}</span>;
    },
    enableSorting: false,
  },
  {
    id: 'details',
    header: 'Details',
    cell: ({ row }) => {
      const log = row.original;
      return log.details ? (
        <details className="text-xs">
          <summary className="cursor-pointer text-primary hover:underline">View Details</summary>
          <pre className="mt-2 p-2 bg-muted rounded-md overflow-x-auto max-w-md">
            {formatDetails(log.details)}
          </pre>
        </details>
      ) : (
        <span className="text-sm text-muted-foreground">-</span>
      );
    },
    enableSorting: false,
  },
];
