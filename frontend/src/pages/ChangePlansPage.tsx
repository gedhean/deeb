import React, { useEffect, useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { DefaultLayout } from '@/components/layouts';
import type { ChangePlanStatus } from '@/lib/api/types';
import { useUsers } from '@/hooks/api/useUsersSWR';
import { useProjectsSWR } from '@/hooks/api/useProjectsSWR';
import { useDatabases } from '@/hooks/api/useDatabases';
import { useChangePlansSWR } from '@/hooks/api/useChangePlansSWR';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Plus, ChevronLeft, ChevronRight, AlertTriangle, FileEdit } from 'lucide-react';
import { ProtectedComponent } from '@/components/ProtectedComponent';
import { OrganizationPermission } from '@/lib/rbac';
import { useCurrentProject } from '@/store';
import { useAuth } from '@/contexts/AuthContext';
import { ChangePlanStatusBadge } from '@/components/ChangePlanStatusBadge';

const PAGE_SIZE = 20;

export const ChangePlansPage: React.FC = () => {
  const navigate = useNavigate();
  const currentProject = useCurrentProject();
  const { user: currentUser } = useAuth();
  const [currentPage, setCurrentPage] = useState(1);

  // Use cached data from store instead of fetching
  const { users, isLoading: usersLoading } = useUsers();
  const { projects, isLoading: projectsLoading } = useProjectsSWR();

  // Filter state - initialize with currentProject and currentUser if available
  const [filters, setFilters] = useState<{
    status?: ChangePlanStatus;
    project_id?: string;
    database_id?: string;
    author_id?: string;
  }>({
    status: 'Open' as ChangePlanStatus,
    project_id: currentProject?.id,
    author_id: currentUser?.id,
  });

  // Sync filters with currentProject and currentUser on initial load (only if filter not already set)
  useEffect(() => {
    const needsUpdate =
      (currentProject?.id && !filters.project_id) || (currentUser?.id && !filters.author_id);

    if (needsUpdate) {
      setFilters((prev) => ({
        ...prev,
        ...(currentProject?.id && !prev.project_id ? { project_id: currentProject.id } : {}),
        ...(currentUser?.id && !prev.author_id ? { author_id: currentUser.id } : {}),
      }));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentProject?.id, currentUser?.id]);

  // Fetch databases only when we have a project filter
  const { databases, isLoading: databasesLoading } = useDatabases(
    filters.project_id ? { project_id: filters.project_id } : {},
  );

  const isLoadingFilters = usersLoading || projectsLoading || databasesLoading;

  // Use SWR hook for change plans with stable params
  const changePlansParams = useMemo(
    () => ({
      ...filters,
      limit: PAGE_SIZE,
      offset: (currentPage - 1) * PAGE_SIZE,
    }),
    [filters, currentPage],
  );

  const { changePlans, total, isLoading, error: swrError } = useChangePlansSWR(changePlansParams);
  const error = swrError || '';

  // Handle filter changes
  const handleFilterChange = (key: keyof typeof filters, value: string | undefined) => {
    setFilters((prev) => {
      const updated = { ...prev };
      if (value) {
        if (key === 'status') {
          updated.status = value as ChangePlanStatus;
        } else if (key === 'project_id') {
          updated.project_id = value;
        } else if (key === 'database_id') {
          updated.database_id = value;
        } else if (key === 'author_id') {
          updated.author_id = value;
        }
      } else {
        delete updated[key];
      }
      return updated;
    });
    setCurrentPage(1); // Reset to first page when filter changes
  };

  // Clear all filters
  const clearFilters = () => {
    setFilters({});
    setCurrentPage(1);
  };

  // Format date
  const formatDate = (dateString: string): string => {
    try {
      return new Date(dateString).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
      });
    } catch {
      return dateString;
    }
  };

  // Helper functions to get names
  const getProjectName = (projectId: string): string => {
    const project = projects.find((p) => p.id === projectId);
    return project?.name || projectId;
  };

  const getDatabaseName = (databaseId: string): string => {
    const database = databases.find((d) => d.id === databaseId);
    return database?.database_name || databaseId;
  };

  const getAuthorName = (authorId: string): string => {
    const author = users.find((u) => u.id === authorId);
    return author?.email || authorId;
  };

  // Calculate pagination
  const totalPages = Math.ceil(total / PAGE_SIZE);
  const hasFilters = Object.keys(filters).length > 0;

  return (
    <DefaultLayout breadcrumbs={[{ label: 'Change Plans', isCurrentPage: true }]}>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Change Plans</h1>
            <p className="text-muted-foreground mt-1">View and manage database change plans</p>
          </div>
          <ProtectedComponent requireOrganizationPermission={OrganizationPermission.ProjectsRead}>
            <Button onClick={() => navigate('/change-plans/create')}>
              <Plus className="mr-2 h-4 w-4" />
              Create Change Plan
            </Button>
          </ProtectedComponent>
        </div>

        {/* Error message */}
        {error && (
          <Card className="border-destructive">
            <CardContent className="pt-6">
              <div className="flex items-center gap-2 text-destructive">
                <AlertTriangle className="h-4 w-4" />
                <span className="text-sm">{error}</span>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Filters */}
        <Card>
          <CardHeader>
            <CardTitle>Filters</CardTitle>
            <CardDescription>
              Filter change plans by status, project, database, or author
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-4">
              {/* Status Filter */}
              <div className="space-y-2">
                <label htmlFor="filter-status" className="text-sm font-medium">
                  Status
                </label>
                {isLoadingFilters ? (
                  <Skeleton className="h-9 w-full" />
                ) : (
                  <Select
                    value={filters.status || 'all'}
                    onValueChange={(value) =>
                      handleFilterChange(
                        'status',
                        value === 'all' ? undefined : (value as ChangePlanStatus),
                      )
                    }
                  >
                    <SelectTrigger id="filter-status">
                      <SelectValue placeholder="All Statuses" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Statuses</SelectItem>
                      <SelectItem value="Draft">Draft</SelectItem>
                      <SelectItem value="Open">Open</SelectItem>
                      <SelectItem value="Approved">Approved</SelectItem>
                      <SelectItem value="InProgress">InProgress</SelectItem>
                      <SelectItem value="Completed">Completed</SelectItem>
                      <SelectItem value="Failed">Failed</SelectItem>
                      <SelectItem value="Cancelled">Cancelled</SelectItem>
                      <SelectItem value="Closed">Closed</SelectItem>
                    </SelectContent>
                  </Select>
                )}
              </div>

              {/* Project Filter */}
              <div className="space-y-2">
                <label htmlFor="filter-project" className="text-sm font-medium">
                  Project
                </label>
                {isLoadingFilters ? (
                  <Skeleton className="h-9 w-full" />
                ) : (
                  <Select
                    value={filters.project_id || 'all'}
                    onValueChange={(value) =>
                      handleFilterChange('project_id', value === 'all' ? undefined : value)
                    }
                  >
                    <SelectTrigger id="filter-project">
                      <SelectValue placeholder="All Projects" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Projects</SelectItem>
                      {projects.map((project) => (
                        <SelectItem key={project.id} value={project.id}>
                          {project.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              </div>

              {/* Database Filter */}
              <div className="space-y-2">
                <label htmlFor="filter-database" className="text-sm font-medium">
                  Database
                </label>
                {isLoadingFilters ? (
                  <Skeleton className="h-9 w-full" />
                ) : (
                  <Select
                    value={filters.database_id || 'all'}
                    onValueChange={(value) =>
                      handleFilterChange('database_id', value === 'all' ? undefined : value)
                    }
                  >
                    <SelectTrigger id="filter-database">
                      <SelectValue placeholder="All Databases" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Databases</SelectItem>
                      {databases.map((database) => (
                        <SelectItem key={database.id} value={database.id}>
                          {database.database_name}
                          {database.environment
                            ? ` (${database.environment.name || database.environment_id})`
                            : ''}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              </div>

              {/* Author Filter */}
              <div className="space-y-2">
                <label htmlFor="filter-author" className="text-sm font-medium">
                  Author
                </label>
                {isLoadingFilters ? (
                  <Skeleton className="h-9 w-full" />
                ) : (
                  <Select
                    value={filters.author_id || 'all'}
                    onValueChange={(value) =>
                      handleFilterChange('author_id', value === 'all' ? undefined : value)
                    }
                  >
                    <SelectTrigger id="filter-author">
                      <SelectValue placeholder="All Authors" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Authors</SelectItem>
                      {users.map((user) => (
                        <SelectItem key={user.id} value={user.id}>
                          {user.email}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              </div>
            </div>

            {/* Clear Filters Button */}
            {hasFilters && (
              <div className="mt-4">
                <Button variant="outline" size="sm" onClick={clearFilters}>
                  Clear Filters
                </Button>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Change Plans Table */}
        <Card>
          <CardHeader>
            <CardTitle>Change Plans</CardTitle>
            <CardDescription>
              {total > 0
                ? `${total} change plan${total === 1 ? '' : 's'} found`
                : 'No change plans found'}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-3">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
              </div>
            ) : changePlans.length === 0 ? (
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <FileEdit className="h-12 w-12 text-muted-foreground" />
                  </EmptyMedia>
                  <EmptyTitle>
                    {hasFilters ? 'No Change Plans Found' : 'No Change Plans Yet'}
                  </EmptyTitle>
                  <EmptyDescription>
                    {hasFilters
                      ? 'No change plans match your filters. Try adjusting your search criteria.'
                      : 'Get started by creating your first change plan.'}
                  </EmptyDescription>
                </EmptyHeader>
                <EmptyContent>
                  {hasFilters ? (
                    <Button variant="outline" onClick={clearFilters}>
                      Clear Filters
                    </Button>
                  ) : (
                    <Button onClick={() => navigate('/change-plans/create')}>
                      <Plus className="mr-2 h-4 w-4" />
                      Create Change Plan
                    </Button>
                  )}
                </EmptyContent>
              </Empty>
            ) : (
              <>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Title</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Project</TableHead>
                      <TableHead>Database</TableHead>
                      <TableHead>Author</TableHead>
                      <TableHead>Change Type</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {changePlans.map((changePlan) => (
                      <TableRow
                        key={changePlan.id}
                        onClick={() => navigate(`/change-plans/${changePlan.id}`)}
                        className="cursor-pointer hover:bg-muted/50 transition-colors"
                      >
                        <TableCell className="font-medium">
                          <div className="flex flex-col">
                            <span className="truncate max-w-[200px]">{changePlan.title}</span>
                            {changePlan.description && (
                              <span className="text-xs text-muted-foreground truncate max-w-[200px]">
                                {changePlan.description}
                              </span>
                            )}
                          </div>
                        </TableCell>
                        <TableCell>
                          <ChangePlanStatusBadge status={changePlan.status} />
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {getProjectName(changePlan.project_id)}
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {getDatabaseName(changePlan.target_database_id)}
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {getAuthorName(changePlan.author_id)}
                        </TableCell>
                        <TableCell>
                          <Badge variant="outline">{changePlan.change_type}</Badge>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatDate(changePlan.created_at)}
                        </TableCell>
                        <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={(e) => {
                              e.stopPropagation();
                              navigate(`/change-plans/${changePlan.id}`);
                            }}
                          >
                            View
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>

                {/* Pagination */}
                {totalPages > 1 && (
                  <div className="flex items-center justify-between mt-4 pt-4 border-t">
                    <div className="text-sm text-muted-foreground">
                      Page {currentPage} of {totalPages} ({total} total)
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setCurrentPage((prev) => Math.max(1, prev - 1))}
                        disabled={currentPage === 1 || isLoading}
                      >
                        <ChevronLeft className="h-4 w-4" />
                        Previous
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setCurrentPage((prev) => Math.min(totalPages, prev + 1))}
                        disabled={currentPage === totalPages || isLoading}
                      >
                        Next
                        <ChevronRight className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                )}
              </>
            )}
          </CardContent>
        </Card>
      </div>
    </DefaultLayout>
  );
};
