import type { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '@/components/data-table/data-table-column-header';
import { DatabaseStatusBadge } from '@/components/DatabaseStatusBadge';
import type { Database } from '@/lib/api/types';

// Format date
const formatDate = (dateString: string) => {
  try {
    return new Date(dateString).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return dateString;
  }
};

export interface DatabasesColumnsProps {
  getProjectName: (projectId: string | null | undefined) => string;
  getEnvironmentName: (environmentId: string | null | undefined) => string;
}

export const getDatabasesColumns = ({
  getProjectName,
  getEnvironmentName,
}: DatabasesColumnsProps): ColumnDef<Database>[] => [
  {
    accessorKey: 'database_name',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Database Name" />,
    cell: ({ row }) => <span className="font-medium">{row.getValue('database_name')}</span>,
  },
  {
    id: 'project',
    header: 'Project',
    cell: ({ row }) => {
      const database = row.original;
      return <span className="text-muted-foreground">{getProjectName(database.project_id)}</span>;
    },
    enableSorting: false,
  },
  {
    id: 'environment',
    header: 'Environment',
    cell: ({ row }) => {
      const database = row.original;
      return (
        <span className="text-muted-foreground">{getEnvironmentName(database.environment_id)}</span>
      );
    },
    enableSorting: false,
  },
  {
    accessorKey: 'status',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Status" />,
    cell: ({ row }) => <DatabaseStatusBadge status={row.getValue('status')} variant="database" />,
  },
  {
    accessorKey: 'created_at',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Created" />,
    cell: ({ row }) => formatDate(row.getValue('created_at')),
  },
];
