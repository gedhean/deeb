import React, { useState } from 'react';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { useUserById, useUpdateUser } from '@/hooks/api/useUsersSWR';
import type { UpdateUserRequest } from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { UserInfoCard } from '@/components/UserInfoCard';
import { UserPermissionsCard } from '@/components/UserPermissionsCard';
import type { OrganizationRole } from '@/lib/rbac';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { AlertTriangle, Pencil } from 'lucide-react';
import { showSuccessToast } from '@/lib/errorHandling';
import { Input } from '@/components/ui/input';
import type { UserFormData, UserFormErrors } from '@/components/user-form';

export const ProfilePage: React.FC = () => {
  const { user: currentUser } = useAuth();
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [editFormData, setEditFormData] = useState<UserFormData>({
    display_name: '',
    password: '',
    organization_role: 'OrganizationDeveloper',
  });
  const [formErrors, setFormErrors] = useState<UserFormErrors>({});

  // Use SWR hooks for data fetching
  const { user, isLoading, error } = useUserById(currentUser?.id);
  const { updateUser, isLoading: isUpdating } = useUpdateUser();

  // Validate edit form (password is optional for edit)
  const validateEditForm = (): boolean => {
    const errors: {
      password?: string;
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

    const updateData: UpdateUserRequest = {};

    if (editFormData.password && editFormData.password.trim() !== '') {
      updateData.password = editFormData.password;
    }
    if (editFormData.display_name !== (user.display_name || '')) {
      updateData.display_name = editFormData.display_name?.trim() || undefined;
    }

    const result = await updateUser(user.id, updateData);
    if (result) {
      setIsEditOpen(false);
      setFormErrors({});
      showSuccessToast('Profile updated successfully');

      // Reload current user data in AuthContext
      // The AuthContext will pick up the changes on next getCurrentUser call
      window.location.reload();
    }
  };

  // Open edit form
  const openEditForm = () => {
    if (!user) return;
    setEditFormData({
      display_name: user.display_name || '',
      password: '',
      organization_role: user.organization_role as
        | 'OrganizationAdmin'
        | 'OrganizationDBA'
        | 'OrganizationDeveloper',
    });
    setFormErrors({});
    setIsEditOpen(true);
  };

  if (!currentUser) {
    return (
      <DefaultLayout breadcrumbs={[{ label: 'Profile', isCurrentPage: true }]}>
        <Card>
          <CardHeader>
            <CardTitle>Access Denied</CardTitle>
            <CardDescription>You must be logged in to view your profile.</CardDescription>
          </CardHeader>
        </Card>
      </DefaultLayout>
    );
  }

  return (
    <DefaultLayout breadcrumbs={[{ label: 'Profile', isCurrentPage: true }]}>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">My Profile</h1>
            <p className="text-muted-foreground mt-1">View and manage your account information</p>
          </div>
          {user && (
            <Button onClick={openEditForm} variant="outline">
              <Pencil className="mr-2 h-4 w-4" />
              Edit Profile
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
            <UserInfoCard user={user} />

            {/* User Permissions Card - Using shared component */}
            <UserPermissionsCard role={user.organization_role as OrganizationRole} />
          </>
        ) : (
          <Card>
            <CardContent className="pt-6">
              <div className="text-center py-8">
                <p className="text-muted-foreground">Failed to load profile</p>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Edit Profile Sheet */}
        <Sheet open={isEditOpen} onOpenChange={setIsEditOpen}>
          <SheetContent className="w-full sm:w-[540px] overflow-y-auto">
            <SheetHeader>
              <SheetTitle>Edit Profile</SheetTitle>
              <SheetDescription>Update your display name and password.</SheetDescription>
            </SheetHeader>
            <div className="space-y-4 py-4">
              {/* Display Name Field */}
              <div className="space-y-2">
                <label htmlFor="profile-display-name" className="text-sm font-medium">
                  Display Name
                </label>
                <Input
                  id="profile-display-name"
                  type="text"
                  value={editFormData.display_name || ''}
                  onChange={(e) =>
                    setEditFormData({ ...editFormData, display_name: e.target.value })
                  }
                  placeholder="John Doe"
                />
              </div>

              {/* Password Field */}
              <div className="space-y-2">
                <label htmlFor="profile-password" className="text-sm font-medium">
                  Password
                </label>
                <Input
                  id="profile-password"
                  type="password"
                  value={editFormData.password || ''}
                  onChange={(e) => setEditFormData({ ...editFormData, password: e.target.value })}
                  placeholder="Leave blank to keep current password"
                  aria-invalid={!!formErrors.password}
                />
                <p className="text-xs text-muted-foreground">
                  Leave blank to keep current password. Enter new password to change it.
                </p>
                {formErrors.password && (
                  <p className="text-sm text-destructive">{formErrors.password}</p>
                )}
              </div>
            </div>
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
      </div>
    </DefaultLayout>
  );
};
