import React, { useEffect, useState, useMemo, useCallback } from 'react';
import { subDays } from 'date-fns';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { projectMembersService } from '@/lib/api/services/projects';
import { hasOrganizationPermission, OrganizationPermission, isAdmin } from '@/lib/rbac';
import type {
  AddProjectMemberRequest,
  CreateProjectRequest,
  Project,
  ProjectMemberWithUser,
  ProjectRole,
  UpdateProjectRequest,
} from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { showSuccessToast, handleApiError } from '@/lib/errorHandling';
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { DataTable, DataTableToolbar, DataTableViewOptions } from '@/components/data-table';
import type { FilterDefinition } from '@/components/data-table/types';
import { getProjectsColumns } from './projects/columns';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { RoleBadge } from '@/components/RoleBadge';
import { Plus, AlertTriangle, FolderOpen, Users, Trash2 } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useProjectPermissions } from '@/hooks/api/useProjectPermissions';
import { useUsers } from '@/hooks/api/useUsersSWR';
import {
  useProjectsSWR,
  useCreateProjectSWR,
  useUpdateProjectSWR,
  useDeleteProjectSWR,
} from '@/hooks/api/useProjectsSWR';

export const ProjectsPage: React.FC = () => {
  const { user } = useAuth();
  const navigate = useNavigate();

  // Pagination state
  const [pageIndex, setPageIndex] = useState(0);
  const [pageSize, setPageSize] = useState(10);

  // Create pagination params with useMemo
  const paginationParams = useMemo(
    () => ({
      limit: pageSize,
      offset: pageIndex * pageSize,
    }),
    [pageIndex, pageSize],
  );

  // Use cached projects from SWR hook
  const {
    projects: cachedProjects,
    total,
    isLoading: projectsLoading,
    error: projectsError,
  } = useProjectsSWR(paginationParams);

  // Use mutation hooks for create/update/delete
  const { createProject, isLoading: isCreating, error: createError } = useCreateProjectSWR();
  const { updateProject, isLoading: isUpdating, error: updateError } = useUpdateProjectSWR();
  const { deleteProject, isLoading: isDeleting, error: deleteError } = useDeleteProjectSWR();

  const [error, setError] = useState('');
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [isMembersOpen, setIsMembersOpen] = useState(false);
  const [selectedProject, setSelectedProject] = useState<Project | null>(null);

  // Members management state
  const [members, setMembers] = useState<ProjectMemberWithUser[]>([]);
  const [isLoadingMembers, setIsLoadingMembers] = useState(false);
  const [isSubmittingMember, setIsSubmittingMember] = useState(false);
  const [isAddMemberOpen, setIsAddMemberOpen] = useState(false);
  const [isRemoveMemberOpen, setIsRemoveMemberOpen] = useState(false);
  const [selectedMember, setSelectedMember] = useState<ProjectMemberWithUser | null>(null);
  const [addMemberForm, setAddMemberForm] = useState<AddProjectMemberRequest>({
    user_id: '',
    project_role: 'ProjectDeveloper',
  });

  // Use cached users from store
  const { users: availableUsers, isLoading: isLoadingUsers } = useUsers();

  // Form state
  const [formData, setFormData] = useState<CreateProjectRequest>({
    name: '',
    description: '',
  });
  const [formErrors, setFormErrors] = useState<{ name?: string; description?: string }>({});

  // Check permissions for members management
  const { canManageMembers } = useProjectPermissions(members);

  // Check organization-level permissions
  const userIsAdmin = isAdmin(user?.organization_role);
  const canCreateProjects = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.ProjectsCreate,
  );
  const canUpdateProjects = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.ProjectsUpdate,
  );
  const canDeleteProjects = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.ProjectsDelete,
  );

  // Projects are already loaded via useProjectsSWR hook
  // Admins see all projects, non-admins see projects from the hook (which may be filtered by backend)
  const projects = cachedProjects;
  const isLoading = projectsLoading;

  // Set error from SWR hook or mutation hooks if any
  useEffect(() => {
    if (projectsError) {
      setError(projectsError);
    } else if (createError) {
      setError(createError);
    } else if (updateError) {
      setError(updateError);
    } else if (deleteError) {
      setError(deleteError);
    }
  }, [projectsError, createError, updateError, deleteError]);

  // Validate form
  const validateForm = (): boolean => {
    const errors: { name?: string; description?: string } = {};

    if (!formData.name.trim()) {
      errors.name = 'Project name is required';
    } else if (formData.name.trim().length < 3) {
      errors.name = 'Project name must be at least 3 characters';
    }

    if (!formData.description.trim()) {
      errors.description = 'Description is required';
    }

    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  // Handle create
  const handleCreate = async () => {
    if (!validateForm()) {
      return;
    }

    const result = await createProject(formData);
    if (result) {
      setIsCreateOpen(false);
      setFormData({ name: '', description: '' });
      setFormErrors({});
      showSuccessToast('Project created successfully');
    } else {
      // Error is already set by the mutation hook and displayed via useEffect
      handleApiError(new Error('Failed to create project'), {
        showToast: true,
        toastTitle: 'Failed to create project',
      });
    }
  };

  // Handle edit
  const handleEdit = async () => {
    if (!selectedProject || !validateForm()) {
      return;
    }

    const updateData: UpdateProjectRequest = {};
    if (formData.name !== selectedProject.name) {
      updateData.name = formData.name;
    }
    if (formData.description !== selectedProject.description) {
      updateData.description = formData.description;
    }

    const result = await updateProject(selectedProject.id, updateData);
    if (result) {
      setIsEditOpen(false);
      setSelectedProject(null);
      setFormData({ name: '', description: '' });
      setFormErrors({});
      showSuccessToast('Project updated successfully');
    } else {
      // Error is already set by the mutation hook and displayed via useEffect
      handleApiError(new Error('Failed to update project'), {
        showToast: true,
        toastTitle: 'Failed to update project',
      });
    }
  };

  // Handle delete
  const handleDelete = async () => {
    if (!selectedProject) {
      return;
    }

    const success = await deleteProject(selectedProject.id);
    if (success) {
      setIsDeleteOpen(false);
      setSelectedProject(null);
      showSuccessToast('Project deleted successfully');
    } else {
      // Error is already set by the mutation hook and displayed via useEffect
      handleApiError(new Error('Failed to delete project'), {
        showToast: true,
        toastTitle: 'Failed to delete project',
      });
    }
  };

  // Open create form
  const openCreateForm = () => {
    setFormData({ name: '', description: '' });
    setFormErrors({});
    setIsCreateOpen(true);
  };

  // Open edit form
  const openEditForm = useCallback((project: Project) => {
    setSelectedProject(project);
    setFormData({
      name: project.name,
      description: project.description,
    });
    setFormErrors({});
    setIsEditOpen(true);
  }, []);

  // Open delete confirmation
  const openDeleteConfirmation = useCallback((project: Project) => {
    setSelectedProject(project);
    setIsDeleteOpen(true);
  }, []);

  // Load members
  const loadMembers = async (projectId: string) => {
    try {
      setIsLoadingMembers(true);
      const membersList = await projectMembersService.listProjectMembers(projectId);
      setMembers(membersList);
    } catch (err) {
      handleApiError(err, {
        showToast: true,
        toastTitle: 'Failed to load members',
      });
      console.error('Failed to load members:', err);
    } finally {
      setIsLoadingMembers(false);
    }
  };

  // Users are now loaded from cache via useUsers() hook above

  // Handle add member
  const handleAddMember = async () => {
    if (!selectedProject || !addMemberForm.user_id) {
      return;
    }

    try {
      setIsSubmittingMember(true);
      await projectMembersService.addProjectMember(selectedProject.id, addMemberForm);
      setIsAddMemberOpen(false);
      setAddMemberForm({ user_id: '', project_role: 'ProjectDeveloper' });
      showSuccessToast('Member added successfully');
      await loadMembers(selectedProject.id);
    } catch (err) {
      handleApiError(err, {
        showToast: true,
        toastTitle: 'Failed to add member',
      });
      console.error('Failed to add member:', err);
    } finally {
      setIsSubmittingMember(false);
    }
  };

  // Handle remove member
  const handleRemoveMember = async () => {
    if (!selectedProject || !selectedMember) {
      return;
    }

    try {
      setIsSubmittingMember(true);
      await projectMembersService.removeProjectMember(
        selectedProject.id,
        selectedMember.member.user_id,
      );
      setIsRemoveMemberOpen(false);
      setSelectedMember(null);
      showSuccessToast('Member removed successfully');
      await loadMembers(selectedProject.id);
    } catch (err) {
      handleApiError(err, {
        showToast: true,
        toastTitle: 'Failed to remove member',
      });
      console.error('Failed to remove member:', err);
    } finally {
      setIsSubmittingMember(false);
    }
  };

  // Handle update role
  const handleUpdateRole = async (member: ProjectMemberWithUser, newRole: ProjectRole) => {
    if (!selectedProject) {
      return;
    }

    try {
      setIsSubmittingMember(true);
      await projectMembersService.updateProjectMemberRole(
        selectedProject.id,
        member.member.user_id,
        newRole,
      );
      showSuccessToast('Member role updated successfully');
      await loadMembers(selectedProject.id);
    } catch (err) {
      handleApiError(err, {
        showToast: true,
        toastTitle: 'Failed to update role',
      });
      console.error('Failed to update role:', err);
    } finally {
      setIsSubmittingMember(false);
    }
  };

  // Get columns for projects table
  const projectsColumns = useMemo(
    () =>
      getProjectsColumns({
        onView: (project) => navigate(`/projects/${project.id}`),
        onEdit: openEditForm,
        onDelete: openDeleteConfirmation,
        canUpdate: canUpdateProjects,
        canDelete: canDeleteProjects,
      }),
    [navigate, openEditForm, openDeleteConfirmation, canUpdateProjects, canDeleteProjects],
  );

  // Define filter definitions for the toolbar
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
    [],
  );

  // Render access denied
  if (!user) {
    return (
      <DefaultLayout breadcrumbs={[{ label: 'Projects', isCurrentPage: true }]}>
        <Card>
          <CardHeader>
            <CardTitle>Access Denied</CardTitle>
            <CardDescription>You must be logged in to access this page.</CardDescription>
          </CardHeader>
        </Card>
      </DefaultLayout>
    );
  }

  return (
    <DefaultLayout breadcrumbs={[{ label: 'Projects', isCurrentPage: true }]}>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Projects</h1>
            <p className="text-muted-foreground mt-1">
              {userIsAdmin
                ? 'Manage projects and organize database change requests'
                : 'Manage members for projects you own'}
            </p>
          </div>
          {canCreateProjects && (
            <Button onClick={openCreateForm}>
              <Plus className="mr-2 h-4 w-4" />
              Create Project
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

        {/* Projects Table */}
        <Card>
          <CardHeader>
            <CardTitle>Projects</CardTitle>
            <CardDescription>
              {isLoading ? (
                'Loading projects...'
              ) : (
                <>
                  {projects.length} project{projects.length === 1 ? '' : 's'}
                </>
              )}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {!isLoading && projects.length === 0 ? (
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <FolderOpen className="h-12 w-12 text-muted-foreground" />
                  </EmptyMedia>
                  <EmptyTitle>No Projects Yet</EmptyTitle>
                  <EmptyDescription>
                    Get started by creating your first project to organize database change requests.
                  </EmptyDescription>
                </EmptyHeader>
                <EmptyContent>
                  {canCreateProjects ? (
                    <Button onClick={openCreateForm}>
                      <Plus className="mr-2 h-4 w-4" />
                      Create Project
                    </Button>
                  ) : (
                    <p className="text-sm text-muted-foreground">
                      You don&apos;t have permission to view all projects. Contact an admin to be
                      added as a project owner.
                    </p>
                  )}
                </EmptyContent>
              </Empty>
            ) : (
              <DataTable
                columns={projectsColumns}
                data={projects}
                isLoading={isLoading}
                emptyMessage="No projects found."
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
                onRowClick={(project) => navigate(`/projects/${project.id}`)}
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

        {/* Create Project Sheet */}
        <Sheet open={isCreateOpen} onOpenChange={setIsCreateOpen}>
          <SheetContent className="w-full sm:w-[540px] overflow-y-auto min-w-[600px]">
            <SheetHeader>
              <SheetTitle>Create Project</SheetTitle>
              <SheetDescription>
                Create a new project to organize database change requests.
              </SheetDescription>
            </SheetHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <label htmlFor="create-name" className="text-sm font-medium">
                  Project Name *
                </label>
                <Input
                  id="create-name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Enter project name"
                  aria-invalid={!!formErrors.name}
                />
                {formErrors.name && <p className="text-sm text-destructive">{formErrors.name}</p>}
              </div>
              <div className="space-y-2">
                <label htmlFor="create-description" className="text-sm font-medium">
                  Description *
                </label>
                <textarea
                  id="create-description"
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="Enter project description"
                  className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                  aria-invalid={!!formErrors.description}
                />
                {formErrors.description && (
                  <p className="text-sm text-destructive">{formErrors.description}</p>
                )}
              </div>
            </div>
            <SheetFooter>
              <Button variant="outline" onClick={() => setIsCreateOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleCreate} disabled={isCreating}>
                {isCreating ? 'Creating...' : 'Create Project'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>

        {/* Edit Project Sheet */}
        <Sheet open={isEditOpen} onOpenChange={setIsEditOpen}>
          <SheetContent className="w-full sm:w-[540px] overflow-y-auto min-w-[600px]">
            <SheetHeader>
              <SheetTitle>Edit Project</SheetTitle>
              <SheetDescription>Update project information.</SheetDescription>
            </SheetHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <label htmlFor="edit-name" className="text-sm font-medium">
                  Project Name *
                </label>
                <Input
                  id="edit-name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Enter project name"
                  aria-invalid={!!formErrors.name}
                />
                {formErrors.name && <p className="text-sm text-destructive">{formErrors.name}</p>}
              </div>
              <div className="space-y-2">
                <label htmlFor="edit-description" className="text-sm font-medium">
                  Description *
                </label>
                <textarea
                  id="edit-description"
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="Enter project description"
                  className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                  aria-invalid={!!formErrors.description}
                />
                {formErrors.description && (
                  <p className="text-sm text-destructive">{formErrors.description}</p>
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
          <SheetContent className="w-full sm:w-[540px]">
            <SheetHeader>
              <SheetTitle>Delete Project</SheetTitle>
              <SheetDescription>
                Are you sure you want to delete this project? This action cannot be undone.
              </SheetDescription>
            </SheetHeader>
            <div className="py-4">
              {selectedProject && (
                <div className="space-y-2">
                  <p className="text-sm font-medium">{selectedProject.name}</p>
                  <p className="text-sm text-muted-foreground">{selectedProject.description}</p>
                </div>
              )}
            </div>
            <SheetFooter>
              <Button variant="outline" onClick={() => setIsDeleteOpen(false)}>
                Cancel
              </Button>
              <Button variant="destructive" onClick={handleDelete} disabled={isDeleting}>
                {isDeleting ? 'Deleting...' : 'Delete Project'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>

        {/* Members Management Sheet */}
        <Sheet open={isMembersOpen} onOpenChange={setIsMembersOpen}>
          <SheetContent className="w-full sm:w-[640px] overflow-y-auto">
            <SheetHeader>
              <SheetTitle>Project Members</SheetTitle>
              <SheetDescription>
                Manage project members and owners. Only admins and project owners can manage
                members.
              </SheetDescription>
            </SheetHeader>
            <div className="space-y-4 py-4">
              {!canManageMembers && (
                <Card className="border-destructive">
                  <CardContent className="pt-6">
                    <div className="flex items-center gap-2 text-destructive">
                      <AlertTriangle className="h-4 w-4" />
                      <span className="text-sm">
                        You do not have permission to manage members. Only admins and project owners
                        can manage members.
                      </span>
                    </div>
                  </CardContent>
                </Card>
              )}

              {canManageMembers && (
                <>
                  <div className="flex items-center justify-between">
                    <h3 className="text-lg font-semibold">Members</h3>
                    <Button
                      onClick={() => {
                        setAddMemberForm({ user_id: '', project_role: 'ProjectDeveloper' });
                        setIsAddMemberOpen(true);
                      }}
                      size="sm"
                    >
                      <Plus className="mr-2 h-4 w-4" />
                      Add Member
                    </Button>
                  </div>

                  {isLoadingMembers ? (
                    <div className="space-y-2">
                      {[1, 2, 3].map((i) => (
                        <Skeleton key={i} className="h-16 w-full" />
                      ))}
                    </div>
                  ) : members.length === 0 ? (
                    <Empty>
                      <EmptyHeader>
                        <EmptyMedia variant="icon">
                          <Users className="h-12 w-12 text-muted-foreground" />
                        </EmptyMedia>
                        <EmptyTitle>No Members</EmptyTitle>
                        <EmptyDescription>
                          Add members to this project to grant them access.
                        </EmptyDescription>
                      </EmptyHeader>
                      <EmptyContent>
                        <Button
                          onClick={() => {
                            setAddMemberForm({ user_id: '', project_role: 'ProjectDeveloper' });
                            setIsAddMemberOpen(true);
                          }}
                        >
                          <Plus className="mr-2 h-4 w-4" />
                          Add Member
                        </Button>
                      </EmptyContent>
                    </Empty>
                  ) : (
                    <div className="overflow-x-auto -mx-6 px-6">
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>User</TableHead>
                            <TableHead>Role</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {members.map((memberWithUser) => (
                            <TableRow key={memberWithUser.member.id}>
                              <TableCell>
                                <div>
                                  <div className="font-medium">
                                    {memberWithUser.user.email || memberWithUser.member.user_id}
                                  </div>
                                  {memberWithUser.user.display_name && (
                                    <div className="text-sm text-muted-foreground">
                                      {memberWithUser.user.display_name}
                                    </div>
                                  )}
                                </div>
                              </TableCell>
                              <TableCell>
                                {canManageMembers ? (
                                  <Select
                                    value={memberWithUser.member.role}
                                    onValueChange={(value) =>
                                      handleUpdateRole(memberWithUser, value as ProjectRole)
                                    }
                                    disabled={isSubmittingMember}
                                  >
                                    <SelectTrigger className="w-[180px]">
                                      <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                      <SelectItem value="ProjectOwner">Project Owner</SelectItem>
                                      <SelectItem value="ProjectDeveloper">
                                        Project Developer
                                      </SelectItem>
                                      <SelectItem value="ProjectApprover">
                                        Project Approver
                                      </SelectItem>
                                      <SelectItem value="ProjectViewer">Project Viewer</SelectItem>
                                    </SelectContent>
                                  </Select>
                                ) : (
                                  <RoleBadge role={memberWithUser.member.role} showTooltip={true} />
                                )}
                              </TableCell>
                              <TableCell className="text-right">
                                {canManageMembers && (
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => {
                                      setSelectedMember(memberWithUser);
                                      setIsRemoveMemberOpen(true);
                                    }}
                                    className="h-8 w-8 p-0 text-destructive hover:text-destructive"
                                    aria-label="Remove member"
                                  >
                                    <Trash2 className="h-4 w-4" />
                                  </Button>
                                )}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </div>
                  )}
                </>
              )}
            </div>
          </SheetContent>
        </Sheet>

        {/* Add Member Dialog */}
        <Dialog open={isAddMemberOpen} onOpenChange={setIsAddMemberOpen}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Add Member</DialogTitle>
              <DialogDescription>
                Add a user to this project and assign them a role.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <label htmlFor="member-user" className="text-sm font-medium">
                  User *
                </label>
                <Select
                  value={addMemberForm.user_id}
                  onValueChange={(value) => setAddMemberForm({ ...addMemberForm, user_id: value })}
                  disabled={isLoadingUsers || isSubmittingMember}
                >
                  <SelectTrigger id="member-user">
                    <SelectValue placeholder="Select a user" />
                  </SelectTrigger>
                  <SelectContent>
                    {availableUsers
                      .filter((u) => !members.some((m) => m.member.user_id === u.id) && u.is_active)
                      .map((u) => (
                        <SelectItem key={u.id} value={u.id}>
                          {u.email}
                          {u.display_name && ` (${u.display_name})`}
                        </SelectItem>
                      ))}
                  </SelectContent>
                </Select>
                {availableUsers.filter(
                  (u) => !members.some((m) => m.member.user_id === u.id) && u.is_active,
                ).length === 0 && (
                  <p className="text-sm text-muted-foreground">
                    No available users to add. All users are already members.
                  </p>
                )}
              </div>
              <div className="space-y-2">
                <label htmlFor="member-role" className="text-sm font-medium">
                  Role *
                </label>
                <Select
                  value={addMemberForm.project_role}
                  onValueChange={(value) =>
                    setAddMemberForm({ ...addMemberForm, project_role: value as ProjectRole })
                  }
                  disabled={isSubmittingMember}
                >
                  <SelectTrigger id="member-role">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="ProjectOwner">Project Owner</SelectItem>
                    <SelectItem value="ProjectDeveloper">Project Developer</SelectItem>
                    <SelectItem value="ProjectApprover">Project Approver</SelectItem>
                    <SelectItem value="ProjectViewer">Project Viewer</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setIsAddMemberOpen(false)}>
                Cancel
              </Button>
              <Button
                onClick={handleAddMember}
                disabled={!addMemberForm.user_id || isSubmittingMember}
              >
                {isSubmittingMember ? 'Adding...' : 'Add Member'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Remove Member Dialog */}
        <Dialog open={isRemoveMemberOpen} onOpenChange={setIsRemoveMemberOpen}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Remove Member</DialogTitle>
              <DialogDescription>
                Are you sure you want to remove this member from the project? This action cannot be
                undone.
              </DialogDescription>
            </DialogHeader>
            <div className="py-4">
              {selectedMember && (
                <div className="space-y-2">
                  <p className="text-sm font-medium">
                    {selectedMember.user.email || selectedMember.member.user_id}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    Role: {selectedMember.member.role}
                  </p>
                </div>
              )}
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setIsRemoveMemberOpen(false)}>
                Cancel
              </Button>
              <Button
                variant="destructive"
                onClick={handleRemoveMember}
                disabled={isSubmittingMember}
              >
                {isSubmittingMember ? 'Removing...' : 'Remove Member'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </DefaultLayout>
  );
};
