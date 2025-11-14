import { BadgeCheck, ChevronsUpDown, LogOut, FolderOpen } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useState, useEffect } from 'react';

import { useAuth } from '@/contexts/AuthContext';
import { RoleBadge } from '@/components/RoleBadge';
import type { OrganizationRole, ProjectRole } from '@/lib/rbac';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { useProjects } from '@/hooks/api/useProjects';
import { projectMembersService } from '@/lib/api/services/projects';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar';

interface UserProject {
  id: string;
  name: string;
  role: ProjectRole;
}

export function NavUser({
  user,
}: {
  user: {
    name: string;
    email: string;
    avatar: string;
    initials?: string;
  };
}) {
  const { isMobile } = useSidebar();
  const { logout, user: authUser } = useAuth();
  const navigate = useNavigate();
  const { projects } = useProjects();
  const [userProjects, setUserProjects] = useState<UserProject[]>([]);

  // Fetch user's projects with their roles
  useEffect(() => {
    const fetchUserProjects = async () => {
      if (!authUser?.id || projects.length === 0) {
        return;
      }

      const projectsWithRoles: UserProject[] = [];

      for (const project of projects) {
        try {
          const members = await projectMembersService.listProjectMembers(project.id);
          const userMembership = members.find((m) => m.member.user_id === authUser.id);

          if (userMembership) {
            projectsWithRoles.push({
              id: project.id,
              name: project.name,
              role: userMembership.member.role as ProjectRole,
            });
          }
        } catch {
          // Skip projects where we can't fetch members
        }
      }

      setUserProjects(projectsWithRoles);
    };

    fetchUserProjects();
  }, [authUser, projects]);

  const handleLogout = () => {
    logout();
  };

  const handleProfileClick = () => {
    navigate('/profile');
  };

  const handleProjectClick = (projectId: string) => {
    navigate(`/projects/${projectId}`);
  };

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
            >
              <Avatar className="h-8 w-8 rounded-lg">
                <AvatarImage src={user.avatar} alt={user.name} />
                <AvatarFallback className="rounded-lg">{user.initials || 'U'}</AvatarFallback>
              </Avatar>
              <div className="flex flex-1 items-center gap-2 text-left text-sm">
                <div className="grid flex-1 leading-tight">
                  <span className="truncate font-semibold">{user.name}</span>
                  <span className="truncate text-xs">{user.email}</span>
                </div>
                {authUser?.organization_role && (
                  <RoleBadge
                    role={authUser.organization_role as OrganizationRole}
                    showTooltip={false}
                    size="sm"
                  />
                )}
              </div>
              <ChevronsUpDown className="ml-auto size-4" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg"
            side={isMobile ? 'bottom' : 'right'}
            align="end"
            sideOffset={4}
          >
            <DropdownMenuLabel className="p-0 font-normal">
              <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                <Avatar className="h-8 w-8 rounded-lg">
                  <AvatarImage src={user.avatar} alt={user.name} />
                  <AvatarFallback className="rounded-lg">{user.initials || 'U'}</AvatarFallback>
                </Avatar>
                <div className="flex flex-1 items-center gap-2 text-left text-sm">
                  <div className="grid flex-1 leading-tight">
                    <span className="truncate font-semibold">{user.name}</span>
                    <span className="truncate text-xs">{user.email}</span>
                  </div>
                  {authUser?.organization_role && (
                    <RoleBadge
                      role={authUser.organization_role as OrganizationRole}
                      showTooltip={true}
                      size="sm"
                    />
                  )}
                </div>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuItem onClick={handleProfileClick}>
                <BadgeCheck />
                Profile
              </DropdownMenuItem>
            </DropdownMenuGroup>
            {userProjects.length > 0 && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuLabel className="px-2 py-1.5 text-xs font-semibold text-muted-foreground">
                  My Projects
                </DropdownMenuLabel>
                <DropdownMenuGroup className="max-h-[200px] overflow-y-auto">
                  {userProjects
                    .filter((project) => project.role) // Only show projects with valid roles
                    .map((project) => (
                      <DropdownMenuItem
                        key={project.id}
                        onClick={() => handleProjectClick(project.id)}
                        className="flex items-center justify-between gap-2"
                      >
                        <div className="flex items-center gap-2">
                          <FolderOpen className="h-4 w-4" />
                          <span className="truncate">{project.name}</span>
                        </div>
                        <RoleBadge role={project.role} showTooltip={false} size="sm" />
                      </DropdownMenuItem>
                    ))}
                </DropdownMenuGroup>
              </>
            )}
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={handleLogout}>
              <LogOut />
              Log out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
