import type { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '@/components/data-table/data-table-column-header';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { ChangePlanStatusBadge } from '@/components/ChangePlanStatusBadge';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  MoreHorizontal,
  Eye,
  FileEdit,
  CheckCircle2,
  Loader,
  XCircle,
  Ban,
  Archive,
  FilePlus,
} from 'lucide-react';
import type { ChangePlan } from '@/lib/api/types';
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

export interface ChangePlansColumnsProps {
  onView: (changePlan: ChangePlan) => void;
  getProjectName: (projectId: string | null | undefined) => string;
  getDatabaseName: (databaseId: string) => string;
  getAuthorName: (authorId: string) => string;
}

export const getChangePlansColumns = ({
  onView,
  getProjectName,
  getDatabaseName,
  getAuthorName,
}: ChangePlansColumnsProps): ColumnDef<ChangePlan>[] => [
  {
    accessorKey: 'title',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Title" />,
    cell: ({ row }) => {
      const changePlan = row.original;
      return (
        <div className="flex flex-col">
          <span className="font-medium truncate max-w-[200px]">{changePlan.title}</span>
          {changePlan.description && (
            <span className="text-xs text-muted-foreground truncate max-w-[200px]">
              {changePlan.description}
            </span>
          )}
        </div>
      );
    },
    meta: {
      filterConfig: {
        variant: 'search',
        placeholder: 'Search titles...',
      },
      displayName: 'Title',
    } as ColumnMeta,
  },
  {
    accessorKey: 'status',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Status" />,
    cell: ({ row }) => <ChangePlanStatusBadge status={row.getValue('status')} />,
    enableSorting: false,
    meta: {
      filterConfig: {
        variant: 'multi-select',
        title: 'Status',
        options: [
          { label: 'Draft', value: 'Draft', icon: FilePlus },
          { label: 'Open', value: 'Open', icon: FileEdit },
          { label: 'Approved', value: 'Approved', icon: CheckCircle2 },
          { label: 'In Progress', value: 'InProgress', icon: Loader },
          { label: 'Completed', value: 'Completed', icon: CheckCircle2 },
          { label: 'Failed', value: 'Failed', icon: XCircle },
          { label: 'Cancelled', value: 'Cancelled', icon: Ban },
          { label: 'Closed', value: 'Closed', icon: Archive },
        ],
      },
      displayName: 'Status',
    } as ColumnMeta,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: 'project_id',
    id: 'project',
    header: 'Project',
    cell: ({ row }) => {
      const changePlan = row.original;
      return <span className="text-muted-foreground">{getProjectName(changePlan.project_id)}</span>;
    },
    enableSorting: false,
    meta: {
      filterConfig: {
        variant: 'multi-select',
        title: 'Project',
        options: [], // Options will be provided dynamically in the page
      },
      displayName: 'Project',
    } as ColumnMeta,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: 'target_database_id',
    id: 'database',
    header: 'Database',
    cell: ({ row }) => {
      const changePlan = row.original;
      return (
        <span className="text-muted-foreground">
          {getDatabaseName(changePlan.target_database_id)}
        </span>
      );
    },
    enableSorting: false,
    meta: {
      filterConfig: {
        variant: 'multi-select',
        title: 'Database',
        options: [], // Options will be provided dynamically in the page
      },
      displayName: 'Database',
    } as ColumnMeta,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: 'author_id',
    id: 'author',
    header: 'Author',
    cell: ({ row }) => {
      const changePlan = row.original;
      return <span className="text-muted-foreground">{getAuthorName(changePlan.author_id)}</span>;
    },
    enableSorting: false,
    meta: {
      filterConfig: {
        variant: 'multi-select',
        title: 'Author',
        options: [], // Options will be provided dynamically in the page
      },
      displayName: 'Author',
    } as ColumnMeta,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: 'change_type',
    header: 'Change Type',
    cell: ({ row }) => <Badge variant="outline">{row.getValue('change_type')}</Badge>,
    enableSorting: false,
    meta: {
      filterConfig: {
        variant: 'select',
        title: 'Change Type',
        options: [
          { label: 'DDL', value: 'DDL' },
          { label: 'DML', value: 'DML' },
        ],
      },
      displayName: 'Change Type',
    } as ColumnMeta,
    filterFn: (row, id, value) => {
      return String(row.getValue(id)) === value;
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
      const changePlan = row.original;

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
                  onView(changePlan);
                }}
              >
                <Eye className="h-4 w-4" />
                View change plan
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      );
    },
  },
];
