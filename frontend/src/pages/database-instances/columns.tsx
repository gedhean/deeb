import type { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '@/components/data-table/data-table-column-header';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { DatabaseStatusBadge } from '@/components/DatabaseStatusBadge';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { MoreHorizontal, Pencil, Trash2, Eye, Server } from 'lucide-react';
import type { DatabaseInstance } from '@/lib/api/types';
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

export interface DatabaseInstancesColumnsProps {
  onView: (instance: DatabaseInstance) => void;
  onEdit: (instance: DatabaseInstance) => void;
  onDelete: (instance: DatabaseInstance) => void;
  canUpdate: boolean;
  canDelete: boolean;
  getEnvironmentName: (environmentId?: string) => string;
}

export const getDatabaseInstancesColumns = ({
  onView,
  onEdit,
  onDelete,
  canUpdate,
  canDelete,
  getEnvironmentName,
}: DatabaseInstancesColumnsProps): ColumnDef<DatabaseInstance>[] => [
  {
    accessorKey: 'name',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
    cell: ({ row }) => <span className="font-medium">{row.getValue('name')}</span>,
    meta: {
      filterConfig: {
        variant: 'search',
        placeholder: 'Search names...',
      },
      displayName: 'Name',
    } as ColumnMeta,
  },
  {
    accessorKey: 'database_engine',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Type" />,
    cell: ({ row }) => (
      <Badge variant="secondary" className="font-mono text-xs">
        {row.getValue('database_engine')}
      </Badge>
    ),
    enableSorting: false,
    meta: {
      filterConfig: {
        variant: 'multi-select',
        title: 'Engine',
        options: [
          { label: 'PostgreSQL', value: 'postgres', icon: Server },
          { label: 'MySQL', value: 'mysql', icon: Server },
        ],
      },
      displayName: 'Engine',
    } as ColumnMeta,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    id: 'host',
    accessorKey: 'host',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Host" />,
    cell: ({ row }) => {
      const instance = row.original;
      return (
        <span className="text-muted-foreground">
          {instance.host}:{instance.port}
        </span>
      );
    },
  },
  {
    accessorKey: 'username',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Username" />,
    cell: ({ row }) => <span className="text-muted-foreground">{row.getValue('username')}</span>,
  },
  {
    accessorKey: 'environment_id',
    id: 'environment',
    header: 'Environment',
    cell: ({ row }) => {
      const instance = row.original;
      return (
        <span className="text-muted-foreground">{getEnvironmentName(instance.environment_id)}</span>
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
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: 'status',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Status" />,
    cell: ({ row }) => <DatabaseStatusBadge status={row.getValue('status')} variant="instance" />,
    enableSorting: false,
    meta: {
      filterConfig: {
        variant: 'multi-select',
        title: 'Status',
        options: [
          { label: 'Active', value: 'Active', icon: Server },
          { label: 'Inactive', value: 'Inactive', icon: Server },
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
    cell: ({ row }) => formatDate(row.getValue('created_at')),
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
      const instance = row.original;

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
            <DropdownMenuContent align="end" className="w-[200px]">
              <DropdownMenuItem
                onClick={(e) => {
                  e.stopPropagation();
                  onView(instance);
                }}
              >
                <Eye className="h-4 w-4" />
                View instance
              </DropdownMenuItem>
              {canUpdate && (
                <DropdownMenuItem
                  onClick={(e) => {
                    e.stopPropagation();
                    onEdit(instance);
                  }}
                >
                  <Pencil className="h-4 w-4" />
                  Edit instance
                </DropdownMenuItem>
              )}
              {canDelete && (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    variant="destructive"
                    onClick={(e) => {
                      e.stopPropagation();
                      onDelete(instance);
                    }}
                  >
                    <Trash2 className="h-4 w-4" />
                    Delete instance
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
