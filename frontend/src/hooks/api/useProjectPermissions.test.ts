import { renderHook } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useProjectPermissions } from './useProjectPermissions';
import { useAuth } from '@/contexts/AuthContext';
import type { ProjectMemberWithUser } from '@/lib/api/types';
import type { User } from '@/lib/api/types';

// Mock the AuthContext
vi.mock('@/contexts/AuthContext', () => ({
  useAuth: vi.fn(),
}));

describe('useProjectPermissions', () => {
  const mockUseAuth = useAuth as unknown as ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Helper function to create mock user
  const createMockUser = (
    id: string,
    organizationRole: 'OrganizationAdmin' | 'OrganizationDBA' | 'OrganizationDeveloper',
  ): User => ({
    id,
    email: `user-${id}@example.com`,
    organization_role: organizationRole,
    is_active: true,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  });

  // Helper function to create mock project member
  const createMockProjectMember = (
    userId: string,
    projectRole: 'ProjectOwner' | 'ProjectDeveloper' | 'ProjectApprover' | 'ProjectViewer',
  ): ProjectMemberWithUser => ({
    member: {
      id: `member-${userId}`,
      project_id: 'project-1',
      user_id: userId,
      role: projectRole,
      added_by_id: 'admin-1',
      added_at: '2024-01-01T00:00:00Z',
    },
    user: {
      id: userId,
      email: `user-${userId}@example.com`,
    },
  });

  describe('Admin users', () => {
    it('should grant all permissions to admin users regardless of project role', () => {
      const adminUser = createMockUser('admin-1', 'OrganizationAdmin');
      mockUseAuth.mockReturnValue({ user: adminUser });

      const { result } = renderHook(() => useProjectPermissions([]));

      expect(result.current.canManageMembers).toBe(true);
      expect(result.current.canManageSettings).toBe(true);
      expect(result.current.canCreateChanges).toBe(true);
      expect(result.current.canUpdateChanges).toBe(true);
      expect(result.current.canDeleteChanges).toBe(true);
      expect(result.current.canApproveChanges).toBe(true);
      expect(result.current.canRejectChanges).toBe(true);
      expect(result.current.canExecuteChanges).toBe(true);
      expect(result.current.canManagePolicyRules).toBe(true);
      expect(result.current.isProjectOwner).toBe(false);
      expect(result.current.userProjectRole).toBe(null);
    });
  });

  describe('ProjectOwner role', () => {
    it('should grant all project permissions to ProjectOwner', () => {
      const user = createMockUser('user-1', 'OrganizationDeveloper');
      const projectMembers = [createMockProjectMember('user-1', 'ProjectOwner')];

      mockUseAuth.mockReturnValue({ user });

      const { result } = renderHook(() => useProjectPermissions(projectMembers));

      expect(result.current.canManageMembers).toBe(true);
      expect(result.current.canManageSettings).toBe(true);
      expect(result.current.canCreateChanges).toBe(true);
      expect(result.current.canUpdateChanges).toBe(true);
      expect(result.current.canDeleteChanges).toBe(true);
      expect(result.current.canApproveChanges).toBe(true);
      expect(result.current.canRejectChanges).toBe(true);
      expect(result.current.canExecuteChanges).toBe(true);
      expect(result.current.canManagePolicyRules).toBe(true);
      expect(result.current.isProjectOwner).toBe(true);
      expect(result.current.userProjectRole).toBe('ProjectOwner');
    });
  });

  describe('ProjectDeveloper role', () => {
    it('should grant appropriate permissions to ProjectDeveloper', () => {
      const user = createMockUser('user-2', 'OrganizationDeveloper');
      const projectMembers = [createMockProjectMember('user-2', 'ProjectDeveloper')];

      mockUseAuth.mockReturnValue({ user });

      const { result } = renderHook(() => useProjectPermissions(projectMembers));

      // ProjectDeveloper can create, update, and execute changes (per YAML config)
      expect(result.current.canCreateChanges).toBe(true);
      expect(result.current.canUpdateChanges).toBe(true);
      expect(result.current.canExecuteChanges).toBe(true);

      // ProjectDeveloper cannot delete changes (not in YAML config)
      expect(result.current.canDeleteChanges).toBe(false);

      // ProjectDeveloper cannot manage members, settings, approve/reject, or manage policy rules
      expect(result.current.canManageMembers).toBe(false);
      expect(result.current.canManageSettings).toBe(false);
      expect(result.current.canApproveChanges).toBe(false);
      expect(result.current.canRejectChanges).toBe(false);
      expect(result.current.canManagePolicyRules).toBe(false);

      expect(result.current.isProjectOwner).toBe(false);
      expect(result.current.userProjectRole).toBe('ProjectDeveloper');
    });
  });

  describe('ProjectApprover role', () => {
    it('should grant approval permissions to ProjectApprover', () => {
      const user = createMockUser('user-3', 'OrganizationDeveloper');
      const projectMembers = [createMockProjectMember('user-3', 'ProjectApprover')];

      mockUseAuth.mockReturnValue({ user });

      const { result } = renderHook(() => useProjectPermissions(projectMembers));

      // ProjectApprover can approve and reject changes
      expect(result.current.canApproveChanges).toBe(true);
      expect(result.current.canRejectChanges).toBe(true);

      // ProjectApprover cannot create, update, delete, or execute changes
      expect(result.current.canCreateChanges).toBe(false);
      expect(result.current.canUpdateChanges).toBe(false);
      expect(result.current.canDeleteChanges).toBe(false);
      expect(result.current.canExecuteChanges).toBe(false);

      // ProjectApprover cannot manage members, settings, or policy rules
      expect(result.current.canManageMembers).toBe(false);
      expect(result.current.canManageSettings).toBe(false);
      expect(result.current.canManagePolicyRules).toBe(false);

      expect(result.current.isProjectOwner).toBe(false);
      expect(result.current.userProjectRole).toBe('ProjectApprover');
    });
  });

  describe('ProjectViewer role', () => {
    it('should grant no modification permissions to ProjectViewer', () => {
      const user = createMockUser('user-4', 'OrganizationDeveloper');
      const projectMembers = [createMockProjectMember('user-4', 'ProjectViewer')];

      mockUseAuth.mockReturnValue({ user });

      const { result } = renderHook(() => useProjectPermissions(projectMembers));

      // ProjectViewer cannot perform any modification actions
      expect(result.current.canManageMembers).toBe(false);
      expect(result.current.canManageSettings).toBe(false);
      expect(result.current.canCreateChanges).toBe(false);
      expect(result.current.canUpdateChanges).toBe(false);
      expect(result.current.canDeleteChanges).toBe(false);
      expect(result.current.canApproveChanges).toBe(false);
      expect(result.current.canRejectChanges).toBe(false);
      expect(result.current.canExecuteChanges).toBe(false);
      expect(result.current.canManagePolicyRules).toBe(false);

      expect(result.current.isProjectOwner).toBe(false);
      expect(result.current.userProjectRole).toBe('ProjectViewer');
    });
  });

  describe('Non-member users', () => {
    it('should grant no permissions to users not in the project', () => {
      const user = createMockUser('user-5', 'OrganizationDeveloper');
      const projectMembers = [createMockProjectMember('other-user', 'ProjectOwner')];

      mockUseAuth.mockReturnValue({ user });

      const { result } = renderHook(() => useProjectPermissions(projectMembers));

      // Non-member has no permissions
      expect(result.current.canManageMembers).toBe(false);
      expect(result.current.canManageSettings).toBe(false);
      expect(result.current.canCreateChanges).toBe(false);
      expect(result.current.canUpdateChanges).toBe(false);
      expect(result.current.canDeleteChanges).toBe(false);
      expect(result.current.canApproveChanges).toBe(false);
      expect(result.current.canRejectChanges).toBe(false);
      expect(result.current.canExecuteChanges).toBe(false);
      expect(result.current.canManagePolicyRules).toBe(false);

      expect(result.current.isProjectOwner).toBe(false);
      expect(result.current.userProjectRole).toBe(null);
    });
  });

  describe('Edge cases', () => {
    it('should handle undefined project members', () => {
      const user = createMockUser('user-6', 'OrganizationDeveloper');
      mockUseAuth.mockReturnValue({ user });

      const { result } = renderHook(() => useProjectPermissions(undefined));

      expect(result.current.canManageMembers).toBe(false);
      expect(result.current.userProjectRole).toBe(null);
    });

    it('should handle undefined user', () => {
      mockUseAuth.mockReturnValue({ user: undefined });

      const { result } = renderHook(() => useProjectPermissions([]));

      expect(result.current.canManageMembers).toBe(false);
      expect(result.current.userProjectRole).toBe(null);
    });

    it('should handle empty project members array', () => {
      const user = createMockUser('user-7', 'OrganizationDeveloper');
      mockUseAuth.mockReturnValue({ user });

      const { result } = renderHook(() => useProjectPermissions([]));

      expect(result.current.canManageMembers).toBe(false);
      expect(result.current.userProjectRole).toBe(null);
    });
  });

  describe('Memoization', () => {
    it('should memoize results when inputs do not change', () => {
      const user = createMockUser('user-8', 'OrganizationDeveloper');
      const projectMembers = [createMockProjectMember('user-8', 'ProjectOwner')];

      mockUseAuth.mockReturnValue({ user });

      const { result, rerender } = renderHook(() => useProjectPermissions(projectMembers));

      const firstResult = result.current;
      rerender();
      const secondResult = result.current;

      // Results should be the same object due to memoization
      expect(firstResult).toBe(secondResult);
    });

    it('should recompute when user changes', () => {
      const user1 = createMockUser('user-9', 'OrganizationDeveloper');
      const user2 = createMockUser('user-10', 'OrganizationAdmin');
      const projectMembers = [createMockProjectMember('user-9', 'ProjectViewer')];

      mockUseAuth.mockReturnValue({ user: user1 });

      const { result, rerender } = renderHook(() => useProjectPermissions(projectMembers));

      expect(result.current.canManageMembers).toBe(false);

      // Change user to admin
      mockUseAuth.mockReturnValue({ user: user2 });
      rerender();

      expect(result.current.canManageMembers).toBe(true);
    });

    it('should recompute when project members change', () => {
      const user = createMockUser('user-11', 'OrganizationDeveloper');
      const projectMembers1 = [createMockProjectMember('user-11', 'ProjectViewer')];
      const projectMembers2 = [createMockProjectMember('user-11', 'ProjectOwner')];

      mockUseAuth.mockReturnValue({ user });

      const { result, rerender } = renderHook(({ members }) => useProjectPermissions(members), {
        initialProps: { members: projectMembers1 },
      });

      expect(result.current.canManageMembers).toBe(false);
      expect(result.current.userProjectRole).toBe('ProjectViewer');

      // Change project role
      rerender({ members: projectMembers2 });

      expect(result.current.canManageMembers).toBe(true);
      expect(result.current.userProjectRole).toBe('ProjectOwner');
    });
  });
});
