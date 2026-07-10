import React, { useEffect, useState, useMemo, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { subDays } from 'date-fns';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { databasesService } from '@/lib/api/services/databases';
import { projectsService } from '@/lib/api/services/projects';
import { environmentsService } from '@/lib/api/services/environments';
import { databaseInstancesService } from '@/lib/api/services/database-instances';
import { CreateDatabaseDialog } from '@/components/CreateDatabaseDialog';
import { hasOrganizationPermission, OrganizationPermission } from '@/lib/rbac';
import { useProjectsSWR } from '@/hooks/api/useProjectsSWR';
import { useDatabases } from '@/hooks/api/useDatabases';
import { useEnvironmentsSWR } from '@/hooks/api/useEnvironmentsSWR';
import { useDatabaseInstancesSWR } from '@/hooks/api/useDatabaseInstancesSWR';
import { REFERENCE_LIST_LIMIT } from '@/lib/api/pagination';
import {
  useServerTableFilters,
  type ServerTableFilterMapping,
} from '@/hooks/useServerTableFilters';
import type { CreateDatabaseRequest, Database, UpdateDatabaseRequest } from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
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

const DATABASES_FILTER_MAPPING: ServerTableFilterMapping = {
  searchColumnIds: ['database_name'],
  equals: {
    project: 'project_id',
    environment: 'environment_id',
    instance: 'database_instance_id',
    status: 'status',
  },
  dateRanges: { created_at: ['created_after', 'created_before'] },
};

// Filter definitions are static: the reference-backed filters (project /
// environment / instance) search the server as you type instead of preloading
// a full option list, so they scale to any number of entities.
const DATABASES_FILTER_DEFINITIONS: FilterDefinition[] = [
  {
    id: 'database_name',
    type: 'search',
    title: 'Database Name',
    placeholder: 'Search database names...',
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
    id: 'environment',
    type: 'async',
    title: 'Environment',
    placeholder: 'Search environments...',
    loadOptions: (search) =>
      environmentsService
        .listEnvironments({ search, limit: 20 })
        .then((r) => r.items.map((e) => ({ label: e.name, value: e.id }))),
    resolveLabel: (id) =>
      environmentsService
        .getEnvironment(id)
        .then((e) => e.name)
        .catch(() => undefined),
  },
  {
    id: 'instance',
    type: 'async',
    title: 'Instance',
    placeholder: 'Search instances...',
    loadOptions: (search) =>
      databaseInstancesService
        .listDatabaseInstances({ search, limit: 20 })
        .then((r) => r.items.map((i) => ({ label: i.name, value: i.id }))),
    resolveLabel: (id) =>
      databaseInstancesService
        .getDatabaseInstance(id)
        .then((i) => i.name)
        .catch(() => undefined),
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
      { label: 'Last 7 days', getValue: () => [subDays(new Date(), 7), new Date()] },
      { label: 'Last 30 days', getValue: () => [subDays(new Date(), 30), new Date()] },
      { label: 'Last 90 days', getValue: () => [subDays(new Date(), 90), new Date()] },
    ],
  },
];

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

  // Default to the current project when one is selected (async filter => string value).
  const defaultColumnFilters = useMemo(
    () => (currentProject ? [{ id: 'project', value: currentProject.id }] : []),
    [currentProject],
  );

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
    mapping: DATABASES_FILTER_MAPPING,
    filterDefinitions: DATABASES_FILTER_DEFINITIONS,
    initialPageSize: 50,
    initialColumnFilters: defaultColumnFilters,
  });

  // Reference lists are used to resolve related names in the table and to
  // populate the edit-form dropdowns. Filter options are fetched server-side
  // (async), so these are only for display/name-resolution of the current page.
  const { projects, isLoading: projectsLoading } = useProjectsSWR({ limit: REFERENCE_LIST_LIMIT });
  const {
    databases,
    total,
    isLoading: databasesLoading,
    refetch: refetchDatabases,
  } = useDatabases(params);
  const { environments } = useEnvironmentsSWR({ limit: REFERENCE_LIST_LIMIT });
  const { databaseInstances } = useDatabaseInstancesSWR({ limit: REFERENCE_LIST_LIMIT });

  const isLoading = projectsLoading || databasesLoading;

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

    // For edit mode, project_id and environment_id are only required if they are currently unassigned
    if (isEditOpen && selectedDatabase) {
      // Only validate project_id if it's currently unassigned
      if (!selectedDatabase.project_id && !formData.project_id) {
        errors.project_id = 'Project is required';
      }
      // Only validate environment_id if it's currently unassigned
      if (!selectedDatabase.environment_id && !formData.environment_id) {
        errors.environment_id = 'Environment is required';
      }
    } else {
      // For create mode, both are required
      if (!formData.project_id) {
        errors.project_id = 'Project is required';
      }
      if (!formData.environment_id) {
        errors.environment_id = 'Environment is required';
      }
    }

    if (!formData.database_instance_id) {
      errors.database_instance_id = 'Database instance is required';
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

      // Only allow updating project_id if it's currently unassigned (null/undefined)
      if (
        !selectedDatabase.project_id &&
        formData.project_id &&
        formData.project_id !== selectedDatabase.project_id
      ) {
        updateData.project_id = formData.project_id;
      }

      // Only allow updating environment_id if it's currently unassigned (null/undefined)
      if (
        !selectedDatabase.environment_id &&
        formData.environment_id &&
        formData.environment_id !== selectedDatabase.environment_id
      ) {
        updateData.environment_id = formData.environment_id;
      }

      // Description can always be updated
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

  // Check if any server-side filters are active (anything beyond pagination)
  const hasActiveFilters = useMemo(
    () => Object.keys(params).some((key) => key !== 'limit' && key !== 'offset'),
    [params],
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
            {!isLoading && databases.length === 0 && !hasActiveFilters ? (
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
                  totalCount: total,
                  onPageChange: setPageIndex,
                  onPageSizeChange: setPageSize,
                  pageSizeOptions: [10, 20, 50, 100],
                  manualFiltering: true,
                }}
                onRowClick={(database) => navigate(`/databases/${database.id}`)}
                onFiltersChange={onFiltersChange}
              >
                {(table) => (
                  <DataTableToolbar table={table} filters={DATABASES_FILTER_DEFINITIONS}>
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
                Update database project, environment, and description. Instance and name cannot be
                changed.
              </SheetDescription>
            </SheetHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <label htmlFor="edit-project" className="text-sm font-medium">
                  Project *
                </label>
                <Select
                  value={formData.project_id}
                  onValueChange={(value) => setFormData({ ...formData, project_id: value })}
                  disabled={
                    isSubmitting ||
                    (selectedDatabase?.project_id !== null &&
                      selectedDatabase?.project_id !== undefined)
                  }
                >
                  <SelectTrigger id="edit-project">
                    <SelectValue placeholder="Select a project" />
                  </SelectTrigger>
                  <SelectContent>
                    {projects.map((project) => (
                      <SelectItem key={project.id} value={project.id}>
                        {project.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {formErrors.project_id && (
                  <p className="text-sm text-destructive">{formErrors.project_id}</p>
                )}
                {selectedDatabase?.project_id ? (
                  <p className="text-xs text-muted-foreground">
                    Project cannot be changed after it has been assigned
                  </p>
                ) : (
                  <p className="text-xs text-muted-foreground">
                    Project can only be set once. Once assigned, it cannot be changed.
                  </p>
                )}
              </div>
              <div className="space-y-2">
                <label htmlFor="edit-environment" className="text-sm font-medium">
                  Environment *
                </label>
                <Select
                  value={formData.environment_id}
                  onValueChange={(value) => setFormData({ ...formData, environment_id: value })}
                  disabled={
                    isSubmitting ||
                    (selectedDatabase?.environment_id !== null &&
                      selectedDatabase?.environment_id !== undefined)
                  }
                >
                  <SelectTrigger id="edit-environment">
                    <SelectValue placeholder="Select an environment" />
                  </SelectTrigger>
                  <SelectContent>
                    {environments.map((env) => (
                      <SelectItem key={env.id} value={env.id}>
                        {env.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {formErrors.environment_id && (
                  <p className="text-sm text-destructive">{formErrors.environment_id}</p>
                )}
                {selectedDatabase?.environment_id ? (
                  <p className="text-xs text-muted-foreground">
                    Environment cannot be changed after it has been assigned
                  </p>
                ) : (
                  <p className="text-xs text-muted-foreground">
                    Environment can only be set once. Once assigned, it cannot be changed.
                  </p>
                )}
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
                  className="bg-muted"
                />
                <p className="text-xs text-muted-foreground">
                  Database instance cannot be changed after creation
                </p>
              </div>
              <div className="space-y-2">
                <label htmlFor="edit-name" className="text-sm font-medium">
                  Database Name
                </label>
                <Input
                  id="edit-name"
                  value={formData.database_name}
                  disabled
                  className="bg-muted"
                />
                <p className="text-xs text-muted-foreground">
                  Database name cannot be changed after creation
                </p>
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
                  disabled={isSubmitting}
                />
              </div>
            </div>
            <SheetFooter>
              <Button
                variant="outline"
                onClick={() => setIsEditOpen(false)}
                disabled={isSubmitting}
              >
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
