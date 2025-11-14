import React from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { RoleBadge } from '@/components/RoleBadge';
import { Calendar, Mail, Shield, User as UserIcon } from 'lucide-react';
import type { User } from '@/lib/api/types';
import type { OrganizationRole } from '@/lib/rbac';

interface UserInfoCardProps {
  user: User;
  title?: string;
  description?: string;
}

/**
 * Shared component for displaying user information
 * Used in both ProfilePage and UserDetailPage
 */
export const UserInfoCard: React.FC<UserInfoCardProps> = ({
  user,
  title = 'Profile Information',
  description = 'Your account details',
}) => {
  // Format date helper
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

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-4 md:grid-cols-2">
          {/* Email */}
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
              <Mail className="h-4 w-4" />
              Email
            </div>
            <p className="text-base">{user.email}</p>
          </div>

          {/* Display Name */}
          {user.display_name && (
            <div className="space-y-2">
              <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                <UserIcon className="h-4 w-4" />
                Display Name
              </div>
              <p className="text-base">{user.display_name}</p>
            </div>
          )}

          {/* Organization Role */}
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
              <Shield className="h-4 w-4" />
              Organization Role
            </div>
            <RoleBadge role={user.organization_role as OrganizationRole} showTooltip={true} />
          </div>

          {/* Status */}
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
              Status
            </div>
            <Badge variant={user.is_active ? 'default' : 'secondary'}>
              {user.is_active ? 'Active' : 'Inactive'}
            </Badge>
          </div>

          {/* Created At */}
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
              <Calendar className="h-4 w-4" />
              Created At
            </div>
            <p className="text-base">{formatDate(user.created_at)}</p>
          </div>

          {/* Last Login */}
          {user.last_login_at && (
            <div className="space-y-2">
              <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                <Calendar className="h-4 w-4" />
                Last Login
              </div>
              <p className="text-base">{formatDate(user.last_login_at)}</p>
            </div>
          )}

          {/* Updated At */}
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
  );
};
