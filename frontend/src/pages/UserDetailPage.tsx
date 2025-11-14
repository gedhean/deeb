import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { usersService } from '@/lib/api/services/users';
import type { User, UpdateUserRequest, OrganizationRole } from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { UserInfoCard } from '@/components/UserInfoCard';
import { UserPermissionsCard } from '@/components/UserPermissionsCard';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { AlertTriangle, Pencil } from 'lucide-react';
import { UserForm, type UserFormData, type UserFormErrors } from '@/components/user-form';
import { showSuccessToast, handleApiError } from '@/lib/errorHandling';
import { hasOrganizationPermission, OrganizationPermission } from '@/lib/rbac';

export const UserDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { user: currentUser } = useAuth();
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [editFormData, setEditFormData] = useState<UserFormData>({
    display_name: '',
    organization_role: 'OrganizationDeveloper',
    is_active: true,
  });
  const [formErrors, setFormErrors] = useState<UserFormErrors>({});

  // Check permissions
  const canReadUsers = hasOrganizationPermission(
    currentUser?.organization_role,
    OrganizationPermission.UsersRead,
  );
  const canUpdateUsers = hasOrganizationPermission(
    currentUser?.organization_role,
    OrganizationPermission.UsersUpdate,
  );

  useEffect(() => {
    if (id) {
      loadUser(id);
    }
  }, [id]);

  const loadUser = async (userId: string) => {
    try {
      setIsLoading(true);
      setError('');
      const userData = await usersService.getUser(userId);
      setUser(userData);
      setEditFormData({
        display_name: userData.display_name || '',
        password: '',
        organization_role: userData.organization_role as OrganizationRole,
        is_active: userData.is_active,
      });
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to load user. Please try again.');
      }
      console.error('Failed to load user:', err);
    } finally {
      setIsLoading(false);
    }
  };

  // Validate edit form (password is optional for edit)
  const validateEditForm = (): boolean => {
    const errors: {
      password?: string;
      organization_role?: string;
    } = {};

    // Validate password if provided
    if (editFormData.password && editFormData.password.trim() !== '') {
      if (editFormData.password.length < 8) {
        errors.password = 'Password must be at least 8 characters';
      }
    }

    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  // Handle edit
  const handleEdit = async () => {
    if (!user || !validateEditForm()) {
      return;
    }

    try {
      setIsSubmitting(true);
      const updateData: UpdateUserRequest = {};

      if (editFormData.password && editFormData.password.trim() !== '') {
        updateData.password = editFormData.password;
      }
      if (editFormData.display_name !== (user.display_name || '')) {
        updateData.display_name = editFormData.display_name?.trim() || undefined;
      }
      if (editFormData.organization_role !== user.organization_role) {
        updateData.organization_role = editFormData.organization_role;
      }
      if (editFormData.is_active !== user.is_active) {
        updateData.is_active = editFormData.is_active;
      }

      await usersService.updateUser(user.id, updateData);
      setIsEditOpen(false);
      setFormErrors({});
      await loadUser(user.id);
      showSuccessToast('User updated successfully');
    } catch (err) {
      handleApiError(err, { fallbackMessage: 'Failed to update user' });
      console.error('Failed to update user:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Open edit form
  const openEditForm = () => {
    if (!user) return;
    setEditFormData({
      display_name: user.display_name || '',
      password: '',
      organization_role: user.organization_role as OrganizationRole,
      is_active: user.is_active,
    });
    setFormErrors({});
    setIsEditOpen(true);
  };

  // Render access denied
  if (!canReadUsers) {
    return (
      <DefaultLayout
        breadcrumbs={[
          { label: 'Users', href: '/users' },
          { label: 'User Details', isCurrentPage: true },
        ]}
      >
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
    <DefaultLayout
      breadcrumbs={[
        { label: 'Users', href: '/users' },
        { label: user?.email || 'User Details', isCurrentPage: true },
      ]}
    >
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">User Details</h1>
            <p className="text-muted-foreground mt-1">View and manage user information</p>
          </div>
          {user && canUpdateUsers && (
            <Button onClick={openEditForm} variant="outline">
              <Pencil className="mr-2 h-4 w-4" />
              Edit User
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

        {/* Loading state */}
        {isLoading ? (
          <Card>
            <CardContent className="pt-6">
              <div className="space-y-4">
                <Skeleton className="h-8 w-64" />
                <div className="space-y-2">
                  <Skeleton className="h-4 w-32" />
                  <Skeleton className="h-4 w-48" />
                  <Skeleton className="h-4 w-40" />
                </div>
              </div>
            </CardContent>
          </Card>
        ) : user ? (
          <>
            {/* User Information Card - Using shared component */}
            <UserInfoCard
              user={user}
              title="User Information"
              description="Basic user account details"
            />

            {/* User Permissions Card - Using shared component */}
            <UserPermissionsCard role={user.organization_role as OrganizationRole} />
          </>
        ) : (
          <Card>
            <CardContent className="pt-6">
              <div className="text-center py-8">
                <p className="text-muted-foreground">User not found</p>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Edit User Sheet */}
        <Sheet open={isEditOpen} onOpenChange={setIsEditOpen}>
          <SheetContent className="w-full sm:w-[540px] overflow-y-auto min-w-[600px]">
            <SheetHeader>
              <SheetTitle>Edit User</SheetTitle>
              <SheetDescription>Update user organization role and active status.</SheetDescription>
            </SheetHeader>
            <UserForm
              mode="edit"
              formData={{ ...editFormData, email: user?.email || '' }}
              formErrors={formErrors}
              onChange={(data) => setEditFormData({ ...editFormData, ...data })}
            />
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
      </div>
    </DefaultLayout>
  );
};
