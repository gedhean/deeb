import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { useUsers, useCreateUser, useUpdateUser, useDeleteUser } from '@/hooks/api/useUsersSWR';
import type { CreateUserRequest, User, UpdateUserRequest } from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Plus, Pencil, Trash2, AlertTriangle, Users } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { RoleBadge } from '@/components/RoleBadge';
import { UserForm, type UserFormData, type UserFormErrors } from '@/components/user-form';
import {
  hasOrganizationPermission,
  OrganizationPermission,
  type OrganizationRole,
} from '@/lib/rbac';

export const UsersPage: React.FC = () => {
  const navigate = useNavigate();
  const { user } = useAuth();
  const { users, isLoading, error: usersError } = useUsers();
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);

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
  const openCreateForm = () => {
    setFormData({
      email: '',
      password: '',
      display_name: '',
      organization_role: 'OrganizationDeveloper',
    });
    setFormErrors({});
    setIsCreateOpen(true);
  };

  // Open edit form
  const openEditForm = (user: User) => {
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
  };

  // Open delete confirmation
  const openDeleteConfirmation = (user: User) => {
    setSelectedUser(user);
    setIsDeleteOpen(true);
  };

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
        {isLoading ? (
          <Card>
            <CardContent className="pt-6">
              <div className="space-y-4">
                <Skeleton className="h-9 w-full" />
                <div className="space-y-2">
                  <Skeleton className="h-6 w-32" />
                  <Skeleton className="h-4 w-48" />
                  <Skeleton className="h-4 w-32" />
                </div>
              </div>
            </CardContent>
          </Card>
        ) : users.length === 0 ? (
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
              <div className="overflow-x-auto -mx-6 px-6">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Email</TableHead>
                      <TableHead>Role</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Last Login</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {users.map((user) => (
                      <TableRow
                        key={user.id}
                        onClick={() => navigate(`/users/${user.id}`)}
                        className="cursor-pointer hover:bg-muted/50 transition-colors"
                      >
                        <TableCell className="font-medium">{user.email}</TableCell>
                        <TableCell>
                          <RoleBadge
                            role={user.organization_role as OrganizationRole}
                            showTooltip={true}
                          />
                        </TableCell>
                        <TableCell>
                          <Badge variant={user.is_active ? 'default' : 'secondary'}>
                            {user.is_active ? 'Active' : 'Inactive'}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          {user.last_login_at ? formatDate(user.last_login_at) : 'Never'}
                        </TableCell>
                        <TableCell>{formatDate(user.created_at)}</TableCell>
                        <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                          <div className="flex items-center justify-end gap-2">
                            {canUpdateUsers && (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  openEditForm(user);
                                }}
                                className="h-11 w-11 p-0 touch-manipulation"
                                aria-label="Edit user"
                              >
                                <Pencil className="h-4 w-4" />
                              </Button>
                            )}
                            {canDeleteUsers && (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  openDeleteConfirmation(user);
                                }}
                                className="h-11 w-11 p-0 text-destructive hover:text-destructive touch-manipulation"
                                aria-label="Delete user"
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            )}
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
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
