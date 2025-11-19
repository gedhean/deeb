import React, { useState, useMemo, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { subDays } from 'date-fns';
import { DefaultLayout } from '@/components/layouts';
import { useUsers } from '@/hooks/api/useUsersSWR';
import { useProjectsSWR } from '@/hooks/api/useProjectsSWR';
import { useDatabases } from '@/hooks/api/useDatabases';
import { useChangePlansSWR } from '@/hooks/api/useChangePlansSWR';
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

export const ChangePlansPage: React.FC = () => {
  const navigate = useNavigate();
  const currentProject = useCurrentProject();
  const { user: currentUser } = useAuth();

  // Pagination state
  const [pageIndex, setPageIndex] = useState(0);
  const [pageSize, setPageSize] = useState(20);

  // Use cached data from store instead of fetching
  const { users } = useUsers();
  const { projects } = useProjectsSWR();

  // Fetch all databases for filtering
  const { databases } = useDatabases({});

  // Use SWR hook for change plans with pagination
  const changePlansParams = useMemo(
    () => ({
      limit: pageSize,
      offset: pageIndex * pageSize,
    }),
    [pageIndex, pageSize],
  );

  const { changePlans, total, isLoading, error: swrError } = useChangePlansSWR(changePlansParams);
  const error = swrError || '';

  // Helper functions to get names
  const getProjectName = useCallback(
    (projectId: string): string => {
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

  // Prepare filter options for project, database, and author
  const projectOptions = useMemo(
    () =>
      projects.map((project) => ({
        label: project.name,
        value: project.id,
      })),
    [projects],
  );

  const databaseOptions = useMemo(
    () =>
      databases.map((database) => ({
        label: database.database_name,
        value: database.id,
      })),
    [databases],
  );

  const authorOptions = useMemo(
    () =>
      users.map((user) => ({
        label: user.email,
        value: user.id,
      })),
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

  // Calculate initial column filters
  const initialColumnFilters = useMemo(() => {
    const filters: { id: string; value: unknown }[] = [];

    // Set default status to "Open"
    filters.push({ id: 'status', value: ['Open'] });

    // Set default project if currentProject exists
    if (currentProject?.id) {
      filters.push({ id: 'project', value: [currentProject.id] });
    }

    // Set default author if currentUser exists
    if (currentUser?.id) {
      filters.push({ id: 'author', value: [currentUser.id] });
    }

    return filters;
  }, [currentProject, currentUser]);

  // Define filter definitions for the toolbar
  const filterDefinitions: FilterDefinition[] = useMemo(
    () => [
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
        type: 'faceted',
        title: 'Project',
        options: projectOptions,
        defaultActive: true,
      },
      {
        id: 'database',
        type: 'faceted',
        title: 'Database',
        options: databaseOptions,
      },
      {
        id: 'author',
        type: 'faceted',
        title: 'Author',
        options: authorOptions,
        defaultActive: true,
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
          {
            label: 'Last 7 days',
            getValue: () => [subDays(new Date(), 7), new Date()],
          },
          {
            label: 'Last 30 days',
            getValue: () => [subDays(new Date(), 30), new Date()],
          },
          {
            label: 'Last 90 days',
            getValue: () => [subDays(new Date(), 90), new Date()],
          },
        ],
      },
    ],
    [projectOptions, databaseOptions, authorOptions],
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
                onPageSizeChange: (newSize) => {
                  setPageSize(newSize);
                  setPageIndex(0);
                },
                pageSizeOptions: [10, 20, 30, 50],
              }}
              onRowClick={(changePlan) => navigate(`/change-plans/${changePlan.id}`)}
            >
              {(table) => (
                <DataTableToolbar table={table} filters={filterDefinitions}>
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
