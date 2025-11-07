'use client';

import * as React from 'react';
import {
  Database,
  FileCheck,
  History,
  LayoutDashboard,
  Users,
  Building2,
  FolderKanban,
  FileEdit,
} from 'lucide-react';

import { useAuth } from '@/contexts/AuthContext';
import {
  hasOrganizationPermission,
  hasAnyOrganizationRole,
  OrganizationPermission,
} from '@/lib/rbac';
import { NavMain } from '@/components/nav-main';
import { NavOrg } from '@/components/nav-org';
import { NavUser } from '@/components/nav-user';
import { ProjectSwitcher } from '@/components/project-switcher';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from '@/components/ui/sidebar';
import { useProjects } from '@/hooks/api/useProjects';

// Deeb navigation structure
const navMainItems = [
  {
    title: 'Dashboard',
    url: '/dashboard',
    icon: LayoutDashboard,
    isActive: true,
  },
  {
    title: 'Change Plans',
    url: '/change-plans',
    icon: FileEdit,
    items: [
      {
        title: 'All Change Plans',
        url: '/change-plans',
      },
      {
        title: 'Create Change Plan',
        url: '/change-plans/create',
      },
    ],
  },
  {
    title: 'Databases',
    url: '/databases',
    icon: Database,
    items: [
      {
        title: 'All Databases',
        url: '/databases',
      },
      {
        title: 'Instances',
        url: '/database-instances',
      },
    ],
  },
  {
    title: 'Projects',
    url: '/projects',
    icon: FolderKanban,
  },
];

// Organization-level navigation items
const navOrgItems = [
  {
    title: 'Audit Logs',
    url: '/audit',
    icon: History,
  },
  {
    title: 'Environments',
    url: '/environments',
    icon: Building2,
  },
  {
    title: 'Policies',
    url: '/policies',
    icon: FileCheck,
  },
  {
    title: 'Users',
    url: '/users',
    icon: Users,
  },
];

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const { user } = useAuth();
  const { projects } = useProjects();

  // Get initials from email for avatar fallback
  const getInitials = (email: string | undefined): string => {
    if (!email) {
      return '';
    }

    return email
      .split('@')[0]
      .split('.')
      .map((part) => part[0])
      .join('')
      .toUpperCase()
      .slice(0, 2);
  };

  // Format user data for NavUser component
  const userData = user
    ? {
        name: user?.email.split('@')[0], // Use email username as name
        email: user?.email,
        avatar: '', // No avatar URL yet
        initials: getInitials(user?.email),
      }
    : {
        name: 'who are you?',
        email: 'fix@me.pls',
        avatar: '🔥',
        initials: '🤔',
      };

  // Filter navigation items based on user role and permissions
  const filteredNavMainItems = navMainItems.filter((item) => {
    if (!user) {
      return false;
    }

    // Projects - Admin or users with ProjectsRead permission (which includes project owners through useProjects)
    if (item.url === '/projects') {
      // Show for admins or anyone who can read projects
      return (
        hasAnyOrganizationRole(user.organization_role, ['OrganizationAdmin']) ||
        hasOrganizationPermission(user.organization_role, OrganizationPermission.ProjectsRead)
      );
    }

    // Default: show item
    return true;
  });

  // Filter navigation items based on user role and permissions
  const filteredNavOrgItems = navOrgItems.filter((item) => {
    if (!user) {
      return false;
    }

    // Audit Logs - Admin only
    if (item.url === '/audit') {
      return hasOrganizationPermission(
        user.organization_role,
        OrganizationPermission.AuditLogsRead,
      );
    }

    // Environments - Admin only
    if (item.url === '/environments') {
      return hasAnyOrganizationRole(user.organization_role, ['OrganizationAdmin']);
    }

    // Policies - Admin only
    if (item.url === '/policies') {
      return hasAnyOrganizationRole(user.organization_role, ['OrganizationAdmin']);
    }

    // Users - Admin only
    if (item.url === '/users') {
      return hasOrganizationPermission(user.organization_role, OrganizationPermission.UsersRead);
    }

    // Settings - Admin only
    if (item.url === '/settings') {
      return hasAnyOrganizationRole(user.organization_role, ['OrganizationAdmin']);
    }

    // Default: show item
    return true;
  });

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <ProjectSwitcher projects={projects} />
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={filteredNavMainItems} />
        <NavOrg items={filteredNavOrgItems} />
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={userData} />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
