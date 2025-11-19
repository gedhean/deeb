import React, { useState, useMemo, useCallback } from 'react';
import { subDays } from 'date-fns';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { hasOrganizationPermission, OrganizationPermission } from '@/lib/rbac';
import type {
  CreateEnvironmentRequest,
  UpdateEnvironmentRequest,
  Environment,
} from '@/lib/api/types';
import {
  useEnvironmentsSWR,
  useCreateEnvironmentSWR,
  useUpdateEnvironmentSWR,
  useDeleteEnvironmentSWR,
} from '@/hooks/api/useEnvironmentsSWR';
import { usePolicies } from '@/hooks/api/usePoliciesSWR';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
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
import { DataTable, DataTableToolbar } from '@/components/data-table';
import type { FilterDefinition } from '@/components/data-table/types';
import { getEnvironmentsColumns } from './environments/columns';
import { Plus, AlertTriangle, Globe } from 'lucide-react';

export const EnvironmentsPage: React.FC = () => {
  const { user } = useAuth();
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [selectedEnvironment, setSelectedEnvironment] = useState<Environment | null>(null);

  // Pagination state
  const [pageIndex, setPageIndex] = useState(0);
  const [pageSize, setPageSize] = useState(10);

  // Form state
  const [formData, setFormData] = useState<CreateEnvironmentRequest>({
    name: '',
    description: '',
    policy_id: undefined,
    allow_self_approval: false,
  });
  const [formErrors, setFormErrors] = useState<{
    name?: string;
    description?: string;
    policy_id?: string;
  }>({});

  // Check permissions
  const canRead = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.EnvironmentsRead,
  );
  const canCreate = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.EnvironmentsCreate,
  );
  const canUpdate = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.EnvironmentsUpdate,
  );
  const canDelete = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.EnvironmentsDelete,
  );

  // Use SWR hooks for data fetching
  const { environments, isLoading, error } = useEnvironmentsSWR();
  const { policies, isLoading: isLoadingPolicies } = usePolicies({
    type: 'SQL_REVIEW',
    limit: 100,
    enabled: canRead,
  });

  // Use mutation hooks
  const { createEnvironment, isLoading: isCreating } = useCreateEnvironmentSWR();
  const { updateEnvironment, isLoading: isUpdating } = useUpdateEnvironmentSWR();
  const { deleteEnvironment, isLoading: isDeleting } = useDeleteEnvironmentSWR();

  // Validate form
  const validateForm = (): boolean => {
    const errors: { name?: string; description?: string; policy_id?: string } = {};

    if (!formData.name.trim()) {
      errors.name = 'Environment name is required';
    } else if (formData.name.trim().length < 2) {
      errors.name = 'Environment name must be at least 2 characters';
    }

    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  // Handle create
  const handleCreate = async () => {
    if (!validateForm()) {
      return;
    }

    const createData: CreateEnvironmentRequest = {
      name: formData.name.trim(),
      description: formData.description?.trim() || undefined,
      policy_id: formData.policy_id || undefined,
      allow_self_approval: formData.allow_self_approval,
    };

    const result = await createEnvironment(createData);
    if (result) {
      setIsCreateOpen(false);
      setFormData({ name: '', description: '', policy_id: undefined, allow_self_approval: false });
      setFormErrors({});
    }
  };

  // Handle edit
  const handleEdit = async () => {
    if (!selectedEnvironment || !validateForm()) {
      return;
    }

    const updateData: UpdateEnvironmentRequest = {};
    if (formData.name !== selectedEnvironment.name) {
      updateData.name = formData.name.trim();
    }
    if (formData.description !== selectedEnvironment.description) {
      updateData.description = formData.description?.trim() || undefined;
    }
    // Always include policy_id if it changed, even when clearing to null
    if (formData.policy_id !== selectedEnvironment.policy_id) {
      updateData.policy_id = formData.policy_id || null;
    }
    if (formData.allow_self_approval !== selectedEnvironment.allow_self_approval) {
      updateData.allow_self_approval = formData.allow_self_approval;
    }

    const result = await updateEnvironment(selectedEnvironment.id, updateData);
    if (result) {
      setIsEditOpen(false);
      setSelectedEnvironment(null);
      setFormData({ name: '', description: '', policy_id: undefined, allow_self_approval: false });
      setFormErrors({});
    }
  };

  // Handle delete
  const handleDelete = async () => {
    if (!selectedEnvironment) {
      return;
    }

    const success = await deleteEnvironment(selectedEnvironment.id);
    if (success) {
      setIsDeleteOpen(false);
      setSelectedEnvironment(null);
    }
  };

  // Open create form
  const openCreateForm = useCallback(() => {
    setFormData({ name: '', description: '', policy_id: undefined, allow_self_approval: false });
    setFormErrors({});
    setIsCreateOpen(true);
  }, []);

  // Open edit form
  const openEditForm = useCallback((environment: Environment) => {
    setSelectedEnvironment(environment);
    setFormData({
      name: environment.name,
      description: environment.description || '',
      policy_id: environment.policy_id || undefined,
      allow_self_approval: environment.allow_self_approval || false,
    });
    setFormErrors({});
    setIsEditOpen(true);
  }, []);

  // Open delete confirmation
  const openDeleteConfirmation = useCallback((environment: Environment) => {
    setSelectedEnvironment(environment);
    setIsDeleteOpen(true);
  }, []);

  // Get policy name by ID
  const getPolicyName = useCallback(
    (policyId?: string): string => {
      if (!policyId) {
        return 'None';
      }
      const policy = policies.find((p) => p.id === policyId);
      return policy?.name || 'Unknown';
    },
    [policies],
  );

  // Get columns with action handlers
  const columns = getEnvironmentsColumns({
    onEdit: openEditForm,
    onDelete: openDeleteConfirmation,
    canUpdate: canUpdate,
    canDelete: canDelete,
    getPolicyName,
  });

  // Create policy filter options
  const policyOptions = useMemo(
    () =>
      policies.map((policy) => ({
        label: policy.name,
        value: policy.id,
      })),
    [policies],
  );

  // Create filter definitions
  const filterDefinitions: FilterDefinition[] = useMemo(
    () => [
      {
        id: 'name',
        type: 'search',
        title: 'Name',
        placeholder: 'Search names...',
        defaultActive: true,
      },
      {
        id: 'policy',
        type: 'faceted',
        title: 'Policy',
        options: policyOptions,
      },
      {
        id: 'allow_self_approval',
        type: 'faceted',
        title: 'Self Approval',
        options: [
          { label: 'Enabled', value: 'true' },
          { label: 'Disabled', value: 'false' },
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
    [policyOptions],
  );

  // Render access denied if user doesn't have read permission
  if (!canRead) {
    return (
      <DefaultLayout breadcrumbs={[{ label: 'Environments', isCurrentPage: true }]}>
        <Card>
          <CardHeader>
            <CardTitle>Access Denied</CardTitle>
            <CardDescription>You do not have permission to access this page.</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              This page requires the environments.read permission.
            </p>
          </CardContent>
        </Card>
      </DefaultLayout>
    );
  }

  return (
    <DefaultLayout breadcrumbs={[{ label: 'Environments', isCurrentPage: true }]}>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Environments</h1>
            <p className="text-muted-foreground mt-1">
              Manage environments and assign policies for database change requests
            </p>
          </div>
          {canCreate && (
            <Button onClick={openCreateForm}>
              <Plus className="mr-2 h-4 w-4" />
              Create Environment
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

        {/* Environments list */}
        {!isLoading && environments.length === 0 ? (
          <Card>
            <CardContent className="pt-6">
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <Globe className="h-12 w-12 text-muted-foreground" />
                  </EmptyMedia>
                  <EmptyTitle>No Environments Yet</EmptyTitle>
                  <EmptyDescription>
                    Get started by creating your first environment (e.g., Dev, Staging, Production).
                  </EmptyDescription>
                </EmptyHeader>
                <EmptyContent>
                  {canCreate && (
                    <Button onClick={openCreateForm}>
                      <Plus className="mr-2 h-4 w-4" />
                      Create Environment
                    </Button>
                  )}
                </EmptyContent>
              </Empty>
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardContent className="pt-6">
              <DataTable
                columns={columns}
                data={environments}
                isLoading={isLoading}
                emptyMessage="No environments found."
                pagination={{
                  pageIndex,
                  pageSize,
                  totalCount: environments.length,
                  onPageChange: setPageIndex,
                  onPageSizeChange: (newSize) => {
                    setPageSize(newSize);
                    setPageIndex(0);
                  },
                  pageSizeOptions: [10, 20, 30, 50],
                }}
              >
                {(table) => <DataTableToolbar table={table} filters={filterDefinitions} />}
              </DataTable>
            </CardContent>
          </Card>
        )}

        {/* Create Environment Sheet */}
        <Sheet open={isCreateOpen} onOpenChange={setIsCreateOpen}>
          <SheetContent className="w-[400px] sm:w-[540px] min-w-[600px]">
            <SheetHeader>
              <SheetTitle>Create Environment</SheetTitle>
              <SheetDescription>
                Create a new environment (e.g., Dev, Staging, Production) and optionally assign a
                policy.
              </SheetDescription>
            </SheetHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <label htmlFor="create-name" className="text-sm font-medium">
                  Environment Name *
                </label>
                <Input
                  id="create-name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Enter environment name (e.g., Production)"
                  aria-invalid={!!formErrors.name}
                />
                {formErrors.name && <p className="text-sm text-destructive">{formErrors.name}</p>}
              </div>
              <div className="space-y-2">
                <label htmlFor="create-description" className="text-sm font-medium">
                  Description
                </label>
                <textarea
                  id="create-description"
                  value={formData.description || ''}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="Enter environment description"
                  className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                  aria-invalid={!!formErrors.description}
                />
                {formErrors.description && (
                  <p className="text-sm text-destructive">{formErrors.description}</p>
                )}
              </div>
              <div className="space-y-2">
                <label htmlFor="create-policy" className="text-sm font-medium">
                  Policy (SQL_REVIEW)
                </label>
                {isLoadingPolicies ? (
                  <Skeleton className="h-9 w-full" />
                ) : (
                  <select
                    id="create-policy"
                    value={formData.policy_id || ''}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        policy_id: e.target.value || undefined,
                      })
                    }
                    className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <option value="">None</option>
                    {policies.map((policy) => (
                      <option key={policy.id} value={policy.id}>
                        {policy.name}
                      </option>
                    ))}
                  </select>
                )}
                {formErrors.policy_id && (
                  <p className="text-sm text-destructive">{formErrors.policy_id}</p>
                )}
              </div>
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="create-allow-self-approval"
                  checked={formData.allow_self_approval}
                  onCheckedChange={(checked) =>
                    setFormData({
                      ...formData,
                      allow_self_approval: checked === true,
                    })
                  }
                />
                <label
                  htmlFor="create-allow-self-approval"
                  className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                >
                  Allow Self Approval
                </label>
              </div>
              <p className="text-xs text-muted-foreground">
                When enabled, developers can approve their own change plans without requiring a
                separate reviewer. Useful for dev/staging environments.
              </p>
            </div>
            <SheetFooter>
              <Button variant="outline" onClick={() => setIsCreateOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleCreate} disabled={isCreating}>
                {isCreating ? 'Creating...' : 'Create Environment'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>

        {/* Edit Environment Sheet */}
        <Sheet open={isEditOpen} onOpenChange={setIsEditOpen}>
          <SheetContent className="min-w-[600px] sm:w-[540px]">
            <SheetHeader>
              <SheetTitle>Edit Environment</SheetTitle>
              <SheetDescription>
                Update environment information and policy assignment.
              </SheetDescription>
            </SheetHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <label htmlFor="edit-name" className="text-sm font-medium">
                  Environment Name *
                </label>
                <Input
                  id="edit-name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Enter environment name"
                  aria-invalid={!!formErrors.name}
                />
                {formErrors.name && <p className="text-sm text-destructive">{formErrors.name}</p>}
              </div>
              <div className="space-y-2">
                <label htmlFor="edit-description" className="text-sm font-medium">
                  Description
                </label>
                <textarea
                  id="edit-description"
                  value={formData.description || ''}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="Enter environment description"
                  className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                  aria-invalid={!!formErrors.description}
                />
                {formErrors.description && (
                  <p className="text-sm text-destructive">{formErrors.description}</p>
                )}
              </div>
              <div className="space-y-2">
                <label htmlFor="edit-policy" className="text-sm font-medium">
                  Policy (SQL_REVIEW)
                </label>
                {isLoadingPolicies ? (
                  <Skeleton className="h-9 w-full" />
                ) : (
                  <select
                    id="edit-policy"
                    value={formData.policy_id || ''}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        policy_id: e.target.value || undefined,
                      })
                    }
                    className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <option value="">None</option>
                    {policies.map((policy) => (
                      <option key={policy.id} value={policy.id}>
                        {policy.name}
                      </option>
                    ))}
                  </select>
                )}
                {formErrors.policy_id && (
                  <p className="text-sm text-destructive">{formErrors.policy_id}</p>
                )}
              </div>
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="edit-allow-self-approval"
                  checked={formData.allow_self_approval}
                  onCheckedChange={(checked) =>
                    setFormData({
                      ...formData,
                      allow_self_approval: checked === true,
                    })
                  }
                />
                <label
                  htmlFor="edit-allow-self-approval"
                  className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                >
                  Allow Self Approval
                </label>
              </div>
              <p className="text-xs text-muted-foreground">
                When enabled, developers can approve their own change plans without requiring a
                separate reviewer. Useful for dev/staging environments.
              </p>
            </div>
            <SheetFooter>
              <Button variant="outline" onClick={() => setIsEditOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleEdit} disabled={isUpdating}>
                {isUpdating ? 'Saving...' : 'Save Changes'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>

        {/* Delete Confirmation Sheet */}
        <Sheet open={isDeleteOpen} onOpenChange={setIsDeleteOpen}>
          <SheetContent>
            <SheetHeader>
              <SheetTitle>Delete Environment</SheetTitle>
              <SheetDescription>
                Are you sure you want to delete this environment? This action cannot be undone.
              </SheetDescription>
            </SheetHeader>
            <div className="py-4">
              {selectedEnvironment && (
                <div className="space-y-2">
                  <p className="text-sm font-medium">{selectedEnvironment.name}</p>
                  <p className="text-sm text-muted-foreground">
                    {selectedEnvironment.description || 'No description'}
                  </p>
                </div>
              )}
            </div>
            <SheetFooter>
              <Button variant="outline" onClick={() => setIsDeleteOpen(false)}>
                Cancel
              </Button>
              <Button variant="destructive" onClick={handleDelete} disabled={isDeleting}>
                {isDeleting ? 'Deleting...' : 'Delete Environment'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>
      </div>
    </DefaultLayout>
  );
};
