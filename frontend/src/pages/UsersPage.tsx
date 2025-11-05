import React, { useEffect, useState } from 'react';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { usersService } from '@/lib/api/services/users';
import type { CreateUserRequest, User, UpdateUserRequest } from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
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

export const UsersPage: React.FC = () => {
  const { user } = useAuth();
  const [users, setUsers] = useState<User[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Form state
  const [formData, setFormData] = useState<CreateUserRequest>({
    email: '',
    password: '',
    organization_role: 'OrganizationDeveloper',
  });
  const [editFormData, setEditFormData] = useState<{
    organization_role: 'OrganizationAdmin' | 'OrganizationDBA' | 'OrganizationDeveloper';
    is_active: boolean;
  }>({
    organization_role: 'OrganizationDeveloper',
    is_active: true,
  });
  const [formErrors, setFormErrors] = useState<{
    email?: string;
    password?: string;
    organization_role?: string;
  }>({});

  // Check if user is admin
  const isAdmin = user?.organization_role === 'OrganizationAdmin';

  // Load users
  useEffect(() => {
    if (!isAdmin) {
      setError('Access denied. Admin access required.');
      setIsLoading(false);
      return;
    }

    loadUsers();
  }, [isAdmin]);

  const loadUsers = async () => {
    try {
      setIsLoading(true);
      setError('');
      const response = await usersService.listUsers({ limit: 100 });
      setUsers(response.items);
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to load users. Please try again.');
      }
      console.error('Failed to load users:', err);
    } finally {
      setIsLoading(false);
    }
  };

  // Validate form
  const validateForm = (): boolean => {
    const errors: { email?: string; password?: string; organization_role?: string } = {};

    if (!formData.email.trim()) {
      errors.email = 'Email is required';
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email.trim())) {
      errors.email = 'Invalid email format';
    }

    if (!formData.password.trim()) {
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

    try {
      setIsSubmitting(true);
      const createData: CreateUserRequest = {
        email: formData.email.trim().toLowerCase(),
        password: formData.password,
        organization_role: formData.organization_role,
      };
      await usersService.createUser(createData);
      setIsCreateOpen(false);
      setFormData({ email: '', password: '', organization_role: 'OrganizationDeveloper' });
      setFormErrors({});
      await loadUsers();
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to create user. Please try again.');
      }
      console.error('Failed to create user:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Handle edit
  const handleEdit = async () => {
    if (!selectedUser || !validateEditForm()) {
      return;
    }

    try {
      setIsSubmitting(true);
      const updateData: UpdateUserRequest = {};

      if (editFormData.organization_role !== selectedUser.organization_role) {
        updateData.organization_role = editFormData.organization_role;
      }
      if (editFormData.is_active !== selectedUser.is_active) {
        updateData.is_active = editFormData.is_active;
      }

      await usersService.updateUser(selectedUser.id, updateData);
      setIsEditOpen(false);
      setSelectedUser(null);
      setEditFormData({ organization_role: 'OrganizationDeveloper', is_active: true });
      setFormErrors({});
      await loadUsers();
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to update user. Please try again.');
      }
      console.error('Failed to update user:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Handle delete (deactivate)
  const handleDelete = async () => {
    if (!selectedUser) {
      return;
    }

    try {
      setIsSubmitting(true);
      await usersService.deleteUser(selectedUser.id);
      setIsDeleteOpen(false);
      setSelectedUser(null);
      await loadUsers();
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to deactivate user. Please try again.');
      }
      console.error('Failed to deactivate user:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Open create form
  const openCreateForm = () => {
    setFormData({ email: '', password: '', organization_role: 'OrganizationDeveloper' });
    setFormErrors({});
    setIsCreateOpen(true);
  };

  // Open edit form
  const openEditForm = (user: User) => {
    setSelectedUser(user);
    setEditFormData({
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

  // Get role badge variant
  const getRoleBadgeVariant = (role: string) => {
    switch (role) {
      case 'OrganizationAdmin':
        return 'default';
      case 'OrganizationDBA':
        return 'secondary';
      case 'OrganizationDeveloper':
        return 'outline';
      default:
        return 'outline';
    }
  };

  // Format role display name
  const formatRoleName = (role: string) => {
    switch (role) {
      case 'OrganizationAdmin':
        return 'Admin';
      case 'OrganizationDBA':
        return 'DBA';
      case 'OrganizationDeveloper':
        return 'Developer';
      default:
        return role;
    }
  };

  // Render access denied
  if (!isAdmin) {
    return (
      <DefaultLayout breadcrumbs={[{ label: 'Users', isCurrentPage: true }]}>
        <Card>
          <CardHeader>
            <CardTitle>Access Denied</CardTitle>
            <CardDescription>You do not have permission to access this page.</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              This page requires Organization Admin role.
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
          <Button onClick={openCreateForm}>
            <Plus className="mr-2 h-4 w-4" />
            Create User
          </Button>
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
                  <Button onClick={openCreateForm}>
                    <Plus className="mr-2 h-4 w-4" />
                    Create User
                  </Button>
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
                      <TableRow key={user.id}>
                        <TableCell className="font-medium">{user.email}</TableCell>
                        <TableCell>
                          <Badge variant={getRoleBadgeVariant(user.organization_role)}>
                            {formatRoleName(user.organization_role)}
                          </Badge>
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
                        <TableCell className="text-right">
                          <div className="flex items-center justify-end gap-2">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => openEditForm(user)}
                              className="h-11 w-11 p-0 touch-manipulation"
                              aria-label="Edit user"
                            >
                              <Pencil className="h-4 w-4" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => openDeleteConfirmation(user)}
                              className="h-11 w-11 p-0 text-destructive hover:text-destructive touch-manipulation"
                              aria-label="Delete user"
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
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
          <SheetContent className="w-full sm:w-[540px] overflow-y-auto">
            <SheetHeader>
              <SheetTitle>Create User</SheetTitle>
              <SheetDescription>
                Create a new user account with email, password, and organization role.
              </SheetDescription>
            </SheetHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <label htmlFor="create-email" className="text-sm font-medium">
                  Email *
                </label>
                <Input
                  id="create-email"
                  type="email"
                  value={formData.email}
                  onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                  placeholder="user@example.com"
                  aria-invalid={!!formErrors.email}
                />
                {formErrors.email && <p className="text-sm text-destructive">{formErrors.email}</p>}
              </div>
              <div className="space-y-2">
                <label htmlFor="create-password" className="text-sm font-medium">
                  Password *
                </label>
                <Input
                  id="create-password"
                  type="password"
                  value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  placeholder="Enter password (min 8 characters)"
                  aria-invalid={!!formErrors.password}
                />
                {formErrors.password && (
                  <p className="text-sm text-destructive">{formErrors.password}</p>
                )}
              </div>
              <div className="space-y-2">
                <label htmlFor="create-role" className="text-sm font-medium">
                  Organization Role *
                </label>
                <select
                  id="create-role"
                  value={formData.organization_role}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      organization_role: e.target.value as
                        | 'OrganizationAdmin'
                        | 'OrganizationDBA'
                        | 'OrganizationDeveloper',
                    })
                  }
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                  aria-invalid={!!formErrors.organization_role}
                >
                  <option value="OrganizationDeveloper">Developer</option>
                  <option value="OrganizationDBA">DBA</option>
                  <option value="OrganizationAdmin">Admin</option>
                </select>
                {formErrors.organization_role && (
                  <p className="text-sm text-destructive">{formErrors.organization_role}</p>
                )}
              </div>
            </div>
            <SheetFooter>
              <Button variant="outline" onClick={() => setIsCreateOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleCreate} disabled={isSubmitting}>
                {isSubmitting ? 'Creating...' : 'Create User'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>

        {/* Edit User Sheet */}
        <Sheet open={isEditOpen} onOpenChange={setIsEditOpen}>
          <SheetContent className="w-full sm:w-[540px] overflow-y-auto">
            <SheetHeader>
              <SheetTitle>Edit User</SheetTitle>
              <SheetDescription>Update user organization role and active status.</SheetDescription>
            </SheetHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <label htmlFor="edit-email" className="text-sm font-medium">
                  Email
                </label>
                <Input
                  id="edit-email"
                  type="email"
                  value={selectedUser?.email || ''}
                  disabled
                  className="bg-muted"
                />
                <p className="text-xs text-muted-foreground">Email cannot be changed</p>
              </div>
              <div className="space-y-2">
                <label htmlFor="edit-role" className="text-sm font-medium">
                  Organization Role *
                </label>
                <select
                  id="edit-role"
                  value={editFormData.organization_role}
                  onChange={(e) =>
                    setEditFormData({
                      ...editFormData,
                      organization_role: e.target.value as
                        | 'OrganizationAdmin'
                        | 'OrganizationDBA'
                        | 'OrganizationDeveloper',
                    })
                  }
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                  aria-invalid={!!formErrors.organization_role}
                >
                  <option value="OrganizationDeveloper">Developer</option>
                  <option value="OrganizationDBA">DBA</option>
                  <option value="OrganizationAdmin">Admin</option>
                </select>
                {formErrors.organization_role && (
                  <p className="text-sm text-destructive">{formErrors.organization_role}</p>
                )}
              </div>
              <div className="space-y-2">
                <label htmlFor="edit-active" className="text-sm font-medium">
                  Status
                </label>
                <select
                  id="edit-active"
                  value={editFormData.is_active ? 'true' : 'false'}
                  onChange={(e) => {
                    setEditFormData({
                      ...editFormData,
                      is_active: e.target.value === 'true',
                    });
                  }}
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="true">Active</option>
                  <option value="false">Inactive</option>
                </select>
              </div>
            </div>
            <SheetFooter>
              <Button variant="outline" onClick={() => setIsEditOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleEdit} disabled={isSubmitting}>
                {isSubmitting ? 'Saving...' : 'Save Changes'}
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
                  <p className="text-sm text-muted-foreground">
                    Role: {formatRoleName(selectedUser.organization_role)}
                  </p>
                </div>
              )}
            </div>
            <SheetFooter>
              <Button variant="outline" onClick={() => setIsDeleteOpen(false)}>
                Cancel
              </Button>
              <Button variant="destructive" onClick={handleDelete} disabled={isSubmitting}>
                {isSubmitting ? 'Deactivating...' : 'Deactivate User'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>
      </div>
    </DefaultLayout>
  );
};
