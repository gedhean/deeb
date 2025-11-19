import type { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '@/components/data-table/data-table-column-header';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { RoleBadge } from '@/components/RoleBadge';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  MoreHorizontal,
  Pencil,
  Trash2,
  Eye,
  Shield,
  User as UserIcon,
  UserCog,
} from 'lucide-react';
import type { User } from '@/lib/api/types';
import type { OrganizationRole } from '@/lib/rbac';
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

export interface UsersColumnsProps {
  onView: (user: User) => void;
  onEdit: (user: User) => void;
  onDelete: (user: User) => void;
  canUpdate: boolean;
  canDelete: boolean;
}

export const getUsersColumns = ({
  onView,
  onEdit,
  onDelete,
  canUpdate,
  canDelete,
}: UsersColumnsProps): ColumnDef<User>[] => [
  {
    accessorKey: 'email',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Email" />,
    cell: ({ row }) => <span className="font-medium">{row.getValue('email')}</span>,
    meta: {
      filterConfig: {
        variant: 'search',
        placeholder: 'Search emails...',
      },
      displayName: 'Email',
    } as ColumnMeta,
  },
  {
    accessorKey: 'organization_role',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Role" />,
    cell: ({ row }) => (
      <RoleBadge role={row.getValue('organization_role') as OrganizationRole} showTooltip={true} />
    ),
    enableSorting: false,
    meta: {
      filterConfig: {
        variant: 'multi-select',
        title: 'Role',
        options: [
          { label: 'Organization Admin', value: 'OrganizationAdmin', icon: Shield },
          { label: 'Organization DBA', value: 'OrganizationDBA', icon: UserCog },
          { label: 'Organization Developer', value: 'OrganizationDeveloper', icon: UserIcon },
        ],
      },
      displayName: 'Role',
    } as ColumnMeta,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: 'is_active',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Status" />,
    cell: ({ row }) => (
      <Badge variant={row.getValue('is_active') ? 'default' : 'secondary'}>
        {row.getValue('is_active') ? 'Active' : 'Inactive'}
      </Badge>
    ),
    meta: {
      filterConfig: {
        variant: 'select',
        title: 'Status',
        options: [
          { label: 'Active', value: 'true' },
          { label: 'Inactive', value: 'false' },
        ],
      },
      displayName: 'Status',
    } as ColumnMeta,
    filterFn: (row, id, value) => {
      return String(row.getValue(id)) === value;
    },
  },
  {
    accessorKey: 'last_login_at',
    header: ({ column }) => <DataTableColumnHeader column={column} title="Last Login" />,
    cell: ({ row }) => {
      const lastLogin = row.getValue('last_login_at') as string | null;
      return lastLogin ? formatDate(lastLogin) : 'Never';
    },
    meta: {
      displayName: 'Last Login',
    } as ColumnMeta,
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
      const user = row.original;

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
                  onView(user);
                }}
              >
                <Eye className="h-4 w-4" />
                View user
              </DropdownMenuItem>
              {canUpdate && (
                <DropdownMenuItem
                  onClick={(e) => {
                    e.stopPropagation();
                    onEdit(user);
                  }}
                >
                  <Pencil className="h-4 w-4" />
                  Edit user
                </DropdownMenuItem>
              )}
              {canDelete && (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    variant="destructive"
                    onClick={(e) => {
                      e.stopPropagation();
                      onDelete(user);
                    }}
                  >
                    <Trash2 className="h-4 w-4" />
                    Deactivate user
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
