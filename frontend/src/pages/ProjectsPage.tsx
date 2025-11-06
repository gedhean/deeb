import React, { useEffect, useState } from 'react';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { projectsService, projectMembersService } from '@/lib/api/services/projects';
import type {
  AddProjectMemberRequest,
  CreateProjectRequest,
  Project,
  ProjectMemberWithUser,
  ProjectRole,
  UpdateProjectRequest,
} from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Plus, Pencil, Trash2, AlertTriangle, FolderOpen, Users } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useProjectPermissions } from '@/hooks/api/useProjectPermissions';
import { usersService } from '@/lib/api/services/users';
import type { User } from '@/lib/api/types';
import { useProjects } from '@/hooks/api/useProjects';

export const ProjectsPage: React.FC = () => {
  const { user } = useAuth();
  const navigate = useNavigate();
  const { projects: availableProjects } = useProjects(); // Projects from hook (may include user's projects)
  const [projects, setProjects] = useState<Project[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [isMembersOpen, setIsMembersOpen] = useState(false);
  const [selectedProject, setSelectedProject] = useState<Project | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Members management state
  const [members, setMembers] = useState<ProjectMemberWithUser[]>([]);
  const [isLoadingMembers, setIsLoadingMembers] = useState(false);
  const [isAddMemberOpen, setIsAddMemberOpen] = useState(false);
  const [isRemoveMemberOpen, setIsRemoveMemberOpen] = useState(false);
  const [selectedMember, setSelectedMember] = useState<ProjectMemberWithUser | null>(null);
  const [availableUsers, setAvailableUsers] = useState<User[]>([]);
  const [isLoadingUsers, setIsLoadingUsers] = useState(false);
  const [addMemberForm, setAddMemberForm] = useState<AddProjectMemberRequest>({
    user_id: '',
    project_role: 'ProjectDeveloper',
  });

  // Form state
  const [formData, setFormData] = useState<CreateProjectRequest>({
    name: '',
    description: '',
  });
  const [formErrors, setFormErrors] = useState<{ name?: string; description?: string }>({});

  // Check permissions for members management
  const { canManageMembers } = useProjectPermissions(members);

  // Check if user is admin
  const isAdmin = user?.organization_role === 'OrganizationAdmin';

  // Load projects - admins see all, project owners fetch their owned projects
  useEffect(() => {
    if (!user) {
      setError('Authentication required.');
      setIsLoading(false);
      return;
    }

    if (isAdmin) {
      // Admins can list all projects
      loadProjects();
    } else {
      // For Project Owners, check projects from useProjects hook
      loadProjectsForOwner();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user, isAdmin, availableProjects]);

  // Load projects where user is owner
  const loadProjectsForOwner = async () => {
    try {
      setIsLoading(true);
      setError('');

      // Check projects from useProjects hook to see which ones user owns
      const userProjects: Project[] = [];
      for (const project of availableProjects) {
        try {
          const members = await projectMembersService.listProjectMembers(project.id);
          const isOwner = members.some(
            (m) => m.member.user_id === user?.id && m.member.role === 'ProjectOwner',
          );
          if (isOwner) {
            userProjects.push(project);
          }
        } catch {
          // Skip projects where we can't check membership or don't have access
        }
      }
      setProjects(userProjects);
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to load projects. Contact an admin to be added as a project owner.');
      }
      console.error('Failed to load projects:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const loadProjects = async () => {
    try {
      setIsLoading(true);
      setError('');
      const response = await projectsService.listProjects({ limit: 100 });
      setProjects(response.items);
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to load projects. Please try again.');
      }
      console.error('Failed to load projects:', err);
    } finally {
      setIsLoading(false);
    }
  };

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

    try {
      setIsSubmitting(true);
      await projectsService.createProject(formData);
      setIsCreateOpen(false);
      setFormData({ name: '', description: '' });
      setFormErrors({});
      showSuccessToast('Project created successfully');
      await loadProjects();
    } catch (err) {
      const errorMsg = handleApiError(err, {
        showToast: true,
        toastTitle: 'Failed to create project',
      });
      setError(errorMsg);
      console.error('Failed to create project:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Handle edit
  const handleEdit = async () => {
    if (!selectedProject || !validateForm()) {
      return;
    }

    try {
      setIsSubmitting(true);
      const updateData: UpdateProjectRequest = {};
      if (formData.name !== selectedProject.name) {
        updateData.name = formData.name;
      }
      if (formData.description !== selectedProject.description) {
        updateData.description = formData.description;
      }

      await projectsService.updateProject(selectedProject.id, updateData);
      setIsEditOpen(false);
      setSelectedProject(null);
      setFormData({ name: '', description: '' });
      setFormErrors({});
      showSuccessToast('Project updated successfully');
      await loadProjects();
    } catch (err) {
      const errorMsg = handleApiError(err, {
        showToast: true,
        toastTitle: 'Failed to update project',
      });
      setError(errorMsg);
      console.error('Failed to update project:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Handle delete
  const handleDelete = async () => {
    if (!selectedProject) {
      return;
    }

    try {
      setIsSubmitting(true);
      await projectsService.deleteProject(selectedProject.id);
      setIsDeleteOpen(false);
      setSelectedProject(null);
      showSuccessToast('Project deleted successfully');
      await loadProjects();
    } catch (err) {
      const errorMsg = handleApiError(err, {
        showToast: true,
        toastTitle: 'Failed to delete project',
      });
      setError(errorMsg);
      console.error('Failed to delete project:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Open create form
  const openCreateForm = () => {
    setFormData({ name: '', description: '' });
    setFormErrors({});
    setIsCreateOpen(true);
  };

  // Open edit form
  const openEditForm = (project: Project) => {
    setSelectedProject(project);
    setFormData({
      name: project.name,
      description: project.description,
    });
    setFormErrors({});
    setIsEditOpen(true);
  };

  // Open delete confirmation
  const openDeleteConfirmation = (project: Project) => {
    setSelectedProject(project);
    setIsDeleteOpen(true);
  };

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

  // Load available users for adding members
  const loadAvailableUsers = async () => {
    try {
      setIsLoadingUsers(true);
      const response = await usersService.listUsers({ limit: 100 });
      setAvailableUsers(response.items);
    } catch (err) {
      console.error('Failed to load users:', err);
    } finally {
      setIsLoadingUsers(false);
    }
  };

  // Handle add member
  const handleAddMember = async () => {
    if (!selectedProject || !addMemberForm.user_id) {
      return;
    }

    try {
      setIsSubmitting(true);
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
      setIsSubmitting(false);
    }
  };

  // Handle remove member
  const handleRemoveMember = async () => {
    if (!selectedProject || !selectedMember) {
      return;
    }

    try {
      setIsSubmitting(true);
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
      setIsSubmitting(false);
    }
  };

  // Handle update role
  const handleUpdateRole = async (member: ProjectMemberWithUser, newRole: ProjectRole) => {
    if (!selectedProject) {
      return;
    }

    try {
      setIsSubmitting(true);
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
      setIsSubmitting(false);
    }
  };

  // Get role badge color
  const getRoleBadgeVariant = (role: ProjectRole): 'default' | 'secondary' | 'destructive' => {
    switch (role) {
      case 'ProjectOwner':
        return 'default';
      case 'ProjectDeveloper':
        return 'secondary';
      case 'ProjectApprover':
        return 'secondary';
      case 'ProjectViewer':
        return 'secondary';
      default:
        return 'secondary';
    }
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

  // Non-admins can access but with limited functionality
  const canCreateProjects = isAdmin;

  return (
    <DefaultLayout breadcrumbs={[{ label: 'Projects', isCurrentPage: true }]}>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Projects</h1>
            <p className="text-muted-foreground mt-1">
              {isAdmin
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

        {/* Projects list */}
        {isLoading ? (
          <Card>
            <CardContent className="pt-6">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Description</TableHead>
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
                        <Skeleton className="h-4 w-48" />
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
        ) : projects.length === 0 ? (
          <Card>
            <CardContent className="pt-6">
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
                  {isAdmin ? (
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
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardContent className="pt-6">
              <div className="overflow-x-auto -mx-6 px-6">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Description</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {projects.map((project) => (
                      <TableRow
                        key={project.id}
                        onClick={() => navigate(`/projects/${project.id}`)}
                        className="cursor-pointer hover:bg-muted/50 transition-colors"
                      >
                        <TableCell className="font-medium">{project.name}</TableCell>
                        <TableCell className="max-w-md">
                          <span className="line-clamp-2 text-muted-foreground">
                            {project.description}
                          </span>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatDate(project.created_at)}
                        </TableCell>
                        <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                          <div className="flex items-center justify-end gap-2">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={(e) => {
                                e.stopPropagation();
                                navigate(`/projects/${project.id}`);
                              }}
                              className="h-11 w-11 p-0 touch-manipulation"
                              aria-label="View project details"
                            >
                              <Users className="h-4 w-4" />
                            </Button>
                            {isAdmin && (
                              <>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    openEditForm(project);
                                  }}
                                  className="h-11 w-11 p-0 touch-manipulation"
                                  aria-label="Edit project"
                                >
                                  <Pencil className="h-4 w-4" />
                                </Button>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    openDeleteConfirmation(project);
                                  }}
                                  className="h-11 w-11 p-0 text-destructive hover:text-destructive touch-manipulation"
                                  aria-label="Delete project"
                                >
                                  <Trash2 className="h-4 w-4" />
                                </Button>
                              </>
                            )}
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Create Project Sheet */}
        <Sheet open={isCreateOpen} onOpenChange={setIsCreateOpen}>
          <SheetContent className="w-full sm:w-[540px] overflow-y-auto">
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
              <Button onClick={handleCreate} disabled={isSubmitting}>
                {isSubmitting ? 'Creating...' : 'Create Project'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>

        {/* Edit Project Sheet */}
        <Sheet open={isEditOpen} onOpenChange={setIsEditOpen}>
          <SheetContent className="w-full sm:w-[540px] overflow-y-auto">
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
              <Button onClick={handleEdit} disabled={isSubmitting}>
                {isSubmitting ? 'Saving...' : 'Save Changes'}
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
              <Button variant="destructive" onClick={handleDelete} disabled={isSubmitting}>
                {isSubmitting ? 'Deleting...' : 'Delete Project'}
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
                      onClick={async () => {
                        setAddMemberForm({ user_id: '', project_role: 'ProjectDeveloper' });
                        await loadAvailableUsers();
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
                          onClick={async () => {
                            setAddMemberForm({ user_id: '', project_role: 'ProjectDeveloper' });
                            await loadAvailableUsers();
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
                                    disabled={isSubmitting}
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
                                  <Badge variant={getRoleBadgeVariant(memberWithUser.member.role)}>
                                    {memberWithUser.member.role}
                                  </Badge>
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
                  disabled={isLoadingUsers || isSubmitting}
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
                  disabled={isSubmitting}
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
              <Button onClick={handleAddMember} disabled={!addMemberForm.user_id || isSubmitting}>
                {isSubmitting ? 'Adding...' : 'Add Member'}
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
              <Button variant="destructive" onClick={handleRemoveMember} disabled={isSubmitting}>
                {isSubmitting ? 'Removing...' : 'Remove Member'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </DefaultLayout>
  );
};
