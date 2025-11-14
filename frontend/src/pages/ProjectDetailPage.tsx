import React, { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { projectsService, projectMembersService } from '@/lib/api/services/projects';
import type {
  Project,
  ProjectMemberWithUser,
  ProjectRole,
  AddProjectMemberRequest,
} from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { RoleBadge } from '@/components/RoleBadge';
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
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty';
import { useProjectPermissions } from '@/hooks/api/useProjectPermissions';
import { usersService } from '@/lib/api/services/users';
import type { User } from '@/lib/api/types';
import { showSuccessToast, handleApiError } from '@/lib/errorHandling';
import { ArrowLeft, AlertTriangle, Users, Plus, Trash2, Calendar } from 'lucide-react';

export const ProjectDetailPage: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { user } = useAuth();
  const [project, setProject] = useState<Project | null>(null);
  const [members, setMembers] = useState<ProjectMemberWithUser[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingMembers, setIsLoadingMembers] = useState(false);
  const [error, setError] = useState('');
  const [isAddMemberOpen, setIsAddMemberOpen] = useState(false);
  const [isRemoveMemberOpen, setIsRemoveMemberOpen] = useState(false);
  const [selectedMember, setSelectedMember] = useState<ProjectMemberWithUser | null>(null);
  const [availableUsers, setAvailableUsers] = useState<User[]>([]);
  const [isLoadingUsers, setIsLoadingUsers] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [addMemberForm, setAddMemberForm] = useState<AddProjectMemberRequest>({
    user_id: '',
    project_role: 'ProjectDeveloper',
  });

  // Check permissions for members management
  const { canManageMembers } = useProjectPermissions(members);

  useEffect(() => {
    if (id) {
      loadProject(id);
      loadMembers(id);
    }
  }, [id]);

  const loadProject = async (projectId: string) => {
    try {
      setIsLoading(true);
      setError('');
      const projectData = await projectsService.getProject(projectId);
      setProject(projectData);
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to load project. Please try again.');
      }
      console.error('Failed to load project:', err);
    } finally {
      setIsLoading(false);
    }
  };

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

  const handleAddMember = async () => {
    if (!id || !addMemberForm.user_id) {
      return;
    }

    try {
      setIsSubmitting(true);
      await projectMembersService.addProjectMember(id, addMemberForm);
      setIsAddMemberOpen(false);
      setAddMemberForm({ user_id: '', project_role: 'ProjectDeveloper' });
      showSuccessToast('Member added successfully');
      await loadMembers(id);
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

  const handleRemoveMember = async () => {
    if (!id || !selectedMember) {
      return;
    }

    try {
      setIsSubmitting(true);
      await projectMembersService.removeProjectMember(id, selectedMember.member.user_id);
      setIsRemoveMemberOpen(false);
      setSelectedMember(null);
      showSuccessToast('Member removed successfully');
      await loadMembers(id);
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

  const handleUpdateRole = async (member: ProjectMemberWithUser, newRole: ProjectRole) => {
    if (!id) {
      return;
    }

    try {
      setIsSubmitting(true);
      await projectMembersService.updateProjectMemberRole(id, member.member.user_id, newRole);
      showSuccessToast('Member role updated successfully');
      await loadMembers(id);
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

  const formatDate = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch {
      return dateString;
    }
  };

  if (isLoading) {
    return (
      <DefaultLayout
        breadcrumbs={[
          { label: 'Projects', href: '/projects' },
          { label: 'Loading...', isCurrentPage: true },
        ]}
      >
        <div className="space-y-6">
          <Card>
            <CardContent className="pt-6">
              <Skeleton className="h-8 w-64 mb-4" />
              <Skeleton className="h-4 w-full" />
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <Skeleton className="h-8 w-48 mb-4" />
              <div className="space-y-2">
                {[1, 2, 3].map((i) => (
                  <Skeleton key={i} className="h-16 w-full" />
                ))}
              </div>
            </CardContent>
          </Card>
        </div>
      </DefaultLayout>
    );
  }

  if (error || !project) {
    return (
      <DefaultLayout
        breadcrumbs={[
          { label: 'Projects', href: '/projects' },
          { label: 'Error', isCurrentPage: true },
        ]}
      >
        <Card>
          <CardHeader>
            <CardTitle>Error</CardTitle>
            <CardDescription>{error || 'Project not found'}</CardDescription>
          </CardHeader>
          <CardContent>
            <Button onClick={() => navigate('/projects')} variant="outline">
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back to Projects
            </Button>
          </CardContent>
        </Card>
      </DefaultLayout>
    );
  }

  return (
    <DefaultLayout
      breadcrumbs={[
        { label: 'Projects', href: '/projects' },
        { label: project.name, isCurrentPage: true },
      ]}
    >
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div>
              <div className="flex items-center gap-3">
                <h1 className="text-3xl font-bold tracking-tight">{project.name}</h1>
                {user && members.length > 0 && (
                  <RoleBadge
                    role={
                      members.find((m) => m.member.user_id === user.id)?.member.role as ProjectRole
                    }
                    showTooltip={true}
                    size="md"
                  />
                )}
              </div>
              <p className="text-muted-foreground mt-1">
                {project.description || 'No description'}
              </p>
            </div>
          </div>
        </div>

        {/* Project Details */}
        <Card>
          <CardHeader>
            <CardTitle>Project Details</CardTitle>
            <CardDescription>Overview of the project information</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-1">
                <div className="text-sm font-medium text-muted-foreground">Name</div>
                <div className="text-sm">{project.name}</div>
              </div>
              <div className="space-y-1">
                <div className="text-sm font-medium text-muted-foreground">Created</div>
                <div className="text-sm flex items-center gap-2">
                  <Calendar className="h-4 w-4" />
                  {formatDate(project.created_at)}
                </div>
              </div>
              <div className="space-y-1 md:col-span-2">
                <div className="text-sm font-medium text-muted-foreground">Description</div>
                <div className="text-sm">{project.description || 'No description provided'}</div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Members Section */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Project Members</CardTitle>
                <CardDescription>
                  Manage team members and their roles for this project
                </CardDescription>
              </div>
              {canManageMembers && (
                <Button
                  onClick={async () => {
                    await loadAvailableUsers();
                    setIsAddMemberOpen(true);
                  }}
                  size="sm"
                >
                  <Plus className="mr-2 h-4 w-4" />
                  Add Member
                </Button>
              )}
            </div>
          </CardHeader>
          <CardContent>
            {!canManageMembers && (
              <div className="mb-4 p-4 border border-destructive/50 rounded-md bg-destructive/10">
                <div className="flex items-center gap-2 text-destructive text-sm">
                  <AlertTriangle className="h-4 w-4" />
                  <span>
                    You do not have permission to manage members. Only admins and project owners can
                    manage members.
                  </span>
                </div>
              </div>
            )}

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
                {canManageMembers && (
                  <EmptyContent>
                    <Button
                      onClick={async () => {
                        await loadAvailableUsers();
                        setIsAddMemberOpen(true);
                      }}
                    >
                      <Plus className="mr-2 h-4 w-4" />
                      Add Member
                    </Button>
                  </EmptyContent>
                )}
              </Empty>
            ) : (
              <div className="overflow-x-auto -mx-6 px-6">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>User</TableHead>
                      <TableHead>Role</TableHead>
                      <TableHead>Added</TableHead>
                      {canManageMembers && <TableHead className="text-right">Actions</TableHead>}
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
                                <SelectItem value="ProjectDeveloper">Project Developer</SelectItem>
                                <SelectItem value="ProjectApprover">Project Approver</SelectItem>
                                <SelectItem value="ProjectViewer">Project Viewer</SelectItem>
                              </SelectContent>
                            </Select>
                          ) : (
                            <RoleBadge
                              role={memberWithUser.member.role as ProjectRole}
                              showTooltip={true}
                            />
                          )}
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatDate(memberWithUser.member.added_at)}
                        </TableCell>
                        {canManageMembers && (
                          <TableCell className="text-right">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => {
                                setSelectedMember(memberWithUser);
                                setIsRemoveMemberOpen(true);
                              }}
                              className="h-8 w-8 p-0 text-destructive hover:text-destructive"
                              aria-label="Remove member"
                              disabled={isSubmitting}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </TableCell>
                        )}
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>

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
