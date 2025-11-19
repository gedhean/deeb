import type { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '@/components/data-table/data-table-column-header';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { MoreHorizontal, Pencil, Trash2 } from 'lucide-react';
import type { Environment } from '@/lib/api/types';
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

export interface EnvironmentsColumnsProps {
  onEdit: (environment: Environment) => void;
  onDelete: (environment: Environment) => void;
  canUpdate: boolean;
  canDelete: boolean;
  getPolicyName: (policyId?: string) => string;
}

export const getEnvironmentsColumns = ({
  onEdit,
  onDelete,
  canUpdate,
  canDelete,
  getPolicyName,
}: EnvironmentsColumnsProps): ColumnDef<Environment>[] => [
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
    accessorKey: 'description',
    header: 'Description',
    cell: ({ row }) => {
      const description = row.getValue('description') as string | null;
      return <span className="text-muted-foreground">{description || '-'}</span>;
    },
    enableSorting: false,
  },
  {
    accessorKey: 'policy_id',
    id: 'policy',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Policy" />,
    cell: ({ row }) => {
      const policyId = row.getValue('policy_id') as string | undefined;
      return getPolicyName(policyId);
    },
    enableSorting: false,
    meta: {
      filterConfig: {
        variant: 'multi-select',
        title: 'Policy',
        options: [],
      },
      displayName: 'Policy',
    } as ColumnMeta,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: 'allow_self_approval',
    header: 'Self Approval',
    cell: ({ row }) => {
      const allowSelfApproval = row.getValue('allow_self_approval') as boolean;
      return allowSelfApproval ? (
        <Badge variant="secondary" className="bg-green-100 text-green-800">
          Enabled
        </Badge>
      ) : (
        <Badge variant="secondary" className="bg-gray-100 text-gray-800">
          Disabled
        </Badge>
      );
    },
    enableSorting: false,
    meta: {
      filterConfig: {
        variant: 'multi-select',
        title: 'Self Approval',
        options: [
          { label: 'Enabled', value: 'true' },
          { label: 'Disabled', value: 'false' },
        ],
      },
      displayName: 'Self Approval',
    } as ColumnMeta,
    filterFn: (row, id, value) => {
      const rowValue = String(row.getValue(id));
      return value.includes(rowValue);
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
      const environment = row.original;

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
              {canUpdate && (
                <DropdownMenuItem
                  onClick={(e) => {
                    e.stopPropagation();
                    onEdit(environment);
                  }}
                >
                  <Pencil className="h-4 w-4" />
                  Edit environment
                </DropdownMenuItem>
              )}
              {canDelete && (
                <>
                  {canUpdate && <DropdownMenuSeparator />}
                  <DropdownMenuItem
                    variant="destructive"
                    onClick={(e) => {
                      e.stopPropagation();
                      onDelete(environment);
                    }}
                  >
                    <Trash2 className="h-4 w-4" />
                    Delete environment
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
