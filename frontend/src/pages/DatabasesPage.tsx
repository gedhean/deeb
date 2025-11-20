import React, { useEffect, useState, useMemo, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { subDays } from 'date-fns';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { databasesService } from '@/lib/api/services/databases';
import { CreateDatabaseDialog } from '@/components/CreateDatabaseDialog';
import { hasOrganizationPermission, OrganizationPermission } from '@/lib/rbac';
import { useProjectsSWR } from '@/hooks/api/useProjectsSWR';
import { useDatabases } from '@/hooks/api/useDatabases';
import { useEnvironmentsSWR } from '@/hooks/api/useEnvironmentsSWR';
import { useDatabaseInstancesSWR } from '@/hooks/api/useDatabaseInstancesSWR';
import type { CreateDatabaseRequest, Database, UpdateDatabaseRequest } from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { DataTable, DataTableToolbar, DataTableViewOptions } from '@/components/data-table';
import type { FilterDefinition } from '@/components/data-table/types';
import { getDatabasesColumns } from './databases/columns';
import { Plus, AlertTriangle, Database as DatabaseIcon } from 'lucide-react';
import { useCurrentProject } from '@/store';

export const DatabasesPage: React.FC = () => {
  const { user } = useAuth();
  const currentProject = useCurrentProject();
  const navigate = useNavigate();
  const [error, setError] = useState('');
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [selectedDatabase, setSelectedDatabase] = useState<Database | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Use cached data from SWR hooks
  const { projects, isLoading: projectsLoading } = useProjectsSWR();
  const { databases, isLoading: databasesLoading, refetch: refetchDatabases } = useDatabases({});
  const { environments } = useEnvironmentsSWR();
  const { databaseInstances } = useDatabaseInstancesSWR();

  const isLoading = projectsLoading || databasesLoading;

  // Pagination state
  const [pageIndex, setPageIndex] = useState(0);
  const [pageSize, setPageSize] = useState(10);

  // Form state
  const [formData, setFormData] = useState<CreateDatabaseRequest>({
    project_id: '',
    database_instance_id: '',
    environment_id: '',
    database_name: '',
    description: '',
  });
  const [formErrors, setFormErrors] = useState<{
    project_id?: string;
    database_instance_id?: string;
    environment_id?: string;
    database_name?: string;
  }>({});

  // Check permissions
  const canReadDatabases = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.DatabasesRead,
  );
  const canCreateDatabases = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.DatabasesCreate,
  );
  const canUpdateDatabases = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.DatabasesUpdate,
  );
  const canDeleteDatabases = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.DatabasesDelete,
  );

  // Check permissions for error display
  useEffect(() => {
    if (!canReadDatabases) {
      setError('Access denied. You do not have permission to view databases.');
    }
  }, [canReadDatabases]);

  // Refetch databases when create dialog closes (after a successful creation)
  const prevIsCreateOpen = React.useRef(isCreateOpen);
  useEffect(() => {
    if (prevIsCreateOpen.current && !isCreateOpen) {
      // Dialog was just closed, refetch to get the new database
      refetchDatabases();
    }
    prevIsCreateOpen.current = isCreateOpen;
  }, [isCreateOpen, refetchDatabases]);

  // Validate form
  const validateForm = (): boolean => {
    const errors: {
      project_id?: string;
      database_instance_id?: string;
      environment_id?: string;
      database_name?: string;
    } = {};

    if (!formData.project_id) {
      errors.project_id = 'Project is required';
    }

    if (!formData.database_instance_id) {
      errors.database_instance_id = 'Database instance is required';
    }

    if (!formData.environment_id) {
      errors.environment_id = 'Environment is required';
    }

    if (!formData.database_name.trim()) {
      errors.database_name = 'Database name is required';
    } else if (formData.database_name.trim().length < 1) {
      errors.database_name = 'Database name must be at least 1 character';
    }

    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  // Handle create

  // Handle edit
  const handleEdit = async () => {
    if (!selectedDatabase || !validateForm()) {
      return;
    }

    try {
      setIsSubmitting(true);
      const updateData: UpdateDatabaseRequest = {};
      if (formData.database_name !== selectedDatabase.database_name) {
        updateData.database_name = formData.database_name.trim();
      }
      if (formData.description !== selectedDatabase.description) {
        updateData.description = formData.description?.trim() || undefined;
      }

      await databasesService.updateDatabase(selectedDatabase.id, updateData);
      setIsEditOpen(false);
      setSelectedDatabase(null);
      setFormData({
        project_id: '',
        database_instance_id: '',
        environment_id: '',
        database_name: '',
        description: '',
      });
      setFormErrors({});
      // Reload databases from cache
      await refetchDatabases();
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to update database. Please try again.');
      }
      console.error('Failed to update database:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Handle delete
  const handleDelete = async () => {
    if (!selectedDatabase) {
      return;
    }

    try {
      setIsSubmitting(true);
      await databasesService.deleteDatabase(selectedDatabase.id);
      setIsDeleteOpen(false);
      setSelectedDatabase(null);
      // Reload databases from cache
      await refetchDatabases();
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to delete database. Please try again.');
      }
      console.error('Failed to delete database:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Open create form
  const openCreateForm = () => {
    setIsCreateOpen(true);
  };

  // Open edit form
  const openEditForm = useCallback((database: Database) => {
    setSelectedDatabase(database);
    setFormData({
      project_id: database.project_id || '',
      database_instance_id: database.database_instance_id,
      environment_id: database.environment_id || '',
      database_name: database.database_name,
      description: database.description || '',
    });
    setFormErrors({});
    setIsEditOpen(true);
  }, []);

  // Open delete confirmation
  const openDeleteConfirmation = useCallback((database: Database) => {
    setSelectedDatabase(database);
    setIsDeleteOpen(true);
  }, []);

  // Get related entity names
  const getProjectName = useCallback(
    (projectId: string | null | undefined): string => {
      if (!projectId) return 'Unassigned';
      const project = projects.find((p) => p.id === projectId);
      return project?.name || 'Unknown';
    },
    [projects],
  );

  const getEnvironmentName = useCallback(
    (environmentId: string | null | undefined): string => {
      if (!environmentId) return 'Unassigned';
      const env = environments.find((e) => e.id === environmentId);
      return env?.name || 'Unknown';
    },
    [environments],
  );

  const getInstanceName = useCallback(
    (instanceId: string): string => {
      const instance = databaseInstances.find((i) => i.id === instanceId);
      return instance?.name || 'Unknown';
    },
    [databaseInstances],
  );

  // Prepare filter options
  const projectOptions = useMemo(
    () =>
      projects.map((project) => ({
        label: project.name,
        value: project.id,
      })),
    [projects],
  );

  const environmentOptions = useMemo(
    () =>
      environments.map((environment) => ({
        label: environment.name,
        value: environment.id,
      })),
    [environments],
  );

  const instanceOptions = useMemo(
    () =>
      databaseInstances.map((instance) => ({
        label: instance.name,
        value: instance.id,
      })),
    [databaseInstances],
  );

  // Get columns for databases table
  const databasesColumns = useMemo(
    () =>
      getDatabasesColumns({
        onView: (database) => navigate(`/databases/${database.id}`),
        onEdit: openEditForm,
        onDelete: openDeleteConfirmation,
        canUpdate: canUpdateDatabases,
        canDelete: canDeleteDatabases,
        getProjectName,
        getEnvironmentName,
        getInstanceName,
      }),
    [
      navigate,
      openEditForm,
      openDeleteConfirmation,
      canUpdateDatabases,
      canDeleteDatabases,
      getProjectName,
      getEnvironmentName,
      getInstanceName,
    ],
  );

  // Calculate initial column filters
  const initialColumnFilters = useMemo(() => {
    const columnFilters: { id: string; value: unknown }[] = [];

    // Set default project if currentProject exists
    if (currentProject?.id) {
      columnFilters.push({ id: 'project', value: [currentProject.id] });
    }

    return columnFilters;
  }, [currentProject]);

  // Define filter definitions for the toolbar
  const filterDefinitions: FilterDefinition[] = useMemo(
    () => [
      {
        id: 'database_name',
        type: 'search',
        title: 'Database Name',
        placeholder: 'Search database names...',
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
        id: 'environment',
        type: 'faceted',
        title: 'Environment',
        options: environmentOptions,
      },
      {
        id: 'instance',
        type: 'faceted',
        title: 'Instance',
        options: instanceOptions,
      },
      {
        id: 'status',
        type: 'faceted',
        title: 'Status',
        options: [
          { label: 'Active', value: 'Active', icon: DatabaseIcon },
          { label: 'Inactive', value: 'Inactive', icon: DatabaseIcon },
          { label: 'Pending', value: 'Pending', icon: DatabaseIcon },
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
    [projectOptions, environmentOptions, instanceOptions],
  );

  // Render access denied
  if (!canReadDatabases) {
    return (
      <DefaultLayout breadcrumbs={[{ label: 'Databases', isCurrentPage: true }]}>
        <Card>
          <CardHeader>
            <CardTitle>Access Denied</CardTitle>
            <CardDescription>You do not have permission to access this page.</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              This page requires database read permissions. Please contact your administrator if you
              need access.
            </p>
          </CardContent>
        </Card>
      </DefaultLayout>
    );
  }

  return (
    <DefaultLayout breadcrumbs={[{ label: 'Databases', isCurrentPage: true }]}>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Databases</h1>
            <p className="text-muted-foreground mt-1">
              Manage databases and assign them to projects and environments
            </p>
          </div>
          {canCreateDatabases && (
            <Button onClick={openCreateForm}>
              <Plus className="mr-2 h-4 w-4" />
              Create Database
            </Button>
          )}
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

        {/* Databases Table */}
        <Card>
          <CardHeader>
            <CardTitle>Databases</CardTitle>
            <CardDescription>
              {isLoading ? (
                'Loading databases...'
              ) : (
                <>
                  {databases.length} database{databases.length === 1 ? '' : 's'}
                </>
              )}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {!isLoading && databases.length === 0 ? (
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <DatabaseIcon className="h-12 w-12 text-muted-foreground" />
                  </EmptyMedia>
                  <EmptyTitle>No Databases Yet</EmptyTitle>
                  <EmptyDescription>
                    Get started by creating your first database and assigning it to a project and
                    environment.
                  </EmptyDescription>
                </EmptyHeader>
                <EmptyContent>
                  {canCreateDatabases && (
                    <Button onClick={openCreateForm}>
                      <Plus className="mr-2 h-4 w-4" />
                      Create Database
                    </Button>
                  )}
                </EmptyContent>
              </Empty>
            ) : (
              <DataTable
                columns={databasesColumns}
                data={databases}
                isLoading={isLoading}
                initialColumnFilters={initialColumnFilters}
                emptyMessage="No databases match your filters. Try adjusting your search criteria."
                pagination={{
                  pageIndex,
                  pageSize,
                  totalCount: databases.length,
                  onPageChange: setPageIndex,
                  onPageSizeChange: (newSize) => {
                    setPageSize(newSize);
                    setPageIndex(0);
                  },
                  pageSizeOptions: [10, 20, 30, 50],
                }}
                onRowClick={(database) => navigate(`/databases/${database.id}`)}
              >
                {(table) => (
                  <DataTableToolbar table={table} filters={filterDefinitions}>
                    <DataTableViewOptions table={table} />
                  </DataTableToolbar>
                )}
              </DataTable>
            )}
          </CardContent>
        </Card>

        {/* Create Database Sheet */}
        <CreateDatabaseDialog open={isCreateOpen} onOpenChange={setIsCreateOpen} />

        {/* Edit Database Sheet */}
        <Sheet open={isEditOpen} onOpenChange={setIsEditOpen}>
          <SheetContent className="overflow-y-auto max-h-[90vh] md:min-w-[600px] sm:w-[540px]">
            <SheetHeader>
              <SheetTitle>Edit Database</SheetTitle>
              <SheetDescription>
                Update database information. Project, environment, and instance cannot be changed.
              </SheetDescription>
            </SheetHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <label htmlFor="edit-project" className="text-sm font-medium">
                  Project
                </label>
                <Input
                  id="edit-project"
                  value={selectedDatabase ? getProjectName(selectedDatabase.project_id) : ''}
                  disabled
                  className="opacity-50"
                />
              </div>
              <div className="space-y-2">
                <label htmlFor="edit-environment" className="text-sm font-medium">
                  Environment
                </label>
                <Input
                  id="edit-environment"
                  value={
                    selectedDatabase ? getEnvironmentName(selectedDatabase.environment_id) : ''
                  }
                  disabled
                  className="opacity-50"
                />
              </div>
              <div className="space-y-2">
                <label htmlFor="edit-instance" className="text-sm font-medium">
                  Database Instance
                </label>
                <Input
                  id="edit-instance"
                  value={
                    selectedDatabase ? getInstanceName(selectedDatabase.database_instance_id) : ''
                  }
                  disabled
                  className="opacity-50"
                />
              </div>
              <div className="space-y-2">
                <label htmlFor="edit-name" className="text-sm font-medium">
                  Database Name *
                </label>
                <Input
                  id="edit-name"
                  value={formData.database_name}
                  onChange={(e) => setFormData({ ...formData, database_name: e.target.value })}
                  placeholder="Enter database name"
                  aria-invalid={!!formErrors.database_name}
                />
                {formErrors.database_name && (
                  <p className="text-sm text-destructive">{formErrors.database_name}</p>
                )}
              </div>
              <div className="space-y-2">
                <label htmlFor="edit-description" className="text-sm font-medium">
                  Description
                </label>
                <textarea
                  id="edit-description"
                  value={formData.description || ''}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="Enter database description"
                  className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                />
              </div>
            </div>
            <SheetFooter>
              <Button variant="outline" onClick={() => setIsEditOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleEdit} disabled={isSubmitting}>
                {isSubmitting ? 'Saving...' : 'Save Changes'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>

        {/* Delete Confirmation Sheet */}
        <Sheet open={isDeleteOpen} onOpenChange={setIsDeleteOpen}>
          <SheetContent>
            <SheetHeader>
              <SheetTitle>Delete Database</SheetTitle>
              <SheetDescription>
                Are you sure you want to delete this database? This action cannot be undone.
              </SheetDescription>
            </SheetHeader>
            <div className="py-4">
              {selectedDatabase && (
                <div className="space-y-2">
                  <p className="text-sm font-medium">{selectedDatabase.database_name}</p>
                  <p className="text-sm text-muted-foreground">
                    {selectedDatabase.description || 'No description'}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    Project: {getProjectName(selectedDatabase.project_id)}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    Environment: {getEnvironmentName(selectedDatabase.environment_id)}
                  </p>
                </div>
              )}
            </div>
            <SheetFooter>
              <Button variant="outline" onClick={() => setIsDeleteOpen(false)}>
                Cancel
              </Button>
              <Button variant="destructive" onClick={handleDelete} disabled={isSubmitting}>
                {isSubmitting ? 'Deleting...' : 'Delete Database'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>
      </div>
    </DefaultLayout>
  );
};
