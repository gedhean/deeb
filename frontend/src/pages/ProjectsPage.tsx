import React, { useEffect, useState } from 'react';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { projectsService } from '@/lib/api/services/projects';
import type { CreateProjectRequest, Project, UpdateProjectRequest } from '@/lib/api/types';
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Plus, Pencil, Trash2, AlertTriangle } from 'lucide-react';

export const ProjectsPage: React.FC = () => {
  const { user } = useAuth();
  const [projects, setProjects] = useState<Project[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [selectedProject, setSelectedProject] = useState<Project | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Form state
  const [formData, setFormData] = useState<CreateProjectRequest>({
    name: '',
    description: '',
  });
  const [formErrors, setFormErrors] = useState<{ name?: string; description?: string }>({});

  // Check if user is admin
  const isAdmin = user?.organization_role === 'OrganizationAdmin';

  // Load projects
  useEffect(() => {
    if (!isAdmin) {
      setError('Access denied. Admin access required.');
      setIsLoading(false);
      return;
    }

    loadProjects();
  }, [isAdmin]);

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
      await loadProjects();
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to create project. Please try again.');
      }
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
      await loadProjects();
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to update project. Please try again.');
      }
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
      await loadProjects();
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to delete project. Please try again.');
      }
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
  if (!isAdmin) {
    return (
      <DefaultLayout breadcrumbs={[{ label: 'Projects', isCurrentPage: true }]}>
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
    <DefaultLayout breadcrumbs={[{ label: 'Projects', isCurrentPage: true }]}>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Projects</h1>
            <p className="text-muted-foreground mt-1">
              Manage projects and organize database change requests
            </p>
          </div>
          <Button onClick={openCreateForm}>
            <Plus className="mr-2 h-4 w-4" />
            Create Project
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
              <div className="text-center py-8">
                <p className="text-muted-foreground">No projects found.</p>
                <Button onClick={openCreateForm} className="mt-4">
                  <Plus className="mr-2 h-4 w-4" />
                  Create Your First Project
                </Button>
              </div>
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
                    <TableHead>Created</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {projects.map((project) => (
                    <TableRow key={project.id}>
                      <TableCell className="font-medium">{project.name}</TableCell>
                      <TableCell className="max-w-md">
                        <span className="line-clamp-2 text-muted-foreground">
                          {project.description}
                        </span>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {formatDate(project.created_at)}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-2">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => openEditForm(project)}
                            className="h-8 w-8 p-0"
                          >
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => openDeleteConfirmation(project)}
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

        {/* Create Project Sheet */}
        <Sheet open={isCreateOpen} onOpenChange={setIsCreateOpen}>
          <SheetContent>
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
          <SheetContent>
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
          <SheetContent>
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
      </div>
    </DefaultLayout>
  );
};
