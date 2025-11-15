'use client';

import { useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { ChevronsUpDown, Plus, FolderKanban, Search } from 'lucide-react';
import { useAuth } from '@/contexts/AuthContext';
import { hasOrganizationPermission, OrganizationPermission } from '@/lib/rbac';

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar';
import { EncryptedText } from '@/components/ui/encrypted-text';
import { useCurrentProject, useSetCurrentProject } from '@/store';
import type { Project } from '@/lib/api/types';

export function ProjectSwitcher({ projects }: { projects: Project[] }) {
  const { isMobile } = useSidebar();
  const { user } = useAuth();
  const navigate = useNavigate();
  const currentProject = useCurrentProject();
  const setCurrentProject = useSetCurrentProject();
  const [searchQuery, setSearchQuery] = useState('');

  // Check if user can create projects (DBA or Admin)
  const canCreateProjects = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.ProjectsCreate,
  );

  // Use current project from store, or first project if available
  const activeProject = currentProject || projects[0];

  // Filter projects based on search query
  const filteredProjects = useMemo(() => {
    if (!searchQuery.trim()) {
      return projects;
    }
    const query = searchQuery.toLowerCase();
    return projects.filter(
      (project) =>
        project.name.toLowerCase().includes(query) ||
        project.description?.toLowerCase().includes(query),
    );
  }, [projects, searchQuery]);

  const handleProjectChange = (projectId: string) => {
    const selectedProject = projects.find((p) => p.id === projectId);
    if (selectedProject) {
      setCurrentProject(selectedProject);
      setSearchQuery(''); // Clear search when project is selected
    }
  };

  const handleAddProject = () => {
    navigate('/projects');
  };

  if (!activeProject || projects.length === 0) {
    return null;
  }

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
            >
              <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
                <FolderKanban className="size-4" />
              </div>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-semibold">
                  <EncryptedText text={activeProject.name} revealDelayMs={30} />
                </span>
                <span className="truncate text-xs">{activeProject.description || 'Project'}</span>
              </div>
              <ChevronsUpDown className="ml-auto" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg"
            align="start"
            side={isMobile ? 'bottom' : 'right'}
            sideOffset={4}
          >
            <DropdownMenuLabel className="text-xs text-muted-foreground">
              Projects
            </DropdownMenuLabel>
            {/* Search Input */}
            <div className="px-2 py-1.5">
              <div className="relative">
                <Search className="absolute left-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  type="text"
                  placeholder="Search projects..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="h-8 pl-8 pr-2 text-sm"
                  autoFocus
                  onClick={(e) => e.stopPropagation()}
                  onKeyDown={(e) => {
                    e.stopPropagation();
                    // Prevent closing dropdown on Escape if search has value
                    if (e.key === 'Escape' && searchQuery) {
                      e.preventDefault();
                      setSearchQuery('');
                    }
                  }}
                />
              </div>
            </div>
            <DropdownMenuSeparator />
            {/* Filtered Projects */}
            {filteredProjects.length === 0 ? (
              <div className="px-2 py-6 text-center text-sm text-muted-foreground">
                No projects found
              </div>
            ) : (
              filteredProjects.map((project) => (
                <DropdownMenuItem
                  key={project.id}
                  onClick={() => handleProjectChange(project.id)}
                  className="gap-2 p-2"
                >
                  <div className="flex size-6 items-center justify-center rounded-sm border">
                    <FolderKanban className="size-4 shrink-0" />
                  </div>
                  {project.name}
                </DropdownMenuItem>
              ))
            )}
            {canCreateProjects && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem className="gap-2 p-2" onClick={handleAddProject}>
                  <div className="flex size-6 items-center justify-center rounded-md border bg-background">
                    <Plus className="size-4" />
                  </div>
                  <div className="font-medium text-muted-foreground">Add project</div>
                </DropdownMenuItem>
              </>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
