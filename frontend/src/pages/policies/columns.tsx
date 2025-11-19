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
import { MoreHorizontal, Pencil, Trash2, FileCheck2, AlertTriangle, Info } from 'lucide-react';
import type { Policy } from '@/lib/api/types';
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

export interface PoliciesColumnsProps {
  onEdit: (policy: Policy) => void;
  onDelete: (policy: Policy) => void;
  canUpdate: boolean;
  canDelete: boolean;
  getReferencingEnvironmentsCount: (policyId: string) => number;
}

export const getPoliciesColumns = ({
  onEdit,
  onDelete,
  canUpdate,
  canDelete,
  getReferencingEnvironmentsCount,
}: PoliciesColumnsProps): ColumnDef<Policy>[] => [
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
      return (
        <span className="text-muted-foreground line-clamp-2">
          {description || 'No description'}
        </span>
      );
    },
    enableSorting: false,
  },
  {
    accessorKey: 'type',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Type" />,
    cell: ({ row }) => (
      <Badge variant="secondary" className="font-mono text-xs">
        {row.getValue('type')}
      </Badge>
    ),
    enableSorting: false,
    meta: {
      filterConfig: {
        variant: 'multi-select',
        title: 'Type',
        options: [
          { label: 'Syntax', value: 'Syntax', icon: FileCheck2 },
          { label: 'Security', value: 'Security', icon: AlertTriangle },
          { label: 'Performance', value: 'Performance', icon: Info },
          { label: 'Backwards Compatibility', value: 'BackwardsCompatibility', icon: FileCheck2 },
          { label: 'Best Practices', value: 'BestPractices', icon: Info },
        ],
      },
      displayName: 'Type',
    } as ColumnMeta,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    id: 'environments',
    header: 'Environments',
    cell: ({ row }) => {
      const policy = row.original;
      const count = getReferencingEnvironmentsCount(policy.id);
      return count > 0 ? (
        <div className="flex items-center gap-2">
          <Badge variant="outline" className="text-xs">
            {count}
          </Badge>
          <span className="text-xs text-yellow-600 dark:text-yellow-400">⚠️ Referenced</span>
        </div>
      ) : (
        <span className="text-xs text-muted-foreground">None</span>
      );
    },
    enableSorting: false,
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
      const policy = row.original;

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
                    onEdit(policy);
                  }}
                >
                  <Pencil className="h-4 w-4" />
                  Edit policy
                </DropdownMenuItem>
              )}
              {canDelete && (
                <>
                  {canUpdate && <DropdownMenuSeparator />}
                  <DropdownMenuItem
                    variant="destructive"
                    onClick={(e) => {
                      e.stopPropagation();
                      onDelete(policy);
                    }}
                  >
                    <Trash2 className="h-4 w-4" />
                    Delete policy
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
