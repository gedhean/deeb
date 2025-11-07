import React, { useEffect, useState } from 'react';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { policiesService } from '@/lib/api/services/policies';
import { environmentsService } from '@/lib/api/services/environments';
import type {
  CreatePolicyRequest,
  Policy,
  SQLReviewPolicyConfig,
  UpdatePolicyRequest,
  Environment,
  MigrationStatus,
} from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { Plus, Pencil, Trash2, AlertTriangle, Code2, ArrowUpCircle, RefreshCw } from 'lucide-react';
import { SQLReviewRuleSelector } from '@/components/sql-review-rule-selector';
import { toast } from 'sonner';

export const PoliciesPage: React.FC = () => {
  const { user } = useAuth();
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [environments, setEnvironments] = useState<Environment[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [filterType, setFilterType] = useState<'SQL_REVIEW' | 'all'>('SQL_REVIEW');
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [selectedPolicy, setSelectedPolicy] = useState<Policy | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [useJsonEditor, setUseJsonEditor] = useState(false);
  const [jsonEditorError, setJsonEditorError] = useState('');
  const [migrationStatus, setMigrationStatus] = useState<MigrationStatus | null>(null);
  const [isMigrating, setIsMigrating] = useState(false);

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

  // Check if user is admin
  const isAdmin = user?.organization_role === 'OrganizationAdmin';

  // Load environments to check references
  useEffect(() => {
    if (!isAdmin) {
      return;
    }

    const loadEnvironments = async () => {
      try {
        const response = await environmentsService.listEnvironments({ limit: 100 });
        setEnvironments(response.items);
      } catch (err) {
        console.error('Failed to load environments:', err);
      }
    };

    loadEnvironments();
  }, [isAdmin]);

  // Load policies
  useEffect(() => {
    if (!isAdmin) {
      setError('Access denied. Admin access required.');
      setIsLoading(false);
      return;
    }

    const loadPolicies = async () => {
      try {
        setIsLoading(true);
        setError('');
        const response = await policiesService.listPolicies({
          type: filterType === 'SQL_REVIEW' ? 'SQL_REVIEW' : undefined,
          limit: 100,
        });
        setPolicies(response.items);
      } catch (err) {
        if (err instanceof APIError) {
          setError(err.message);
        } else {
          setError('Failed to load policies. Please try again.');
        }
        console.error('Failed to load policies:', err);
      } finally {
        setIsLoading(false);
      }
    };

    loadPolicies();
  }, [isAdmin, filterType]);

  // Load migration status
  useEffect(() => {
    if (!isAdmin) {
      return;
    }

    const loadMigrationStatus = async () => {
      try {
        const status = await policiesService.getMigrationStatus();
        setMigrationStatus(status);
      } catch (err) {
        console.error('Failed to load migration status:', err);
      }
    };

    loadMigrationStatus();
  }, [isAdmin, policies]); // Reload when policies change

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

    try {
      setIsSubmitting(true);
      const createData: CreatePolicyRequest = {
        name: formData.name.trim(),
        description: formData.description?.trim() || undefined,
        type: 'SQL_REVIEW',
        policy_config: useJsonEditor
          ? (JSON.parse(formData.policy_config as unknown as string) as SQLReviewPolicyConfig)
          : formData.policy_config,
      };
      await policiesService.createPolicy(createData);
      setIsCreateOpen(false);
      setFormData({ name: '', description: '', type: 'SQL_REVIEW', policy_config: { rules: {} } });
      setFormErrors({});
      setUseJsonEditor(false);
      // Reload policies
      const response = await policiesService.listPolicies({
        type: filterType === 'SQL_REVIEW' ? 'SQL_REVIEW' : undefined,
        limit: 100,
      });
      setPolicies(response.items);
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to create policy. Please try again.');
      }
      console.error('Failed to create policy:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Handle edit
  const handleEdit = async () => {
    if (!selectedPolicy || !validateForm()) {
      return;
    }

    try {
      setIsSubmitting(true);
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

      await policiesService.updatePolicy(selectedPolicy.id, updateData);
      setIsEditOpen(false);
      setSelectedPolicy(null);
      setFormData({ name: '', description: '', type: 'SQL_REVIEW', policy_config: { rules: {} } });
      setFormErrors({});
      setUseJsonEditor(false);
      // Reload policies
      const response = await policiesService.listPolicies({
        type: filterType === 'SQL_REVIEW' ? 'SQL_REVIEW' : undefined,
        limit: 100,
      });
      setPolicies(response.items);
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to update policy. Please try again.');
      }
      console.error('Failed to update policy:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Handle delete
  const handleDelete = async () => {
    if (!selectedPolicy) {
      return;
    }

    try {
      setIsSubmitting(true);
      await policiesService.deletePolicy(selectedPolicy.id);
      setIsDeleteOpen(false);
      setSelectedPolicy(null);
      // Reload policies
      const response = await policiesService.listPolicies({
        type: filterType === 'SQL_REVIEW' ? 'SQL_REVIEW' : undefined,
        limit: 100,
      });
      setPolicies(response.items);
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to delete policy. Please try again.');
      }
      console.error('Failed to delete policy:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Handle migrate single policy
  const handleMigratePolicy = async (policy: Policy) => {
    try {
      setIsMigrating(true);
      const result = await policiesService.migratePolicy(policy.id);

      if (result.success) {
        toast.success(`Policy "${policy.name}" migrated successfully`, {
          description: `Added ${result.rules_added} new rules`,
        });

        // Reload policies
        const response = await policiesService.listPolicies({
          type: filterType === 'SQL_REVIEW' ? 'SQL_REVIEW' : undefined,
          limit: 100,
        });
        setPolicies(response.items);
      } else {
        toast.error(`Failed to migrate policy "${policy.name}"`, {
          description: result.error || 'Unknown error',
        });
      }
    } catch (err) {
      if (err instanceof APIError) {
        toast.error('Migration failed', {
          description: err.message,
        });
      } else {
        toast.error('Migration failed', {
          description: 'An unexpected error occurred',
        });
      }
      console.error('Failed to migrate policy:', err);
    } finally {
      setIsMigrating(false);
    }
  };

  // Handle migrate all policies
  const handleMigrateAll = async () => {
    try {
      setIsMigrating(true);
      const result = await policiesService.migrateAllPolicies();

      toast.success('Batch migration completed', {
        description: `${result.success_count} succeeded, ${result.failure_count} failed`,
      });

      // Reload policies
      const response = await policiesService.listPolicies({
        type: filterType === 'SQL_REVIEW' ? 'SQL_REVIEW' : undefined,
        limit: 100,
      });
      setPolicies(response.items);
    } catch (err) {
      if (err instanceof APIError) {
        toast.error('Batch migration failed', {
          description: err.message,
        });
      } else {
        toast.error('Batch migration failed', {
          description: 'An unexpected error occurred',
        });
      }
      console.error('Failed to migrate all policies:', err);
    } finally {
      setIsMigrating(false);
    }
  };

  // Open create form
  const openCreateForm = () => {
    setFormData({ name: '', description: '', type: 'SQL_REVIEW', policy_config: { rules: {} } });
    setFormErrors({});
    setUseJsonEditor(false);
    setJsonEditorError('');
    setIsCreateOpen(true);
  };

  // Open edit form
  const openEditForm = (policy: Policy) => {
    setSelectedPolicy(policy);
    const config = policy.policy_config as SQLReviewPolicyConfig;
    setFormData({
      name: policy.name,
      description: policy.description || '',
      type: policy.type,
      policy_config: config.rules ? config : { rules: {} },
    });
    setFormErrors({});
    setUseJsonEditor(false);
    setJsonEditorError('');
    setIsEditOpen(true);
  };

  // Open delete confirmation
  const openDeleteConfirmation = (policy: Policy) => {
    setSelectedPolicy(policy);
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

  // Check if policy is referenced by environments
  const isPolicyReferenced = (policyId: string): boolean => {
    return environments.some((env) => env.policy_id === policyId);
  };

  // Get environments using this policy
  const getReferencingEnvironments = (policyId: string): Environment[] => {
    return environments.filter((env) => env.policy_id === policyId);
  };

  // Render access denied
  if (!isAdmin) {
    return (
      <DefaultLayout breadcrumbs={[{ label: 'Policies', isCurrentPage: true }]}>
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
          <Button onClick={openCreateForm}>
            <Plus className="mr-2 h-4 w-4" />
            Create Policy
          </Button>
        </div>

        {/* Migration Status Card */}
        {migrationStatus && migrationStatus.needs_migration > 0 && (
          <Card className="border-blue-200 bg-blue-50/50 dark:border-blue-900 dark:bg-blue-950/20">
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-lg flex items-center gap-2">
                    <ArrowUpCircle className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                    Policy Schema Migration Available
                  </CardTitle>
                  <CardDescription className="mt-1">
                    Upgrade your policies to V2 for access to 101 comprehensive SQL review rules
                  </CardDescription>
                </div>
                <Button
                  onClick={handleMigrateAll}
                  disabled={isMigrating}
                  className="bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600"
                >
                  {isMigrating ? (
                    <>
                      <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                      Migrating...
                    </>
                  ) : (
                    <>
                      <ArrowUpCircle className="mr-2 h-4 w-4" />
                      Migrate All to V2
                    </>
                  )}
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-4 gap-4 text-sm">
                <div className="space-y-1">
                  <p className="text-muted-foreground">Total Policies</p>
                  <p className="text-2xl font-bold">{migrationStatus.total}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-muted-foreground">V1 (Legacy)</p>
                  <p className="text-2xl font-bold text-yellow-600 dark:text-yellow-400">
                    {migrationStatus['1.0']}
                  </p>
                </div>
                <div className="space-y-1">
                  <p className="text-muted-foreground">V2 (Enhanced)</p>
                  <p className="text-2xl font-bold text-green-600 dark:text-green-400">
                    {migrationStatus['2.0']}
                  </p>
                </div>
                <div className="space-y-1">
                  <p className="text-muted-foreground">Needs Migration</p>
                  <p className="text-2xl font-bold text-blue-600 dark:text-blue-400">
                    {migrationStatus.needs_migration}
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        )}

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
        {isLoading ? (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {[1, 2, 3].map((i) => (
              <Card key={i}>
                <CardHeader>
                  <Skeleton className="h-6 w-32" />
                  <Skeleton className="h-4 w-48 mt-2" />
                </CardHeader>
                <CardContent>
                  <Skeleton className="h-4 w-full" />
                  <Skeleton className="h-4 w-24 mt-2" />
                </CardContent>
              </Card>
            ))}
          </div>
        ) : policies.length === 0 ? (
          <Card>
            <CardContent className="pt-6">
              <div className="text-center py-8">
                <p className="text-muted-foreground">No policies found.</p>
                <Button onClick={openCreateForm} className="mt-4">
                  <Plus className="mr-2 h-4 w-4" />
                  Create Your First Policy
                </Button>
              </div>
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {policies.map((policy) => (
              <Card key={policy.id}>
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <CardTitle className="text-lg">{policy.name}</CardTitle>
                      <CardDescription className="line-clamp-2 mt-1">
                        {policy.description || 'No description'}
                      </CardDescription>
                    </div>
                    <span
                      className={`text-xs px-2 py-1 rounded-full font-medium ${
                        policy.schema_version === '2.0'
                          ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                          : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
                      }`}
                    >
                      V{policy.schema_version}
                    </span>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <div className="text-xs text-muted-foreground">
                      <span className="font-medium">Type:</span> {policy.type}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      Created {formatDate(policy.created_at)}
                    </div>
                    {isPolicyReferenced(policy.id) && (
                      <div className="text-xs text-yellow-600 dark:text-yellow-400">
                        ⚠️ Referenced by {getReferencingEnvironments(policy.id).length}{' '}
                        environment(s)
                      </div>
                    )}
                    {policy.schema_version === '1.0' && (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleMigratePolicy(policy)}
                        disabled={isMigrating}
                        className="w-full mt-2 border-blue-300 text-blue-600 hover:bg-blue-50 dark:border-blue-700 dark:text-blue-400 dark:hover:bg-blue-950/30"
                      >
                        <ArrowUpCircle className="mr-2 h-3 w-3" />
                        Upgrade to V2
                      </Button>
                    )}
                    <div className="flex items-center justify-end gap-2 pt-2">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => openEditForm(policy)}
                        className="h-8 w-8 p-0"
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => openDeleteConfirmation(policy)}
                        className="h-8 w-8 p-0 text-destructive hover:text-destructive"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
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
                          policy_config: e.target.value as unknown as Record<string, unknown>,
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
              <Button onClick={handleCreate} disabled={isSubmitting}>
                {isSubmitting ? 'Creating...' : 'Create Policy'}
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
                          policy_config: e.target.value as unknown as Record<string, unknown>,
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
                  isSubmitting || (selectedPolicy ? isPolicyReferenced(selectedPolicy.id) : false)
                }
              >
                {isSubmitting ? 'Deleting...' : 'Delete Policy'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>
      </div>
    </DefaultLayout>
  );
};
