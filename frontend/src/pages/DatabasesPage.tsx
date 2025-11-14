import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { databasesService } from '@/lib/api/services/databases';
import { projectsService } from '@/lib/api/services/projects';
import { environmentsService } from '@/lib/api/services/environments';
import { databaseInstancesService } from '@/lib/api/services/database-instances';
import { CreateDatabaseDialog } from '@/components/CreateDatabaseDialog';
import { hasOrganizationPermission, OrganizationPermission } from '@/lib/rbac';
import { DatabaseStatusBadge } from '@/components/DatabaseStatusBadge';
import type {
  CreateDatabaseRequest,
  Database,
  UpdateDatabaseRequest,
  Project,
  Environment,
  DatabaseInstance,
} from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
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
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Plus,
  Pencil,
  Trash2,
  AlertTriangle,
  Database as DatabaseIcon,
  Search,
  X,
  Eye,
} from 'lucide-react';
import { useCurrentProject } from '@/store';

export const DatabasesPage: React.FC = () => {
  const { user } = useAuth();
  const currentProject = useCurrentProject();
  const navigate = useNavigate();
  const [databases, setDatabases] = useState<Database[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [environments, setEnvironments] = useState<Environment[]>([]);
  const [databaseInstances, setDatabaseInstances] = useState<DatabaseInstance[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingDropdowns, setIsLoadingDropdowns] = useState(false);
  const [error, setError] = useState('');
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [selectedDatabase, setSelectedDatabase] = useState<Database | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Filter state - initialize with currentProject if available
  const [filters, setFilters] = useState<{
    search?: string;
    project_id?: string;
    environment_id?: string;
    database_instance_id?: string;
    status?: string;
  }>({
    project_id: currentProject?.id,
  });

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

  // Load dropdown data
  useEffect(() => {
    if (!canReadDatabases) {
      return;
    }

    const loadDropdownData = async () => {
      try {
        setIsLoadingDropdowns(true);
        const [projectsRes, environmentsRes, instancesRes] = await Promise.all([
          projectsService.listProjects({ limit: 100 }),
          environmentsService.listEnvironments({ limit: 100 }),
          databaseInstancesService.listDatabaseInstances({ limit: 100 }),
        ]);
        setProjects(projectsRes.items);
        setEnvironments(environmentsRes.items);
        setDatabaseInstances(instancesRes.items);
      } catch (err) {
        console.error('Failed to load dropdown data:', err);
      } finally {
        setIsLoadingDropdowns(false);
      }
    };

    loadDropdownData();
  }, [canReadDatabases]);

  // Sync filters with currentProject on initial load (only if filter not already set)
  useEffect(() => {
    if (currentProject?.id && !filters.project_id) {
      setFilters((prev) => ({ ...prev, project_id: currentProject.id }));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentProject]);

  // Load databases
  useEffect(() => {
    if (!canReadDatabases) {
      setError('Access denied. You do not have permission to view databases.');
      setIsLoading(false);
      return;
    }

    const loadDatabases = async () => {
      try {
        setIsLoading(true);
        setError('');
        const response = await databasesService.listDatabases({ limit: 100 });
        setDatabases(response.items);
      } catch (err) {
        if (err instanceof APIError) {
          setError(err.message);
        } else {
          setError('Failed to load databases. Please try again.');
        }
        console.error('Failed to load databases:', err);
      } finally {
        setIsLoading(false);
      }
    };

    loadDatabases();
  }, [canReadDatabases]);

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
      // Reload databases
      const response = await databasesService.listDatabases({ limit: 100 });
      setDatabases(response.items);
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
      // Reload databases
      const response = await databasesService.listDatabases({ limit: 100 });
      setDatabases(response.items);
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
  const openEditForm = (database: Database) => {
    setSelectedDatabase(database);
    setFormData({
      project_id: database.project_id,
      database_instance_id: database.database_instance_id,
      environment_id: database.environment_id,
      database_name: database.database_name,
      description: database.description || '',
    });
    setFormErrors({});
    setIsEditOpen(true);
  };

  // Open delete confirmation
  const openDeleteConfirmation = (database: Database) => {
    setSelectedDatabase(database);
    setIsDeleteOpen(true);
  };

  // Format date
  const formatDate = (dateString: string) => {
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

  // Get related entity names
  const getProjectName = (projectId: string): string => {
    const project = projects.find((p) => p.id === projectId);
    return project?.name || 'Unknown';
  };

  const getEnvironmentName = (environmentId: string): string => {
    const env = environments.find((e) => e.id === environmentId);
    return env?.name || 'Unknown';
  };

  const getInstanceName = (instanceId: string): string => {
    const instance = databaseInstances.find((i) => i.id === instanceId);
    return instance?.name || 'Unknown';
  };

  // Handle filter changes
  const handleFilterChange = (key: keyof typeof filters, value: string | undefined) => {
    setFilters((prev) => {
      const updated = { ...prev };
      if (value && value.trim()) {
        updated[key] = value;
      } else {
        delete updated[key];
      }
      return updated;
    });
  };

  // Clear all filters
  const clearFilters = () => {
    setFilters({});
  };

  // Filter databases
  const filteredDatabases = databases.filter((database) => {
    // Search filter
    if (filters.search) {
      const searchLower = filters.search.toLowerCase();
      const matchesSearch =
        database.database_name.toLowerCase().includes(searchLower) ||
        (database.description && database.description.toLowerCase().includes(searchLower));
      if (!matchesSearch) return false;
    }

    // Project filter
    if (filters.project_id && database.project_id !== filters.project_id) {
      return false;
    }

    // Environment filter
    if (filters.environment_id && database.environment_id !== filters.environment_id) {
      return false;
    }

    // Instance filter
    if (
      filters.database_instance_id &&
      database.database_instance_id !== filters.database_instance_id
    ) {
      return false;
    }

    // Status filter
    if (filters.status && database.status !== filters.status) {
      return false;
    }

    return true;
  });

  const hasFilters = Object.keys(filters).length > 0;

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

        {/* Filters */}
        <Card>
          <CardHeader>
            <CardTitle>Filters</CardTitle>
            <CardDescription>
              Filter databases by name, project, environment, instance, or status
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
              {/* Search Filter */}
              <div className="space-y-2">
                <label htmlFor="filter-search" className="text-sm font-medium">
                  Search
                </label>
                <div className="relative">
                  <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    id="filter-search"
                    placeholder="Search databases..."
                    value={filters.search || ''}
                    onChange={(e) => handleFilterChange('search', e.target.value)}
                    className="pl-9"
                  />
                </div>
              </div>

              {/* Project Filter */}
              <div className="space-y-2">
                <label htmlFor="filter-project" className="text-sm font-medium">
                  Project
                </label>
                {isLoadingDropdowns ? (
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

              {/* Environment Filter */}
              <div className="space-y-2">
                <label htmlFor="filter-environment" className="text-sm font-medium">
                  Environment
                </label>
                {isLoadingDropdowns ? (
                  <Skeleton className="h-9 w-full" />
                ) : (
                  <Select
                    value={filters.environment_id || 'all'}
                    onValueChange={(value) =>
                      handleFilterChange('environment_id', value === 'all' ? undefined : value)
                    }
                  >
                    <SelectTrigger id="filter-environment">
                      <SelectValue placeholder="All Environments" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Environments</SelectItem>
                      {environments.map((env) => (
                        <SelectItem key={env.id} value={env.id}>
                          {env.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              </div>

              {/* Instance Filter */}
              <div className="space-y-2">
                <label htmlFor="filter-instance" className="text-sm font-medium">
                  Instance
                </label>
                {isLoadingDropdowns ? (
                  <Skeleton className="h-9 w-full" />
                ) : (
                  <Select
                    value={filters.database_instance_id || 'all'}
                    onValueChange={(value) =>
                      handleFilterChange(
                        'database_instance_id',
                        value === 'all' ? undefined : value,
                      )
                    }
                  >
                    <SelectTrigger id="filter-instance">
                      <SelectValue placeholder="All Instances" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Instances</SelectItem>
                      {databaseInstances.map((instance) => (
                        <SelectItem key={instance.id} value={instance.id}>
                          {instance.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              </div>

              {/* Status Filter */}
              <div className="space-y-2">
                <label htmlFor="filter-status" className="text-sm font-medium">
                  Status
                </label>
                <Select
                  value={filters.status || 'all'}
                  onValueChange={(value) =>
                    handleFilterChange('status', value === 'all' ? undefined : value)
                  }
                >
                  <SelectTrigger id="filter-status">
                    <SelectValue placeholder="All Statuses" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Statuses</SelectItem>
                    <SelectItem value="Active">Active</SelectItem>
                    <SelectItem value="Inactive">Inactive</SelectItem>
                    <SelectItem value="Pending">Pending</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            {/* Clear Filters Button */}
            {hasFilters && (
              <div className="mt-4">
                <Button variant="outline" size="sm" onClick={clearFilters}>
                  <X className="mr-2 h-4 w-4" />
                  Clear Filters
                </Button>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Databases list */}
        {isLoading ? (
          <Card>
            <CardContent className="pt-6">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Database Name</TableHead>
                    <TableHead>Project</TableHead>
                    <TableHead>Environment</TableHead>
                    <TableHead>Instance</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {[1, 2, 3].map((i) => (
                    <TableRow key={i}>
                      <TableCell>
                        <Skeleton className="h-4 w-32" />
                      </TableCell>
                      <TableCell>
                        <Skeleton className="h-4 w-24" />
                      </TableCell>
                      <TableCell>
                        <Skeleton className="h-4 w-24" />
                      </TableCell>
                      <TableCell>
                        <Skeleton className="h-4 w-32" />
                      </TableCell>
                      <TableCell>
                        <Skeleton className="h-4 w-20" />
                      </TableCell>
                      <TableCell>
                        <Skeleton className="h-4 w-24" />
                      </TableCell>
                      <TableCell className="text-right">
                        <Skeleton className="h-8 w-16 ml-auto" />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        ) : filteredDatabases.length === 0 ? (
          <Card>
            <CardContent className="pt-6">
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <DatabaseIcon className="h-12 w-12 text-muted-foreground" />
                  </EmptyMedia>
                  <EmptyTitle>{hasFilters ? 'No Databases Found' : 'No Databases Yet'}</EmptyTitle>
                  <EmptyDescription>
                    {hasFilters
                      ? 'No databases match your filters. Try adjusting your search criteria.'
                      : 'Get started by creating your first database and assigning it to a project and environment.'}
                  </EmptyDescription>
                </EmptyHeader>
                <EmptyContent>
                  {hasFilters ? (
                    <Button variant="outline" onClick={clearFilters}>
                      <X className="mr-2 h-4 w-4" />
                      Clear Filters
                    </Button>
                  ) : canCreateDatabases ? (
                    <Button onClick={openCreateForm}>
                      <Plus className="mr-2 h-4 w-4" />
                      Create Database
                    </Button>
                  ) : null}
                </EmptyContent>
              </Empty>
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardHeader>
              <CardTitle>Databases</CardTitle>
              <CardDescription>
                {filteredDatabases.length} of {databases.length} database
                {databases.length === 1 ? '' : 's'}
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-6">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Database Name</TableHead>
                    <TableHead>Project</TableHead>
                    <TableHead>Environment</TableHead>
                    <TableHead>Instance</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredDatabases.map((database) => (
                    <TableRow
                      key={database.id}
                      onClick={() => navigate(`/databases/${database.id}`)}
                      className="cursor-pointer hover:bg-muted/50 transition-colors"
                    >
                      <TableCell className="font-medium">{database.database_name}</TableCell>
                      <TableCell className="text-muted-foreground">
                        {getProjectName(database.project_id)}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {getEnvironmentName(database.environment_id)}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {getInstanceName(database.database_instance_id)}
                      </TableCell>
                      <TableCell>
                        <DatabaseStatusBadge status={database.status} variant="database" />
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {formatDate(database.created_at)}
                      </TableCell>
                      <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                        <div className="flex items-center justify-end gap-2">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => navigate(`/databases/${database.id}`)}
                            className="h-8 w-8 p-0"
                          >
                            <Eye className="h-4 w-4" />
                          </Button>
                          {canUpdateDatabases && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => openEditForm(database)}
                              className="h-8 w-8 p-0"
                            >
                              <Pencil className="h-4 w-4" />
                            </Button>
                          )}
                          {canDeleteDatabases && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => openDeleteConfirmation(database)}
                              className="h-8 w-8 p-0 text-destructive hover:text-destructive"
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        )}

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
