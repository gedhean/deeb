import React from 'react';
import { AppSidebar } from '@/components/app-sidebar';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Separator } from '@/components/ui/separator';
import { SidebarInset, SidebarProvider, SidebarTrigger } from '@/components/ui/sidebar';
import { ChevronDown, FolderGit2, Check } from 'lucide-react';

/**
 * Breadcrumb item configuration
 */
export interface BreadcrumbItemConfig {
  /** Link text */
  label: string;
  /** Optional href for clickable breadcrumb items */
  href?: string;
  /** Whether this is the current/active page (last item) */
  isCurrentPage?: boolean;
}

/**
 * Project configuration for the project selector
 */
export interface ProjectConfig {
  /** Unique identifier for the project */
  id: string;
  /** Project display name */
  name: string;
}

/**
 * Default layout component props
 */
export interface DefaultLayoutProps {
  /** Page content to render in the main area */
  children: React.ReactNode;
  /** Breadcrumb items to display in the header */
  breadcrumbs?: BreadcrumbItemConfig[];
  /** List of available projects for the dropdown */
  projects?: ProjectConfig[];
  /** Currently selected project ID */
  currentProjectId?: string;
  /** Callback when a project is selected */
  onProjectChange?: (projectId: string) => void;
}

/**
 * Default application layout with sidebar and breadcrumb navigation.
 * Provides a consistent layout structure across authenticated pages.
 *
 * @param children - The main page content
 * @param breadcrumbs - Optional breadcrumb navigation items
 * @param projects - Optional list of projects for the dropdown selector
 * @param currentProjectId - Currently selected project ID
 * @param onProjectChange - Callback when a project is selected
 */
export const DefaultLayout: React.FC<DefaultLayoutProps> = ({
  children,
  breadcrumbs = [],
  projects = [],
  currentProjectId,
  onProjectChange,
}) => {
  const currentProject = projects.find((p) => p.id === currentProjectId);

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <header className="flex h-16 shrink-0 items-center gap-2 transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-12">
          <div className="flex items-center gap-2 px-4">
            <SidebarTrigger className="-ml-1" />
            <Separator orientation="vertical" className="mr-2 data-[orientation=vertical]:h-4" />

            {/* Project Selector Dropdown */}
            {projects.length > 0 && (
              <>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" className="h-8 gap-2 px-2">
                      <FolderGit2 className="size-4" />
                      <span className="hidden md:inline-block text-muted-foreground text-xs">
                        {currentProject?.name || 'Select Project'}
                      </span>
                      <ChevronDown className="size-4 opacity-50" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start" className="w-56">
                    <DropdownMenuLabel className="text-muted-foreground text-xs">
                      Select Project
                    </DropdownMenuLabel>
                    <DropdownMenuSeparator />
                    {projects.map((project) => (
                      <DropdownMenuItem
                        key={project.id}
                        onClick={() => onProjectChange?.(project.id)}
                        className="flex items-center justify-between cursor-pointer"
                      >
                        <div className="flex items-center gap-2">
                          <FolderGit2 className="size-4" />
                          <span>{project.name}</span>
                        </div>
                        {project.id === currentProjectId && <Check className="size-4" />}
                      </DropdownMenuItem>
                    ))}
                  </DropdownMenuContent>
                </DropdownMenu>
                {breadcrumbs.length > 0 && (
                  <Separator
                    orientation="vertical"
                    className="mx-2 data-[orientation=vertical]:h-4"
                  />
                )}
              </>
            )}

            {/* Breadcrumb Navigation */}
            {breadcrumbs.length > 0 && (
              <Breadcrumb>
                <BreadcrumbList>
                  {breadcrumbs.map((item, index) => (
                    <React.Fragment key={index}>
                      {index > 0 && <BreadcrumbSeparator className="hidden md:block" />}
                      <BreadcrumbItem className={index === 0 ? 'hidden md:block' : ''}>
                        {item.isCurrentPage || !item.href ? (
                          <BreadcrumbPage>{item.label}</BreadcrumbPage>
                        ) : (
                          <BreadcrumbLink href={item.href}>{item.label}</BreadcrumbLink>
                        )}
                      </BreadcrumbItem>
                    </React.Fragment>
                  ))}
                </BreadcrumbList>
              </Breadcrumb>
            )}
          </div>
        </header>
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">{children}</div>
      </SidebarInset>
    </SidebarProvider>
  );
};
