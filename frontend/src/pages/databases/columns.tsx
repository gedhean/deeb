import type { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '@/components/data-table/data-table-column-header';
import { Button } from '@/components/ui/button';
import { DatabaseStatusBadge } from '@/components/DatabaseStatusBadge';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { MoreHorizontal, Pencil, Trash2, Eye, Database as DatabaseIcon } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import type { Database } from '@/lib/api/types';
import type { ColumnMeta } from '@/components/data-table/types';

// Format date
const formatDate = (dateString: string) => {
  try {
    return new Date(dateString).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    });
  } catch {
    return dateString;
  }
};

export interface DatabasesColumnsProps {
  onView: (database: Database) => void;
  onEdit: (database: Database) => void;
  onDelete: (database: Database) => void;
  canUpdate: boolean;
  canDelete: boolean;
  getProjectName: (projectId: string | null | undefined) => string;
  getEnvironmentName: (environmentId: string | null | undefined) => string;
  getInstanceName: (instanceId: string) => string;
}

export const getDatabasesColumns = ({
  onView,
  onEdit,
  onDelete,
  canUpdate,
  canDelete,
  getProjectName,
  getEnvironmentName,
  getInstanceName,
}: DatabasesColumnsProps): ColumnDef<Database>[] => [
  {
    accessorKey: 'database_name',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Database Name" />,
    cell: ({ row }) => (
      <Badge variant="secondary" className="font-mono text-xs">
        {' '}
        {row.getValue('database_name')}
      </Badge>
    ),
    meta: {
      filterConfig: {
        variant: 'search',
        placeholder: 'Search database names...',
      },
      displayName: 'Database Name',
    } as ColumnMeta,
  },
  {
    accessorKey: 'project_id',
    id: 'project',
    header: 'Project',
    cell: ({ row }) => {
      const database = row.original;
      return <span className="text-muted-foreground">{getProjectName(database.project_id)}</span>;
    },
    enableSorting: false,
    meta: {
      filterConfig: {
        variant: 'multi-select',
        title: 'Project',
        options: [],
      },
      displayName: 'Project',
    } as ColumnMeta,
    filterFn: (row, id, value) => {
      const projectId = row.getValue(id);
      return projectId ? value.includes(projectId) : false;
    },
  },
  {
    accessorKey: 'environment_id',
    id: 'environment',
    header: 'Environment',
    cell: ({ row }) => {
      const database = row.original;
      return (
        <span className="text-muted-foreground">{getEnvironmentName(database.environment_id)}</span>
      );
    },
    enableSorting: false,
    meta: {
      filterConfig: {
        variant: 'multi-select',
        title: 'Environment',
        options: [],
      },
      displayName: 'Environment',
    } as ColumnMeta,
    filterFn: (row, id, value) => {
      const environmentId = row.getValue(id);
      return environmentId ? value.includes(environmentId) : false;
    },
  },
  {
    accessorKey: 'database_instance_id',
    id: 'instance',
    header: 'Instance',
    cell: ({ row }) => {
      const database = row.original;
      return (
        <span className="text-muted-foreground">
          {getInstanceName(database.database_instance_id)}
        </span>
      );
    },
    enableSorting: false,
    meta: {
      filterConfig: {
        variant: 'multi-select',
        title: 'Instance',
        options: [],
      },
      displayName: 'Instance',
    } as ColumnMeta,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: 'status',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Status" />,
    cell: ({ row }) => <DatabaseStatusBadge status={row.getValue('status')} variant="database" />,
    enableSorting: false,
    meta: {
      filterConfig: {
        variant: 'multi-select',
        title: 'Status',
        options: [
          { label: 'Active', value: 'Active', icon: DatabaseIcon },
          { label: 'Inactive', value: 'Inactive', icon: DatabaseIcon },
          { label: 'Pending', value: 'Pending', icon: DatabaseIcon },
        ],
      },
      displayName: 'Status',
    } as ColumnMeta,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: 'created_at',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Created" />,
    cell: ({ row }) => (
      <span className="text-muted-foreground">{formatDate(row.getValue('created_at'))}</span>
    ),
    meta: {
      filterConfig: {
        variant: 'date-range',
        title: 'Created',
      },
      displayName: 'Created',
    } as ColumnMeta,
    filterFn: (row, id, value) => {
      const [start, end] = value as [Date, Date];
      const date = new Date(row.getValue(id));
      return date >= start && date <= end;
    },
  },
  {
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => {
      const database = row.original;

      return (
        <div className="flex items-center justify-end">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className="h-8 w-8 p-0"
                onClick={(e) => {
                  e.stopPropagation();
                }}
                aria-label="Open menu"
              >
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-[180px]">
              <DropdownMenuItem
                onClick={(e) => {
                  e.stopPropagation();
                  onView(database);
                }}
              >
                <Eye className="h-4 w-4" />
                View database
              </DropdownMenuItem>
              {canUpdate && (
                <DropdownMenuItem
                  onClick={(e) => {
                    e.stopPropagation();
                    onEdit(database);
                  }}
                >
                  <Pencil className="h-4 w-4" />
                  Edit database
                </DropdownMenuItem>
              )}
              {canDelete && (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    variant="destructive"
                    onClick={(e) => {
                      e.stopPropagation();
                      onDelete(database);
                    }}
                  >
                    <Trash2 className="h-4 w-4" />
                    Delete database
                  </DropdownMenuItem>
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      );
    },
  },
];
