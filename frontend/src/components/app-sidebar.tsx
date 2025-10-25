'use client';

import * as React from 'react';
import {
  AudioWaveform,
  BookOpen,
  Database,
  FileCheck,
  GalleryVerticalEnd,
  History,
  LayoutDashboard,
  Settings2,
} from 'lucide-react';

import { useAuth } from '@/contexts/AuthContext';
import { NavMain } from '@/components/nav-main';
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
    icon: FileCheck,
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
        url: '/databases/instances',
      },
    ],
  },
  {
    title: 'Audit Logs',
    url: '/audit',
    icon: History,
  },
  {
    title: 'Documentation',
    url: '#',
    icon: BookOpen,
    items: [
      {
        title: 'Getting Started',
        url: '#',
      },
      {
        title: 'User Guide',
        url: '#',
      },
      {
        title: 'API Reference',
        url: '#',
      },
    ],
  },
  {
    title: 'Settings',
    url: '/settings',
    icon: Settings2,
    items: [
      {
        title: 'Profile',
        url: '/settings/profile',
      },
      {
        title: 'Security',
        url: '/settings/security',
      },
      {
        title: 'Notifications',
        url: '/settings/notifications',
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
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={userData} />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
