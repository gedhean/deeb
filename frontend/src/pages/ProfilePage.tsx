import React, { useEffect, useState } from 'react';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { usersService } from '@/lib/api/services/users';
import type { UpdateUserRequest } from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { AlertTriangle, Pencil, Calendar, Mail, Shield, User as UserIcon } from 'lucide-react';
import { showSuccessToast, handleApiError } from '@/lib/errorHandling';
import { Input } from '@/components/ui/input';
import type { UserFormData, UserFormErrors } from '@/components/user-form';

export const ProfilePage: React.FC = () => {
  const { user: currentUser } = useAuth();
  const [user, setUser] = useState(currentUser);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [editFormData, setEditFormData] = useState<UserFormData>({
    display_name: '',
    password: '',
    organization_role: 'OrganizationDeveloper',
  });
  const [formErrors, setFormErrors] = useState<UserFormErrors>({});

  useEffect(() => {
    if (currentUser) {
      setUser(currentUser);
    }
  }, [currentUser]);

  useEffect(() => {
    if (currentUser) {
      loadUser();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentUser]);

  const loadUser = async () => {
    if (!currentUser) return;

    try {
      setIsLoading(true);
      setError('');
      const userData = await usersService.getUser(currentUser.id);
      setUser(userData);
      setEditFormData({
        display_name: userData.display_name || '',
        password: '',
        organization_role: userData.organization_role as
          | 'OrganizationAdmin'
          | 'OrganizationDBA'
          | 'OrganizationDeveloper',
      });
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to load profile. Please try again.');
      }
      console.error('Failed to load profile:', err);
    } finally {
      setIsLoading(false);
    }
  };

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

    try {
      setIsSubmitting(true);
      const updateData: UpdateUserRequest = {};

      if (editFormData.password && editFormData.password.trim() !== '') {
        updateData.password = editFormData.password;
      }
      if (editFormData.display_name !== (user.display_name || '')) {
        updateData.display_name = editFormData.display_name?.trim() || undefined;
      }

      await usersService.updateUser(user.id, updateData);
      setIsEditOpen(false);
      setFormErrors({});
      await loadUser();
      showSuccessToast('Profile updated successfully');

      // Reload current user data in AuthContext
      // The AuthContext will pick up the changes on next getCurrentUser call
      window.location.reload();
    } catch (err) {
      handleApiError(err, 'Failed to update profile');
      console.error('Failed to update profile:', err);
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
      organization_role: user.organization_role as
        | 'OrganizationAdmin'
        | 'OrganizationDBA'
        | 'OrganizationDeveloper',
    });
    setFormErrors({});
    setIsEditOpen(true);
  };

  // Format date
  const formatDate = (dateString: string): string => {
    try {
      return new Date(dateString).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
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
            {/* User Information */}
            <Card>
              <CardHeader>
                <CardTitle>Profile Information</CardTitle>
                <CardDescription>Your account details</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                      <Mail className="h-4 w-4" />
                      Email
                    </div>
                    <p className="text-base">{user.email}</p>
                  </div>

                  {user.display_name && (
                    <div className="space-y-2">
                      <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                        <UserIcon className="h-4 w-4" />
                        Display Name
                      </div>
                      <p className="text-base">{user.display_name}</p>
                    </div>
                  )}

                  <div className="space-y-2">
                    <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                      <Shield className="h-4 w-4" />
                      Organization Role
                    </div>
                    <Badge variant={getRoleBadgeVariant(user.organization_role)}>
                      {formatRoleName(user.organization_role)}
                    </Badge>
                  </div>

                  <div className="space-y-2">
                    <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                      Status
                    </div>
                    <Badge variant={user.is_active ? 'default' : 'secondary'}>
                      {user.is_active ? 'Active' : 'Inactive'}
                    </Badge>
                  </div>

                  <div className="space-y-2">
                    <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                      <Calendar className="h-4 w-4" />
                      Created At
                    </div>
                    <p className="text-base">{formatDate(user.created_at)}</p>
                  </div>

                  {user.last_login_at && (
                    <div className="space-y-2">
                      <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                        <Calendar className="h-4 w-4" />
                        Last Login
                      </div>
                      <p className="text-base">{formatDate(user.last_login_at)}</p>
                    </div>
                  )}

                  <div className="space-y-2">
                    <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                      <Calendar className="h-4 w-4" />
                      Updated At
                    </div>
                    <p className="text-base">{formatDate(user.updated_at)}</p>
                  </div>
                </div>
              </CardContent>
            </Card>
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
