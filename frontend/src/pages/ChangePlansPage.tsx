import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { DefaultLayout } from '@/components/layouts';
import { changePlanService } from '@/lib/api/services/changePlans';
import { projectsService } from '@/lib/api/services/projects';
import { databasesService } from '@/lib/api/services/databases';
import { usersService } from '@/lib/api/services/users';
import type { ChangePlan, ChangePlanStatus, Project, Database, User } from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
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

const PAGE_SIZE = 20;

export const ChangePlansPage: React.FC = () => {
  const navigate = useNavigate();
  const currentProject = useCurrentProject();
  const [changePlans, setChangePlans] = useState<ChangePlan[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [databases, setDatabases] = useState<Database[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingFilters, setIsLoadingFilters] = useState(true);
  const [error, setError] = useState('');
  const [total, setTotal] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);

  // Filter state - initialize with currentProject if available
  const [filters, setFilters] = useState<{
    status?: ChangePlanStatus;
    project_id?: string;
    database_id?: string;
    author_id?: string;
  }>({
    project_id: currentProject?.id,
  });

  // Load filter options
  useEffect(() => {
    const loadFilters = async () => {
      try {
        setIsLoadingFilters(true);
        const [projectsRes, databasesRes, usersRes] = await Promise.all([
          projectsService.listProjects({ limit: 100 }),
          databasesService.listDatabases({ limit: 100 }),
          usersService.listUsers({ limit: 100 }),
        ]);
        setProjects(projectsRes.items);
        setDatabases(databasesRes.items);
        setUsers(usersRes.items);
      } catch (err) {
        console.error('Failed to load filter options:', err);
      } finally {
        setIsLoadingFilters(false);
      }
    };

    loadFilters();
  }, []);

  // Sync filters with currentProject on initial load (only if filter not already set)
  useEffect(() => {
    if (currentProject?.id && !filters.project_id) {
      setFilters((prev) => ({ ...prev, project_id: currentProject.id }));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentProject]);

  // Load change plans
  useEffect(() => {
    loadChangePlans();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filters, currentPage]);

  const loadChangePlans = async () => {
    try {
      setIsLoading(true);
      setError('');
      const offset = (currentPage - 1) * PAGE_SIZE;
      const response = await changePlanService.listChangePlans({
        ...filters,
        limit: PAGE_SIZE,
        offset,
      });
      setChangePlans(response.change_plans);
      setTotal(response.total);
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to load change plans. Please try again.');
      }
      console.error('Failed to load change plans:', err);
    } finally {
      setIsLoading(false);
    }
  };

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

  // Get status badge variant
  const getStatusBadgeVariant = (
    status: ChangePlanStatus,
  ): 'default' | 'secondary' | 'destructive' | 'outline' => {
    switch (status) {
      case 'Open':
        return 'default';
      case 'Approved':
        return 'default';
      case 'Completed':
        return 'default';
      case 'Draft':
        return 'secondary';
      case 'InProgress':
        return 'outline';
      case 'Failed':
        return 'destructive';
      case 'Cancelled':
        return 'secondary';
      case 'Closed':
        return 'secondary';
      default:
        return 'secondary';
    }
  };

  // Get status badge color class
  const getStatusBadgeClass = (status: ChangePlanStatus): string => {
    switch (status) {
      case 'Open':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200';
      case 'Approved':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
      case 'Completed':
        return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-200';
      case 'Draft':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
      case 'InProgress':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
      case 'Failed':
        return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
      case 'Cancelled':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
      case 'Closed':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
    }
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
                  <select
                    id="filter-status"
                    value={filters.status || ''}
                    onChange={(e) => handleFilterChange('status', e.target.value || undefined)}
                    className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <option value="">All Statuses</option>
                    <option value="Draft">Draft</option>
                    <option value="Open">Open</option>
                    <option value="Approved">Approved</option>
                    <option value="InProgress">InProgress</option>
                    <option value="Completed">Completed</option>
                    <option value="Failed">Failed</option>
                    <option value="Cancelled">Cancelled</option>
                    <option value="Closed">Closed</option>
                  </select>
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
                  <select
                    id="filter-project"
                    value={filters.project_id || ''}
                    onChange={(e) => handleFilterChange('project_id', e.target.value || undefined)}
                    className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <option value="">All Projects</option>
                    {projects.map((project) => (
                      <option key={project.id} value={project.id}>
                        {project.name}
                      </option>
                    ))}
                  </select>
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
                  <select
                    id="filter-database"
                    value={filters.database_id || ''}
                    onChange={(e) => handleFilterChange('database_id', e.target.value || undefined)}
                    className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <option value="">All Databases</option>
                    {databases.map((database) => (
                      <option key={database.id} value={database.id}>
                        {database.database_name}
                        {database.environment
                          ? ` (${database.environment.name || database.environment_id})`
                          : ''}
                      </option>
                    ))}
                  </select>
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
                  <select
                    id="filter-author"
                    value={filters.author_id || ''}
                    onChange={(e) => handleFilterChange('author_id', e.target.value || undefined)}
                    className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <option value="">All Authors</option>
                    {users.map((user) => (
                      <option key={user.id} value={user.id}>
                        {user.email}
                      </option>
                    ))}
                  </select>
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
                          <Badge
                            variant={getStatusBadgeVariant(changePlan.status)}
                            className={getStatusBadgeClass(changePlan.status)}
                          >
                            {changePlan.status}
                          </Badge>
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
