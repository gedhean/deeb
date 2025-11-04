'use client';

import * as React from 'react';
import {
  AudioWaveform,
  Database,
  FileCheck,
  CircleCheckBig,
  GalleryVerticalEnd,
  History,
  LayoutDashboard,
  Settings2,
  Users,
  Building2,
  FolderKanban,
} from 'lucide-react';

import { useAuth } from '@/contexts/AuthContext';
import { NavMain } from '@/components/nav-main';
import { NavOrg } from '@/components/nav-org';
import { NavProjects } from '@/components/nav-projects';
import { NavUser } from '@/components/nav-user';
import { OrgSwitcher } from '@/components/org-switcher';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from '@/components/ui/sidebar';

// Placeholder orgs data - will be replaced with real data from API in future
const placeholderOrgs = [
  {
    name: 'My Organization',
    logo: GalleryVerticalEnd,
    plan: 'Enterprise',
  },
];

// Deeb navigation structure
const navMainItems = [
  {
    title: 'Dashboard',
    url: '/dashboard',
    icon: LayoutDashboard,
    isActive: true,
  },
  {
    title: 'Projects',
    url: '/projects',
    icon: FolderKanban,
  },
  {
    title: 'Issues',
    url: '/issues',
    icon: FileCheck,
    items: [
      {
        title: 'All Issues',
        url: '/issues',
      },
      {
        title: 'My Issues',
        url: '/issues/mine',
      },
      {
        title: 'Create Issue',
        url: '/issues/new',
      },
    ],
  },
  {
    title: 'Approvals',
    url: '/approvals',
    icon: CircleCheckBig,
    items: [
      {
        title: 'Pending Approval',
        url: '/approvals/pending',
      },
      {
        title: 'Approved',
        url: '/approvals/approved',
      },
      {
        title: 'Rejected',
        url: '/approvals/rejected',
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
    title: 'Settings',
    url: '/settings',
    icon: Settings2,
    items: [
      {
        title: 'Policies',
        url: '/settings/policies',
      },
      {
        title: 'Users & Groups',
        url: '/settings/users-groups',
        icon: Users,
      },
    ],
  },
];

// Placeholder projects - will be fetched from API in future
const placeholderProjects = [
  {
    name: 'My Projects',
    url: '/projects',
    icon: AudioWaveform,
  },
];

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const { user } = useAuth();

  // Get initials from email for avatar fallback
  const getInitials = (email: string): string => {
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
        name: user.email.split('@')[0], // Use email username as name
        email: user.email,
        avatar: '', // No avatar URL yet
        initials: getInitials(user.email),
      }
    : {
        name: 'Guest',
        email: 'guest@example.com',
        avatar: '',
        initials: 'GU',
      };

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <OrgSwitcher orgs={placeholderOrgs} />
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={navMainItems} />
        <NavProjects projects={placeholderProjects} />
        <NavOrg items={navOrgItems} />
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={userData} />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
