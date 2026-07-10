import React, { useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { subDays } from 'date-fns';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { useUsers, useCreateUser, useUpdateUser, useDeleteUser } from '@/hooks/api/useUsersSWR';
import {
  useServerTableFilters,
  type ServerTableFilterMapping,
} from '@/hooks/useServerTableFilters';
import type { CreateUserRequest, User, UpdateUserRequest } from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { DataTable, DataTableToolbar } from '@/components/data-table';
import type { FilterDefinition } from '@/components/data-table/types';
import { getUsersColumns } from './users/columns';
import { Plus, AlertTriangle, Users, Shield, UserCog, User as UserIcon } from 'lucide-react';
import { RoleBadge } from '@/components/RoleBadge';
import { UserForm, type UserFormData, type UserFormErrors } from '@/components/user-form';
import {
  hasOrganizationPermission,
  OrganizationPermission,
  type OrganizationRole,
} from '@/lib/rbac';

const USERS_FILTER_MAPPING: ServerTableFilterMapping = {
  searchColumnIds: ['email'],
  equals: { organization_role: 'organization_role', is_active: 'is_active' },
  dateRanges: { created_at: ['created_after', 'created_before'] },
};

// Filter definitions are static (role / status are enums; no reference-backed
// filters), so they live at module level.
const USERS_FILTER_DEFINITIONS: FilterDefinition[] = [
  {
    id: 'email',
    type: 'search',
    title: 'Email',
    placeholder: 'Search emails...',
    defaultActive: true,
  },
  {
    id: 'organization_role',
    type: 'faceted',
    title: 'Role',
    options: [
      { label: 'Organization Admin', value: 'OrganizationAdmin', icon: Shield },
      { label: 'Organization DBA', value: 'OrganizationDBA', icon: UserCog },
      { label: 'Organization Developer', value: 'OrganizationDeveloper', icon: UserIcon },
    ],
  },
  {
    id: 'is_active',
    type: 'select',
    title: 'Status',
    options: [
      { label: 'Active', value: 'true' },
      { label: 'Inactive', value: 'false' },
    ],
  },
  {
    id: 'created_at',
    type: 'dateRange',
    title: 'Created',
    presets: [
      { label: 'Last 7 days', getValue: () => [subDays(new Date(), 7), new Date()] },
      { label: 'Last 30 days', getValue: () => [subDays(new Date(), 30), new Date()] },
      { label: 'Last 90 days', getValue: () => [subDays(new Date(), 90), new Date()] },
    ],
  },
];

export const UsersPage: React.FC = () => {
  const navigate = useNavigate();
  const { user } = useAuth();
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);

  // Server-side pagination + filtering (state persisted in the URL).
  const {
    params,
    pageIndex,
    pageSize,
    setPageIndex,
    setPageSize,
    onFiltersChange,
    initialColumnFilters,
  } = useServerTableFilters({
    mapping: USERS_FILTER_MAPPING,
    filterDefinitions: USERS_FILTER_DEFINITIONS,
    initialPageSize: 50,
  });

  const { users, total, isLoading, error: usersError } = useUsers(params);

  // Use mutation hooks
  const { createUser, isLoading: isCreating } = useCreateUser();
  const { updateUser, isLoading: isUpdating } = useUpdateUser();
  const { deleteUser, isLoading: isDeleting } = useDeleteUser();

  // Form state
  const [formData, setFormData] = useState<UserFormData>({
    email: '',
    password: '',
    display_name: '',
    organization_role: 'OrganizationDeveloper',
  });
  const [editFormData, setEditFormData] = useState<UserFormData>({
    display_name: '',
    organization_role: 'OrganizationDeveloper',
    is_active: true,
  });
  const [formErrors, setFormErrors] = useState<UserFormErrors>({});

  // Check permissions
  const canReadUsers = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.UsersRead,
  );
  const canCreateUsers = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.UsersCreate,
  );
  const canUpdateUsers = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.UsersUpdate,
  );
  const canDeleteUsers = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.UsersDelete,
  );

  // Compute error message based on permissions and API errors
  const error = !canReadUsers ? 'Access denied. User read permissions required.' : usersError || '';

  // Validate form
  const validateForm = (): boolean => {
    const errors: { email?: string; password?: string; organization_role?: string } = {};

    if (!formData.email?.trim()) {
      errors.email = 'Email is required';
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email.trim())) {
      errors.email = 'Invalid email format';
    }

    if (!formData.password?.trim()) {
      errors.password = 'Password is required';
    } else if (formData.password.length < 8) {
      errors.password = 'Password must be at least 8 characters';
    }

    if (!formData.organization_role) {
      errors.organization_role = 'Organization role is required';
    }

    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  // Validate edit form (password is optional for edit)
  const validateEditForm = (): boolean => {
    const errors: { organization_role?: string } = {};

    // Email is not editable, so no validation needed
    // Organization role is always set, so no validation needed either

    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  // Handle create
  const handleCreate = async () => {
    if (!validateForm()) {
      return;
    }

    const createData: CreateUserRequest = {
      email: formData.email?.trim().toLowerCase() || '',
      password: formData.password || '',
      display_name: formData.display_name?.trim() || undefined,
      organization_role: formData.organization_role,
    };

    const result = await createUser(createData);
    if (result) {
      setIsCreateOpen(false);
      setFormData({
        email: '',
        password: '',
        display_name: '',
        organization_role: 'OrganizationDeveloper',
      });
      setFormErrors({});
    }
  };

  // Handle edit
  const handleEdit = async () => {
    if (!selectedUser || !validateEditForm()) {
      return;
    }

    const updateData: UpdateUserRequest = {};

    if (editFormData.password && editFormData.password.trim() !== '') {
      updateData.password = editFormData.password;
    }
    if (editFormData.display_name !== (selectedUser.display_name || '')) {
      updateData.display_name = editFormData.display_name?.trim() || undefined;
    }
    if (editFormData.organization_role !== selectedUser.organization_role) {
      updateData.organization_role = editFormData.organization_role;
    }
    if (editFormData.is_active !== selectedUser.is_active) {
      updateData.is_active = editFormData.is_active;
    }

    const result = await updateUser(selectedUser.id, updateData);
    if (result) {
      setIsEditOpen(false);
      setSelectedUser(null);
      setEditFormData({
        display_name: '',
        password: '',
        organization_role: 'OrganizationDeveloper',
        is_active: true,
      });
      setFormErrors({});
    }
  };

  // Handle delete (deactivate)
  const handleDelete = async () => {
    if (!selectedUser) {
      return;
    }

    const success = await deleteUser(selectedUser.id);
    if (success) {
      setIsDeleteOpen(false);
      setSelectedUser(null);
    }
  };

  // Open create form
  const openCreateForm = useCallback(() => {
    setFormData({
      email: '',
      password: '',
      display_name: '',
      organization_role: 'OrganizationDeveloper',
    });
    setFormErrors({});
    setIsCreateOpen(true);
  }, []);

  // Open edit form
  const openEditForm = useCallback((user: User) => {
    setSelectedUser(user);
    setEditFormData({
      display_name: user.display_name || '',
      password: '',
      organization_role: user.organization_role as
        | 'OrganizationAdmin'
        | 'OrganizationDBA'
        | 'OrganizationDeveloper',
      is_active: user.is_active,
    });
    setFormErrors({});
    setIsEditOpen(true);
  }, []);

  // Open delete confirmation
  const openDeleteConfirmation = useCallback((user: User) => {
    setSelectedUser(user);
    setIsDeleteOpen(true);
  }, []);

  // Get columns with action handlers
  const columns = getUsersColumns({
    onView: (user) => navigate(`/users/${user.id}`),
    onEdit: openEditForm,
    onDelete: openDeleteConfirmation,
    canUpdate: canUpdateUsers,
    canDelete: canDeleteUsers,
  });

  // Render access denied
  if (!canReadUsers) {
    return (
      <DefaultLayout breadcrumbs={[{ label: 'Users', isCurrentPage: true }]}>
        <Card>
          <CardHeader>
            <CardTitle>Access Denied</CardTitle>
            <CardDescription>You do not have permission to access this page.</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              This page requires the users.read permission.
            </p>
          </CardContent>
        </Card>
      </DefaultLayout>
    );
  }

  return (
    <DefaultLayout breadcrumbs={[{ label: 'Users', isCurrentPage: true }]}>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Users</h1>
            <p className="text-muted-foreground mt-1">Manage users and their organization roles</p>
          </div>
          {canCreateUsers && (
            <Button onClick={openCreateForm}>
              <Plus className="mr-2 h-4 w-4" />
              Create User
            </Button>
          )}
        </div>

        {/* Error message */}
        {error && (
          <Card className="border-destructive">
            <CardContent className="pt-6">
              <div className="flex items-center gap-2 text-destructive">
                <AlertTriangle className="h-4 w-4" />
                <span className="text-sm">{error}</span>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Users list */}
        {!isLoading && users.length === 0 ? (
          <Card>
            <CardContent className="pt-6">
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <Users className="h-12 w-12 text-muted-foreground" />
                  </EmptyMedia>
                  <EmptyTitle>No Users Yet</EmptyTitle>
                  <EmptyDescription>
                    Get started by creating your first user account with email, password, and
                    organization role.
                  </EmptyDescription>
                </EmptyHeader>
                <EmptyContent>
                  {canCreateUsers && (
                    <Button onClick={openCreateForm}>
                      <Plus className="mr-2 h-4 w-4" />
                      Create User
                    </Button>
                  )}
                </EmptyContent>
              </Empty>
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardContent className="pt-6">
              <DataTable
                columns={columns}
                data={users}
                isLoading={isLoading}
                initialColumnFilters={initialColumnFilters}
                emptyMessage="No users found."
                pagination={{
                  pageIndex,
                  pageSize,
                  totalCount: total,
                  onPageChange: setPageIndex,
                  onPageSizeChange: setPageSize,
                  pageSizeOptions: [10, 20, 50, 100],
                  manualFiltering: true,
                }}
                onRowClick={(user) => navigate(`/users/${user.id}`)}
                onFiltersChange={onFiltersChange}
              >
                {(table) => <DataTableToolbar table={table} filters={USERS_FILTER_DEFINITIONS} />}
              </DataTable>
            </CardContent>
          </Card>
        )}

        {/* Create User Sheet */}
        <Sheet open={isCreateOpen} onOpenChange={setIsCreateOpen}>
          <SheetContent className="w-full sm:w-[540px] overflow-y-auto min-w-[600px]">
            <SheetHeader>
              <SheetTitle>Create User</SheetTitle>
              <SheetDescription>
                Create a new user account with email, password, and organization role.
              </SheetDescription>
            </SheetHeader>
            <UserForm
              mode="create"
              formData={formData}
              formErrors={formErrors}
              onChange={(data) => setFormData({ ...formData, ...data })}
            />
            <SheetFooter>
              <Button variant="outline" onClick={() => setIsCreateOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleCreate} disabled={isCreating}>
                {isCreating ? 'Creating...' : 'Create User'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>

        {/* Edit User Sheet */}
        <Sheet open={isEditOpen} onOpenChange={setIsEditOpen}>
          <SheetContent className="w-full sm:w-[540px] overflow-y-auto min-w-[600px]">
            <SheetHeader>
              <SheetTitle>Edit User</SheetTitle>
              <SheetDescription>Update user organization role and active status.</SheetDescription>
            </SheetHeader>
            <UserForm
              mode="edit"
              formData={{ ...editFormData, email: selectedUser?.email || '' }}
              formErrors={formErrors}
              onChange={(data) => setEditFormData({ ...editFormData, ...data })}
            />
            <SheetFooter>
              <Button variant="outline" onClick={() => setIsEditOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleEdit} disabled={isUpdating}>
                {isUpdating ? 'Saving...' : 'Save Changes'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>

        {/* Delete Confirmation Sheet */}
        <Sheet open={isDeleteOpen} onOpenChange={setIsDeleteOpen}>
          <SheetContent className="w-full sm:w-[540px]">
            <SheetHeader>
              <SheetTitle>Deactivate User</SheetTitle>
              <SheetDescription>
                Are you sure you want to deactivate this user? They will no longer be able to access
                the system.
              </SheetDescription>
            </SheetHeader>
            <div className="py-4">
              {selectedUser && (
                <div className="space-y-2">
                  <p className="text-sm font-medium">{selectedUser.email}</p>
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <span>Role:</span>
                    <RoleBadge
                      role={selectedUser.organization_role as OrganizationRole}
                      showTooltip={true}
                    />
                  </div>
                </div>
              )}
            </div>
            <SheetFooter>
              <Button variant="outline" onClick={() => setIsDeleteOpen(false)}>
                Cancel
              </Button>
              <Button variant="destructive" onClick={handleDelete} disabled={isDeleting}>
                {isDeleting ? 'Deactivating...' : 'Deactivate User'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>
      </div>
    </DefaultLayout>
  );
};
