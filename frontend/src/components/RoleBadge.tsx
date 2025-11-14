import React from 'react';
import { Badge } from '@/components/ui/badge';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import type { OrganizationRole, ProjectRole } from '@/lib/rbac';

interface RoleBadgeProps {
  role: OrganizationRole | ProjectRole;
  showTooltip?: boolean;
  size?: 'sm' | 'md' | 'lg';
}

// Organization role display names and descriptions
const ORGANIZATION_ROLE_INFO: Record<
  OrganizationRole,
  { label: string; color: string; description: string }
> = {
  OrganizationAdmin: {
    label: 'Admin',
    color: 'bg-red-500/10 text-red-700 hover:bg-red-500/20 dark:text-red-400',
    description:
      'Full system access. Can manage all projects, databases, users, environments, and policies.',
  },
  OrganizationDBA: {
    label: 'DBA',
    color: 'bg-orange-500/10 text-orange-700 hover:bg-orange-500/20 dark:text-orange-400',
    description:
      'Database administrator. Can manage databases, database instances, create projects, and manage change plans.',
  },
  OrganizationDeveloper: {
    label: 'Developer',
    color: 'bg-blue-500/10 text-blue-700 hover:bg-blue-500/20 dark:text-blue-400',
    description:
      'Standard developer. Can view databases, create projects, and work on change plans within assigned projects.',
  },
};

// Project role display names and descriptions
const PROJECT_ROLE_INFO: Record<
  ProjectRole,
  { label: string; color: string; description: string }
> = {
  ProjectOwner: {
    label: 'Owner',
    color: 'bg-purple-500/10 text-purple-700 hover:bg-purple-500/20 dark:text-purple-400',
    description:
      'Project owner. Full control over project settings, members, and all change plan operations.',
  },
  ProjectDeveloper: {
    label: 'Developer',
    color: 'bg-blue-500/10 text-blue-700 hover:bg-blue-500/20 dark:text-blue-400',
    description:
      'Project developer. Can create and update change plans, and execute approved changes.',
  },
  ProjectApprover: {
    label: 'Approver',
    color: 'bg-green-500/10 text-green-700 hover:bg-green-500/20 dark:text-green-400',
    description: 'Project approver. Can approve or reject change plans but cannot execute them.',
  },
  ProjectViewer: {
    label: 'Viewer',
    color: 'bg-gray-500/10 text-gray-700 hover:bg-gray-500/20 dark:text-gray-400',
    description: 'Project viewer. Read-only access to view project and change plan details.',
  },
};

export const RoleBadge: React.FC<RoleBadgeProps> = ({ role, showTooltip = true, size = 'md' }) => {
  // Guard against undefined role
  if (!role) {
    return null;
  }

  // Determine if this is an organization role or project role
  const isOrgRole = role.startsWith('Organization');
  const roleInfo = isOrgRole
    ? ORGANIZATION_ROLE_INFO[role as OrganizationRole]
    : PROJECT_ROLE_INFO[role as ProjectRole];

  if (!roleInfo) {
    return null;
  }

  const sizeClasses = {
    sm: 'text-xs px-2 py-0.5',
    md: 'text-xs px-2.5 py-0.5',
    lg: 'text-sm px-3 py-1',
  };

  const badge = (
    <Badge variant="secondary" className={`${roleInfo.color} ${sizeClasses[size]} font-medium`}>
      {roleInfo.label}
    </Badge>
  );

  if (!showTooltip) {
    return badge;
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>{badge}</TooltipTrigger>
        <TooltipContent className="max-w-xs">
          <p className="font-semibold">{roleInfo.label}</p>
          <p className="text-xs text-muted-foreground mt-1">{roleInfo.description}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};
