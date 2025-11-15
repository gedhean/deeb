import React from 'react';
import { Link } from 'react-router-dom';
import { AppSidebar } from '@/components/app-sidebar';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';
import { Separator } from '@/components/ui/separator';
import { SidebarInset, SidebarProvider, SidebarTrigger } from '@/components/ui/sidebar';
import { ShootingStars } from '@/components/ui/shooting-stars';
import { StarsBackground } from '@/components/ui/stars-background';

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
 * Default layout component props
 */
export interface DefaultLayoutProps {
  /** Page content to render in the main area */
  children: React.ReactNode;
  /** Breadcrumb items to display in the header */
  breadcrumbs?: BreadcrumbItemConfig[];
}

/**
 * Default application layout with sidebar and breadcrumb navigation.
 * Provides a consistent layout structure across authenticated pages.
 *
 * @param children - The main page content
 * @param breadcrumbs - Optional breadcrumb navigation items
 */
export const DefaultLayout: React.FC<DefaultLayoutProps> = ({ children, breadcrumbs = [] }) => {
  return (
    <div className="relative min-h-screen w-full overflow-hidden rounded-md bg-neutral-900">
      {/* Animated background */}
      <StarsBackground className="absolute z-0 inset-0" />
      <ShootingStars className="absolute z-0 inset-0" />

      <SidebarProvider>
        <AppSidebar />
        <SidebarInset className="bg-transparent">
          <header className="flex h-16 shrink-0 items-center gap-2 transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-12">
            <div className="flex items-center gap-2 px-4">
              <SidebarTrigger className="-ml-1" />
              {breadcrumbs.length > 0 && (
                <>
                  <Separator
                    orientation="vertical"
                    className="mr-2 data-[orientation=vertical]:h-4"
                  />
                  {/* Breadcrumb Navigation */}
                  <Breadcrumb>
                    <BreadcrumbList>
                      {breadcrumbs.map((item, index) => (
                        <React.Fragment key={index}>
                          {index > 0 && <BreadcrumbSeparator className="hidden md:block" />}
                          <BreadcrumbItem className={index === 0 ? 'hidden md:block' : ''}>
                            {item.isCurrentPage || !item.href ? (
                              <BreadcrumbPage>{item.label}</BreadcrumbPage>
                            ) : (
                              <BreadcrumbLink asChild>
                                <Link to={item.href}>{item.label}</Link>
                              </BreadcrumbLink>
                            )}
                          </BreadcrumbItem>
                        </React.Fragment>
                      ))}
                    </BreadcrumbList>
                  </Breadcrumb>
                </>
              )}
            </div>
          </header>
          <div className="flex flex-1 flex-col gap-4 p-4 pt-0">{children}</div>
        </SidebarInset>
      </SidebarProvider>
    </div>
  );
};
