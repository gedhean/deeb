import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { usersService } from '@/lib/api/services/users';
import type { User, UpdateUserRequest } from '@/lib/api/types';
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
import { UserForm, type UserFormData, type UserFormErrors } from '@/components/user-form';

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

  // Check if current user is admin
  const isAdmin = currentUser?.organization_role === 'OrganizationAdmin';

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
        organization_role: userData.organization_role as
          | 'OrganizationAdmin'
          | 'OrganizationDBA'
          | 'OrganizationDeveloper',
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

  // Validate edit form
  const validateEditForm = (): boolean => {
    const errors: { organization_role?: string } = {};
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

  // Open edit form
  const openEditForm = () => {
    if (!user) return;
    setEditFormData({
      display_name: user.display_name || '',
      organization_role: user.organization_role as
        | 'OrganizationAdmin'
        | 'OrganizationDBA'
        | 'OrganizationDeveloper',
      is_active: user.is_active,
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

  // Render access denied
  if (!isAdmin) {
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
              This page requires Organization Admin role.
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
          {user && (
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
            {/* User Information */}
            <Card>
              <CardHeader>
                <CardTitle>User Information</CardTitle>
                <CardDescription>Basic user account details</CardDescription>
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
                <p className="text-muted-foreground">User not found</p>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Edit User Sheet */}
        <Sheet open={isEditOpen} onOpenChange={setIsEditOpen}>
          <SheetContent className="w-full sm:w-[540px] overflow-y-auto">
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
