import React, { useState } from 'react';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
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
import { Badge } from '@/components/ui/badge';
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Plus, Pencil, Trash2, AlertTriangle, Globe } from 'lucide-react';

export const EnvironmentsPage: React.FC = () => {
  const { user } = useAuth();
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [selectedEnvironment, setSelectedEnvironment] = useState<Environment | null>(null);

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

  // Check if user is admin
  const isAdmin = user?.organization_role === 'OrganizationAdmin';

  // Use SWR hooks for data fetching
  const { environments, isLoading, error } = useEnvironmentsSWR();
  const { policies, isLoading: isLoadingPolicies } = usePolicies({
    type: 'SQL_REVIEW',
    limit: 100,
    enabled: isAdmin,
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
  const openCreateForm = () => {
    setFormData({ name: '', description: '', policy_id: undefined, allow_self_approval: false });
    setFormErrors({});
    setIsCreateOpen(true);
  };

  // Open edit form
  const openEditForm = (environment: Environment) => {
    setSelectedEnvironment(environment);
    setFormData({
      name: environment.name,
      description: environment.description || '',
      policy_id: environment.policy_id || undefined,
      allow_self_approval: environment.allow_self_approval || false,
    });
    setFormErrors({});
    setIsEditOpen(true);
  };

  // Open delete confirmation
  const openDeleteConfirmation = (environment: Environment) => {
    setSelectedEnvironment(environment);
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

  // Get policy name by ID
  const getPolicyName = (policyId?: string): string => {
    if (!policyId) {
      return 'None';
    }
    const policy = policies.find((p) => p.id === policyId);
    return policy?.name || 'Unknown';
  };

  // Render access denied
  if (!isAdmin) {
    return (
      <DefaultLayout breadcrumbs={[{ label: 'Environments', isCurrentPage: true }]}>
        <Card>
          <CardHeader>
            <CardTitle>Access Denied</CardTitle>
            <CardDescription>You do not have permission to access this page.</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              This page requires Organization Admin role.
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
          <Button onClick={openCreateForm}>
            <Plus className="mr-2 h-4 w-4" />
            Create Environment
          </Button>
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
        {isLoading ? (
          <Card>
            <CardContent className="pt-6">
              <div className="space-y-4">
                <Skeleton className="h-9 w-full" />
                <div className="space-y-2">
                  <Skeleton className="h-6 w-32" />
                  <Skeleton className="h-4 w-48" />
                  <Skeleton className="h-4 w-32" />
                </div>
              </div>
            </CardContent>
          </Card>
        ) : environments.length === 0 ? (
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
                  <Button onClick={openCreateForm}>
                    <Plus className="mr-2 h-4 w-4" />
                    Create Environment
                  </Button>
                </EmptyContent>
              </Empty>
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardContent className="pt-6">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Description</TableHead>
                    <TableHead>Policy</TableHead>
                    <TableHead>Self Approval</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {environments.map((environment) => (
                    <TableRow key={environment.id}>
                      <TableCell className="font-medium">{environment.name}</TableCell>
                      <TableCell className="text-muted-foreground">
                        {environment.description || '-'}
                      </TableCell>
                      <TableCell>{getPolicyName(environment.policy_id)}</TableCell>
                      <TableCell>
                        {environment.allow_self_approval ? (
                          <Badge variant="secondary" className="bg-green-100 text-green-800">
                            Enabled
                          </Badge>
                        ) : (
                          <Badge variant="secondary" className="bg-gray-100 text-gray-800">
                            Disabled
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell>{formatDate(environment.created_at)}</TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-2">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => openEditForm(environment)}
                            className="h-8 w-8 p-0"
                          >
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => openDeleteConfirmation(environment)}
                            className="h-8 w-8 p-0 text-destructive hover:text-destructive"
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
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
