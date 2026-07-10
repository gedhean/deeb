import React, { useMemo, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { subDays } from 'date-fns';
import { DefaultLayout } from '@/components/layouts';
import { useUsers } from '@/hooks/api/useUsersSWR';
import { useProjectsSWR } from '@/hooks/api/useProjectsSWR';
import { useDatabases } from '@/hooks/api/useDatabases';
import { useChangePlansSWR } from '@/hooks/api/useChangePlansSWR';
import { projectsService } from '@/lib/api/services/projects';
import { databasesService } from '@/lib/api/services/databases';
import { usersService } from '@/lib/api/services/users';
import { REFERENCE_LIST_LIMIT } from '@/lib/api/pagination';
import {
  useServerTableFilters,
  type ServerTableFilterMapping,
} from '@/hooks/useServerTableFilters';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { DataTable, DataTableToolbar, DataTableViewOptions } from '@/components/data-table';
import type { FilterDefinition } from '@/components/data-table/types';
import { getChangePlansColumns } from './change-plans/columns';
import {
  Plus,
  AlertTriangle,
  FilePlus,
  FileEdit,
  CheckCircle2,
  Loader,
  XCircle,
  Ban,
  Archive,
} from 'lucide-react';
import { ProtectedComponent } from '@/components/ProtectedComponent';
import { OrganizationPermission } from '@/lib/rbac';
import { useCurrentProject } from '@/store';
import { useAuth } from '@/contexts/AuthContext';

const CHANGE_PLANS_FILTER_MAPPING: ServerTableFilterMapping = {
  searchColumnIds: ['title'],
  equals: {
    status: 'status',
    project: 'project_id',
    database: 'database_id',
    author: 'author_id',
    change_type: 'change_type',
  },
  dateRanges: { created_at: ['created_after', 'created_before'] },
};

// Filter definitions are static: the reference-backed filters (project /
// database / author) search the server as you type instead of preloading a full
// option list, so they scale to any number of entities.
const CHANGE_PLANS_FILTER_DEFINITIONS: FilterDefinition[] = [
  {
    id: 'title',
    type: 'search',
    title: 'Title',
    placeholder: 'Search titles...',
    defaultActive: true,
  },
  {
    id: 'status',
    type: 'faceted',
    title: 'Status',
    options: [
      { label: 'Draft', value: 'Draft', icon: FilePlus },
      { label: 'Open', value: 'Open', icon: FileEdit },
      { label: 'Approved', value: 'Approved', icon: CheckCircle2 },
      { label: 'In Progress', value: 'InProgress', icon: Loader },
      { label: 'Completed', value: 'Completed', icon: CheckCircle2 },
      { label: 'Failed', value: 'Failed', icon: XCircle },
      { label: 'Cancelled', value: 'Cancelled', icon: Ban },
      { label: 'Closed', value: 'Closed', icon: Archive },
    ],
    defaultActive: true,
  },
  {
    id: 'project',
    type: 'async',
    title: 'Project',
    placeholder: 'Search projects...',
    defaultActive: true,
    loadOptions: (search) =>
      projectsService
        .listProjects({ search, limit: 20 })
        .then((r) => r.items.map((p) => ({ label: p.name, value: p.id }))),
    resolveLabel: (id) =>
      projectsService
        .getProject(id)
        .then((p) => p.name)
        .catch(() => undefined),
  },
  {
    id: 'database',
    type: 'async',
    title: 'Database',
    placeholder: 'Search databases...',
    loadOptions: (search) =>
      databasesService
        .listDatabases({ search, limit: 20 })
        .then((r) => r.items.map((d) => ({ label: d.database_name, value: d.id }))),
    resolveLabel: (id) =>
      databasesService
        .getDatabase(id)
        .then((d) => d.database_name)
        .catch(() => undefined),
  },
  {
    id: 'author',
    type: 'async',
    title: 'Author',
    placeholder: 'Search authors...',
    defaultActive: true,
    loadOptions: (search) =>
      usersService
        .listUsers({ search, limit: 20 })
        .then((r) => r.items.map((u) => ({ label: u.display_name || u.email, value: u.id }))),
    resolveLabel: (id) =>
      usersService
        .getUser(id)
        .then((u) => u.display_name || u.email)
        .catch(() => undefined),
  },
  {
    id: 'change_type',
    type: 'select',
    title: 'Change Type',
    options: [
      { label: 'DDL', value: 'DDL' },
      { label: 'DML', value: 'DML' },
    ],
  },
  {
    id: 'created_at',
    type: 'dateRange',
    title: 'Created',
    presets: [
      { label: 'Last 7 days', getValue: () => [subDays(new Date(), 7), new Date()] },
      { label: 'Last 30 days', getValue: () => [subDays(new Date(), 30), new Date()] },
      { label: 'Last 90 days', getValue: () => [subDays(new Date(), 90), new Date()] },
    ],
  },
];

export const ChangePlansPage: React.FC = () => {
  const navigate = useNavigate();
  const currentProject = useCurrentProject();
  const { user: currentUser } = useAuth();

  // Reference lists feed the toolbar filter options (author/project/database) and
  // resolve names, so fetch the full set rather than the default page of 100.
  const { users } = useUsers({ limit: REFERENCE_LIST_LIMIT });
  const { projects } = useProjectsSWR({ limit: REFERENCE_LIST_LIMIT });
  const { databases } = useDatabases({ limit: REFERENCE_LIST_LIMIT });

  // Default filters: status "Open", current project, current user author. Async
  // filters (project/author) take a STRING value now, not an array.
  const defaultColumnFilters = useMemo(() => {
    const filters: { id: string; value: unknown }[] = [{ id: 'status', value: ['Open'] }];
    if (currentProject?.id) {
      filters.push({ id: 'project', value: currentProject.id });
    }
    if (currentUser?.id) {
      filters.push({ id: 'author', value: currentUser.id });
    }
    return filters;
  }, [currentProject, currentUser]);

  // Server-side pagination + filtering (state persisted in the URL).
  const {
    params,
    pageIndex,
    pageSize,
    setPageIndex,
    setPageSize,
    onFiltersChange,
    initialColumnFilters,
  } = useServerTableFilters({
    mapping: CHANGE_PLANS_FILTER_MAPPING,
    filterDefinitions: CHANGE_PLANS_FILTER_DEFINITIONS,
    initialPageSize: 50,
    initialColumnFilters: defaultColumnFilters,
  });

  const { changePlans, total, isLoading, error: swrError } = useChangePlansSWR(params);
  const error = swrError || '';

  // Helper functions to get names
  const getProjectName = useCallback(
    (projectId: string | null | undefined): string => {
      if (!projectId) return 'Unassigned';
      const project = projects.find((p) => p.id === projectId);
      return project?.name || projectId;
    },
    [projects],
  );

  const getDatabaseName = useCallback(
    (databaseId: string): string => {
      const database = databases.find((d) => d.id === databaseId);
      return database?.database_name || databaseId;
    },
    [databases],
  );

  const getAuthorName = useCallback(
    (authorId: string): string => {
      const author = users.find((u) => u.id === authorId);
      return author?.email || authorId;
    },
    [users],
  );

  // Get columns for change plans table with memoization to avoid unnecessary re-renders
  const changePlansColumns = useMemo(
    () =>
      getChangePlansColumns({
        onView: (changePlan) => navigate(`/change-plans/${changePlan.id}`),
        getProjectName,
        getDatabaseName,
        getAuthorName,
      }),
    [navigate, getProjectName, getDatabaseName, getAuthorName],
  );

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

        {/* Change Plans Table */}
        <Card>
          <CardHeader>
            <CardTitle>Change Plans</CardTitle>
            <CardDescription>
              {isLoading ? (
                'Loading change plans...'
              ) : (
                <>
                  {total} change plan{total === 1 ? '' : 's'} found
                </>
              )}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={changePlansColumns}
              data={changePlans}
              isLoading={isLoading}
              initialColumnFilters={initialColumnFilters}
              emptyMessage="No change plans match your filters. Try adjusting your search criteria."
              pagination={{
                pageIndex,
                pageSize,
                totalCount: total,
                onPageChange: setPageIndex,
                onPageSizeChange: setPageSize,
                pageSizeOptions: [10, 20, 50, 100],
                manualFiltering: true,
              }}
              onRowClick={(changePlan) => navigate(`/change-plans/${changePlan.id}`)}
              onFiltersChange={onFiltersChange}
            >
              {(table) => (
                <DataTableToolbar table={table} filters={CHANGE_PLANS_FILTER_DEFINITIONS}>
                  <DataTableViewOptions table={table} />
                </DataTableToolbar>
              )}
            </DataTable>
          </CardContent>
        </Card>
      </div>
    </DefaultLayout>
  );
};
