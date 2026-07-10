import React, { useState, useMemo, useCallback } from 'react';
import { subDays } from 'date-fns';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import type {
  CreatePolicyRequest,
  Policy,
  SQLReviewPolicyConfig,
  UpdatePolicyRequest,
  Environment,
} from '@/lib/api/types';
import {
  usePolicies,
  useCreatePolicy,
  useUpdatePolicy,
  useDeletePolicy,
} from '@/hooks/api/usePoliciesSWR';
import { useEnvironmentsSWR } from '@/hooks/api/useEnvironmentsSWR';
import {
  useServerTableFilters,
  type ServerTableFilterMapping,
} from '@/hooks/useServerTableFilters';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
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
import { getPoliciesColumns } from './policies/columns';
import { Plus, AlertTriangle, Code2, FileCheck2, Info } from 'lucide-react';
import { SQLReviewRuleSelector } from '@/components/sql-review-rule-selector';
import { hasOrganizationPermission, OrganizationPermission } from '@/lib/rbac';

const POLICIES_FILTER_MAPPING: ServerTableFilterMapping = {
  searchColumnIds: ['name'],
  equals: { type: 'type' },
  dateRanges: { created_at: ['created_after', 'created_before'] },
};

// Filter definitions are static (type is a static enum; no reference-backed
// filters), so they live at module level.
const POLICIES_FILTER_DEFINITIONS: FilterDefinition[] = [
  {
    id: 'name',
    type: 'search',
    title: 'Name',
    placeholder: 'Search names...',
    defaultActive: true,
  },
  {
    id: 'type',
    type: 'faceted',
    title: 'Type',
    options: [
      { label: 'Syntax', value: 'Syntax', icon: FileCheck2 },
      { label: 'Security', value: 'Security', icon: AlertTriangle },
      { label: 'Performance', value: 'Performance', icon: Info },
      { label: 'Backwards Compatibility', value: 'BackwardsCompatibility', icon: FileCheck2 },
      { label: 'Best Practices', value: 'BestPractices', icon: Info },
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

export const PoliciesPage: React.FC = () => {
  const { user } = useAuth();
  const [filterType, setFilterType] = useState<'SQL_REVIEW' | 'all'>('SQL_REVIEW');
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [selectedPolicy, setSelectedPolicy] = useState<Policy | null>(null);
  const [useJsonEditor, setUseJsonEditor] = useState(false);
  const [jsonEditorError, setJsonEditorError] = useState('');

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
    mapping: POLICIES_FILTER_MAPPING,
    filterDefinitions: POLICIES_FILTER_DEFINITIONS,
    initialPageSize: 50,
  });

  // Form state
  const [formData, setFormData] = useState<CreatePolicyRequest>({
    name: '',
    description: '',
    type: 'SQL_REVIEW',
    policy_config: { rules: {} },
  });
  const [formErrors, setFormErrors] = useState<{
    name?: string;
    description?: string;
    policy_config?: string;
  }>({});

  // Check permissions
  const canReadPolicies = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.PoliciesRead,
  );
  const canCreatePolicies = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.PoliciesCreate,
  );
  const canUpdatePolicies = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.PoliciesUpdate,
  );
  const canDeletePolicies = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.PoliciesDelete,
  );

  // Use SWR hooks for data fetching. The dropdown drives `type` when a specific
  // type is selected; otherwise fall back to the toolbar's faceted `type` filter.
  const policiesParams = useMemo(
    () => ({
      limit: pageSize,
      offset: pageIndex * pageSize,
      search: typeof params.search === 'string' ? params.search : undefined,
      created_after: typeof params.created_after === 'string' ? params.created_after : undefined,
      created_before: typeof params.created_before === 'string' ? params.created_before : undefined,
      type:
        filterType === 'SQL_REVIEW'
          ? 'SQL_REVIEW'
          : typeof params.type === 'string'
            ? params.type
            : undefined,
      enabled: canReadPolicies,
    }),
    [params, pageIndex, pageSize, filterType, canReadPolicies],
  );

  const { policies, total, isLoading, error } = usePolicies(policiesParams);
  const { environments } = useEnvironmentsSWR();

  // Use mutation hooks
  const { createPolicy, isLoading: isCreating } = useCreatePolicy();
  const { updatePolicy, isLoading: isUpdating } = useUpdatePolicy();
  const { deletePolicy, isLoading: isDeleting } = useDeletePolicy();

  // Validate form
  const validateForm = (): boolean => {
    const errors: { name?: string; description?: string; policy_config?: string } = {};

    if (!formData.name.trim()) {
      errors.name = 'Policy name is required';
    } else if (formData.name.trim().length < 3) {
      errors.name = 'Policy name must be at least 3 characters';
    }

    if (!useJsonEditor) {
      // Validate that at least one rule is enabled
      const enabledCount = Object.keys(formData.policy_config.rules || {}).length;
      if (enabledCount === 0) {
        errors.policy_config = 'At least one rule must be enabled';
      }
    } else {
      // Validate JSON
      try {
        const config = JSON.parse(formData.policy_config as unknown as string);
        if (!config || typeof config !== 'object' || !config.rules) {
          errors.policy_config = 'Policy config must have a rules object';
        }
      } catch {
        errors.policy_config = 'Invalid JSON format';
      }
    }

    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  // Handle create
  const handleCreate = async () => {
    if (!validateForm()) {
      return;
    }

    const createData: CreatePolicyRequest = {
      name: formData.name.trim(),
      description: formData.description?.trim() || undefined,
      type: 'SQL_REVIEW',
      policy_config: useJsonEditor
        ? (JSON.parse(formData.policy_config as unknown as string) as SQLReviewPolicyConfig)
        : formData.policy_config,
    };

    const result = await createPolicy(createData);
    if (result) {
      setIsCreateOpen(false);
      setFormData({ name: '', description: '', type: 'SQL_REVIEW', policy_config: { rules: {} } });
      setFormErrors({});
      setUseJsonEditor(false);
    }
  };

  // Handle edit
  const handleEdit = async () => {
    if (!selectedPolicy || !validateForm()) {
      return;
    }

    const updateData: UpdatePolicyRequest = {};
    if (formData.name !== selectedPolicy.name) {
      updateData.name = formData.name.trim();
    }
    if (formData.description !== selectedPolicy.description) {
      updateData.description = formData.description?.trim() || undefined;
    }

    const newConfig = useJsonEditor
      ? (JSON.parse(formData.policy_config as unknown as string) as SQLReviewPolicyConfig)
      : formData.policy_config;

    // Only update if config changed
    if (JSON.stringify(newConfig) !== JSON.stringify(selectedPolicy.policy_config)) {
      updateData.policy_config = newConfig;
    }

    const result = await updatePolicy(selectedPolicy.id, updateData);
    if (result) {
      setIsEditOpen(false);
      setSelectedPolicy(null);
      setFormData({ name: '', description: '', type: 'SQL_REVIEW', policy_config: { rules: {} } });
      setFormErrors({});
      setUseJsonEditor(false);
    }
  };

  // Handle delete
  const handleDelete = async () => {
    if (!selectedPolicy) {
      return;
    }

    const success = await deletePolicy(selectedPolicy.id);
    if (success) {
      setIsDeleteOpen(false);
      setSelectedPolicy(null);
    }
  };

  // Open create form
  const openCreateForm = useCallback(() => {
    setFormData({ name: '', description: '', type: 'SQL_REVIEW', policy_config: { rules: {} } });
    setFormErrors({});
    setUseJsonEditor(false);
    setJsonEditorError('');
    setIsCreateOpen(true);
  }, []);

  // Open edit form
  const openEditForm = useCallback((policy: Policy) => {
    setSelectedPolicy(policy);
    const config = policy.policy_config as SQLReviewPolicyConfig;
    // Deep clone the config to avoid mutating the original policy object
    const clonedConfig = config.rules
      ? (JSON.parse(JSON.stringify(config)) as SQLReviewPolicyConfig)
      : { rules: {} };
    setFormData({
      name: policy.name,
      description: policy.description || '',
      type: policy.type,
      policy_config: clonedConfig,
    });
    setFormErrors({});
    setUseJsonEditor(false);
    setJsonEditorError('');
    setIsEditOpen(true);
  }, []);

  // Open delete confirmation
  const openDeleteConfirmation = useCallback((policy: Policy) => {
    setSelectedPolicy(policy);
    setIsDeleteOpen(true);
  }, []);

  // Check if policy is referenced by environments
  const isPolicyReferenced = useCallback(
    (policyId: string): boolean => {
      return environments.some((env) => env.policy_id === policyId);
    },
    [environments],
  );

  // Get environments using this policy
  const getReferencingEnvironments = useCallback(
    (policyId: string): Environment[] => {
      return environments.filter((env) => env.policy_id === policyId);
    },
    [environments],
  );

  // Get count of environments using this policy
  const getReferencingEnvironmentsCount = useCallback(
    (policyId: string): number => {
      return environments.filter((env) => env.policy_id === policyId).length;
    },
    [environments],
  );

  // Get columns with action handlers
  const columns = getPoliciesColumns({
    onEdit: openEditForm,
    onDelete: openDeleteConfirmation,
    canUpdate: canUpdatePolicies,
    canDelete: canDeletePolicies,
    getReferencingEnvironmentsCount,
  });

  // Render access denied
  if (!canReadPolicies) {
    return (
      <DefaultLayout breadcrumbs={[{ label: 'Policies', isCurrentPage: true }]}>
        <Card>
          <CardHeader>
            <CardTitle>Access Denied</CardTitle>
            <CardDescription>You do not have permission to access this page.</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              This page requires the policies.read permission.
            </p>
          </CardContent>
        </Card>
      </DefaultLayout>
    );
  }

  return (
    <DefaultLayout breadcrumbs={[{ label: 'Policies', isCurrentPage: true }]}>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Policies</h1>
            <p className="text-muted-foreground mt-1">
              Manage SQL review policies and validator configurations
            </p>
          </div>
          {canCreatePolicies && (
            <Button onClick={openCreateForm}>
              <Plus className="mr-2 h-4 w-4" />
              Create Policy
            </Button>
          )}
        </div>

        {/* Filter */}
        <div className="flex items-center gap-4">
          <label htmlFor="filter-type" className="text-sm font-medium">
            Filter by Type:
          </label>
          <select
            id="filter-type"
            value={filterType}
            onChange={(e) => setFilterType(e.target.value as 'SQL_REVIEW' | 'all')}
            className="flex h-9 w-48 rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <option value="SQL_REVIEW">SQL_REVIEW</option>
            <option value="all">All Types</option>
          </select>
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

        {/* Policies list */}
        {!isLoading && policies.length === 0 ? (
          <Card>
            <CardContent className="pt-6">
              <div className="text-center py-8">
                <p className="text-muted-foreground">No policies found.</p>
                {canCreatePolicies && (
                  <Button onClick={openCreateForm} className="mt-4">
                    <Plus className="mr-2 h-4 w-4" />
                    Create Your First Policy
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardContent className="pt-6">
              <DataTable
                columns={columns}
                data={policies}
                isLoading={isLoading}
                initialColumnFilters={initialColumnFilters}
                emptyMessage="No policies found."
                pagination={{
                  pageIndex,
                  pageSize,
                  totalCount: total,
                  onPageChange: setPageIndex,
                  onPageSizeChange: setPageSize,
                  pageSizeOptions: [10, 20, 50, 100],
                  manualFiltering: true,
                }}
                onFiltersChange={onFiltersChange}
              >
                {(table) => (
                  <DataTableToolbar table={table} filters={POLICIES_FILTER_DEFINITIONS} />
                )}
              </DataTable>
            </CardContent>
          </Card>
        )}

        {/* Create Policy Sheet */}
        <Sheet open={isCreateOpen} onOpenChange={setIsCreateOpen}>
          <SheetContent className="overflow-y-auto max-h-[90vh] min-w-[600px] sm:w-[540px]">
            <SheetHeader>
              <SheetTitle>Create Policy</SheetTitle>
              <SheetDescription>
                Create a new SQL review policy. Configure which validators to enable.
              </SheetDescription>
            </SheetHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <label htmlFor="create-name" className="text-sm font-medium">
                  Policy Name *
                </label>
                <Input
                  id="create-name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Enter policy name"
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
                  placeholder="Enter policy description"
                  className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                  aria-invalid={!!formErrors.description}
                />
                {formErrors.description && (
                  <p className="text-sm text-destructive">{formErrors.description}</p>
                )}
              </div>
              <div className="space-y-2">
                <label htmlFor="create-type" className="text-sm font-medium">
                  Policy Type *
                </label>
                <select
                  id="create-type"
                  value={formData.type}
                  disabled
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="SQL_REVIEW">SQL_REVIEW</option>
                </select>
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label htmlFor="policy-config" className="text-sm font-medium">
                    Policy Configuration *
                  </label>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      if (useJsonEditor) {
                        // Switch to rule selector
                        try {
                          const config = JSON.parse(
                            formData.policy_config as unknown as string,
                          ) as SQLReviewPolicyConfig;
                          setFormData({
                            ...formData,
                            policy_config: config,
                          });
                        } catch {
                          // Invalid JSON, keep current config
                        }
                      } else {
                        // Switch to JSON editor
                        setFormData({
                          ...formData,
                          policy_config: JSON.stringify(
                            formData.policy_config,
                            null,
                            2,
                          ) as unknown as SQLReviewPolicyConfig,
                        });
                      }
                      setUseJsonEditor(!useJsonEditor);
                      setJsonEditorError('');
                    }}
                    className="h-8"
                  >
                    <Code2 className="mr-2 h-4 w-4" />
                    {useJsonEditor ? 'Use Rule Selector' : 'Use JSON Editor'}
                  </Button>
                </div>
                {useJsonEditor ? (
                  <div className="space-y-2">
                    <textarea
                      id="policy-config"
                      value={(formData.policy_config as unknown as string) || '{}'}
                      onChange={(e) => {
                        setFormData({
                          ...formData,
                          policy_config: e.target.value as unknown as SQLReviewPolicyConfig,
                        });
                        // Validate JSON on change
                        try {
                          JSON.parse(e.target.value);
                          setJsonEditorError('');
                        } catch {
                          setJsonEditorError('Invalid JSON format');
                        }
                      }}
                      placeholder='{"rules": {"mysql.engine.no-select-all": {"type": "mysql.engine.no-select-all", "level": "error", "engine": "mysql"}}}'
                      className="flex min-h-[200px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm font-mono shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                      aria-invalid={!!formErrors.policy_config || !!jsonEditorError}
                    />
                    {(formErrors.policy_config || jsonEditorError) && (
                      <p className="text-sm text-destructive">
                        {formErrors.policy_config || jsonEditorError}
                      </p>
                    )}
                    <p className="text-xs text-muted-foreground">
                      Format:{' '}
                      {`{"rules": {"rule-key": {"type": "...", "level": "error|warning", "engine": "mysql|postgresql", "payload": {...}}}}`}
                    </p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    <SQLReviewRuleSelector
                      config={formData.policy_config}
                      onChange={(newConfig) =>
                        setFormData({ ...formData, policy_config: newConfig })
                      }
                    />
                    {formErrors.policy_config && (
                      <p className="text-sm text-destructive">{formErrors.policy_config}</p>
                    )}
                  </div>
                )}
              </div>
            </div>
            <SheetFooter>
              <Button variant="outline" onClick={() => setIsCreateOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleCreate} disabled={isCreating}>
                {isCreating ? 'Creating...' : 'Create Policy'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>

        {/* Edit Policy Sheet */}
        <Sheet open={isEditOpen} onOpenChange={setIsEditOpen}>
          <SheetContent className="overflow-y-auto max-h-[90vh] min-w-[600px] sm:w-[540px]">
            <SheetHeader>
              <SheetTitle>Edit Policy</SheetTitle>
              <SheetDescription>
                Update policy information and validator configuration.
              </SheetDescription>
            </SheetHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <label htmlFor="edit-name" className="text-sm font-medium">
                  Policy Name *
                </label>
                <Input
                  id="edit-name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Enter policy name"
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
                  placeholder="Enter policy description"
                  className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                  aria-invalid={!!formErrors.description}
                />
                {formErrors.description && (
                  <p className="text-sm text-destructive">{formErrors.description}</p>
                )}
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label htmlFor="edit-policy-config" className="text-sm font-medium">
                    Policy Configuration *
                  </label>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      if (useJsonEditor) {
                        // Switch to rule selector
                        try {
                          const config = JSON.parse(
                            formData.policy_config as unknown as string,
                          ) as SQLReviewPolicyConfig;
                          setFormData({
                            ...formData,
                            policy_config: config,
                          });
                        } catch {
                          // Invalid JSON, keep current config
                        }
                      } else {
                        // Switch to JSON editor
                        setFormData({
                          ...formData,
                          policy_config: JSON.stringify(
                            formData.policy_config,
                            null,
                            2,
                          ) as unknown as SQLReviewPolicyConfig,
                        });
                      }
                      setUseJsonEditor(!useJsonEditor);
                      setJsonEditorError('');
                    }}
                    className="h-8"
                  >
                    <Code2 className="mr-2 h-4 w-4" />
                    {useJsonEditor ? 'Use Rule Selector' : 'Use JSON Editor'}
                  </Button>
                </div>
                {useJsonEditor ? (
                  <div className="space-y-2">
                    <textarea
                      id="edit-policy-config"
                      value={(formData.policy_config as unknown as string) || '{}'}
                      onChange={(e) => {
                        setFormData({
                          ...formData,
                          policy_config: e.target.value as unknown as SQLReviewPolicyConfig,
                        });
                        // Validate JSON on change
                        try {
                          JSON.parse(e.target.value);
                          setJsonEditorError('');
                        } catch {
                          setJsonEditorError('Invalid JSON format');
                        }
                      }}
                      placeholder='{"rules": {"mysql.engine.no-select-all": {"type": "mysql.engine.no-select-all", "level": "error", "engine": "mysql"}}}'
                      className="flex min-h-[200px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm font-mono shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                      aria-invalid={!!formErrors.policy_config || !!jsonEditorError}
                    />
                    {(formErrors.policy_config || jsonEditorError) && (
                      <p className="text-sm text-destructive">
                        {formErrors.policy_config || jsonEditorError}
                      </p>
                    )}
                    <p className="text-xs text-muted-foreground">
                      Format:{' '}
                      {`{"rules": {"rule-key": {"type": "...", "level": "error|warning", "engine": "mysql|postgresql", "payload": {...}}}}`}
                    </p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    <SQLReviewRuleSelector
                      config={formData.policy_config}
                      onChange={(newConfig) =>
                        setFormData({ ...formData, policy_config: newConfig })
                      }
                    />
                    {formErrors.policy_config && (
                      <p className="text-sm text-destructive">{formErrors.policy_config}</p>
                    )}
                  </div>
                )}
              </div>
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
              <SheetTitle>Delete Policy</SheetTitle>
              <SheetDescription>
                {selectedPolicy && isPolicyReferenced(selectedPolicy.id)
                  ? 'This policy is referenced by one or more environments. Deleting it will fail. Please remove or reassign the policy from environments first.'
                  : 'Are you sure you want to delete this policy? This action cannot be undone.'}
              </SheetDescription>
            </SheetHeader>
            <div className="py-4">
              {selectedPolicy && (
                <div className="space-y-2">
                  <p className="text-sm font-medium">{selectedPolicy.name}</p>
                  <p className="text-sm text-muted-foreground">
                    {selectedPolicy.description || 'No description'}
                  </p>
                  {isPolicyReferenced(selectedPolicy.id) && (
                    <div className="mt-4 space-y-2">
                      <p className="text-sm font-medium text-yellow-600 dark:text-yellow-400">
                        Referenced by:
                      </p>
                      <ul className="list-disc list-inside text-sm text-muted-foreground">
                        {getReferencingEnvironments(selectedPolicy.id).map((env) => (
                          <li key={env.id}>{env.name}</li>
                        ))}
                      </ul>
                    </div>
                  )}
                </div>
              )}
            </div>
            <SheetFooter>
              <Button variant="outline" onClick={() => setIsDeleteOpen(false)}>
                Cancel
              </Button>
              <Button
                variant="destructive"
                onClick={handleDelete}
                disabled={
                  isDeleting || (selectedPolicy ? isPolicyReferenced(selectedPolicy.id) : false)
                }
              >
                {isDeleting ? 'Deleting...' : 'Delete Policy'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>
      </div>
    </DefaultLayout>
  );
};
