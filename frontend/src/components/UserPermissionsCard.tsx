import React from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Shield } from 'lucide-react';
import { getOrganizationPermissions, type OrganizationRole } from '@/lib/rbac';

interface UserPermissionsCardProps {
  role: OrganizationRole;
  title?: string;
  description?: string;
}

/**
 * Shared component for displaying user permissions based on organization role
 * Used in both ProfilePage and UserDetailPage
 */
export const UserPermissionsCard: React.FC<UserPermissionsCardProps> = ({
  role,
  title = 'Organization Permissions',
  description,
}) => {
  // Format role display name
  const formatRoleName = (roleStr: string) => {
    switch (roleStr) {
      case 'OrganizationAdmin':
        return 'Admin';
      case 'OrganizationDBA':
        return 'DBA';
      case 'OrganizationDeveloper':
        return 'Developer';
      default:
        return roleStr;
    }
  };

  // Format permission display name
  const formatPermissionName = (permission: string): string => {
    if (permission === '*') {
      return 'All Permissions';
    }
    return permission
      .split('.')
      .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
      .join(' ');
  };

  const defaultDescription = `Your ${formatRoleName(role)} role grants you the following permissions`;

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description || defaultDescription}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
          {getOrganizationPermissions(role).map((permission) => (
            <div
              key={permission}
              className="flex items-center gap-2 p-3 rounded-lg border bg-muted/30"
            >
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10">
                <Shield className="h-4 w-4 text-primary" />
              </div>
              <span className="text-sm font-medium">{formatPermissionName(permission)}</span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
};
