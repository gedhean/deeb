import { describe, it, expect } from 'vitest';
import {
  hasOrganizationPermission,
  hasProjectPermission,
  hasOrganizationRole,
  hasAnyOrganizationRole,
  hasProjectRole,
  hasAnyProjectRole,
  isValidOrganizationRole,
  isValidProjectRole,
  isAdmin,
  isDBA,
  isDeveloper,
  getOrganizationPermissions,
  getProjectPermissions,
  OrganizationPermission,
  ProjectPermission,
  type OrganizationRole,
  type ProjectRole,
} from '../lib/rbac';

describe('RBAC Utilities', () => {
  describe('hasOrganizationPermission', () => {
    it('should return true for admin with any permission', () => {
      expect(
        hasOrganizationPermission('OrganizationAdmin', OrganizationPermission.UsersCreate),
      ).toBe(true);
      expect(
        hasOrganizationPermission('OrganizationAdmin', OrganizationPermission.AuditLogsRead),
      ).toBe(true);
    });

    it('should return true for DBA with database permissions', () => {
      expect(
        hasOrganizationPermission('OrganizationDBA', OrganizationPermission.DatabasesCreate),
      ).toBe(true);
      expect(
        hasOrganizationPermission('OrganizationDBA', OrganizationPermission.DatabasesRead),
      ).toBe(true);
    });

    it('should return false for DBA without permission', () => {
      expect(hasOrganizationPermission('OrganizationDBA', OrganizationPermission.UsersCreate)).toBe(
        false,
      );
      // Note: OrganizationDBA now has projects.create permission per YAML config
      expect(hasOrganizationPermission('OrganizationDBA', OrganizationPermission.UsersUpdate)).toBe(
        false,
      );
    });

    it('should return true for Developer with allowed permissions', () => {
      expect(
        hasOrganizationPermission('OrganizationDeveloper', OrganizationPermission.ProjectsCreate),
      ).toBe(true);
      expect(
        hasOrganizationPermission('OrganizationDeveloper', OrganizationPermission.ProjectsRead),
      ).toBe(true);
    });

    it('should return false for Developer without permission', () => {
      expect(
        hasOrganizationPermission('OrganizationDeveloper', OrganizationPermission.UsersCreate),
      ).toBe(false);
      // Note: OrganizationDeveloper now has databases.create and databases.update permission per YAML config
      expect(
        hasOrganizationPermission('OrganizationDeveloper', OrganizationPermission.DatabasesDelete),
      ).toBe(false);
    });

    it('should return false for undefined role', () => {
      expect(hasOrganizationPermission(undefined, OrganizationPermission.UsersRead)).toBe(false);
    });

    it('should return false for invalid role', () => {
      expect(
        hasOrganizationPermission(
          'InvalidRole' as OrganizationRole,
          OrganizationPermission.UsersRead,
        ),
      ).toBe(false);
    });
  });

  describe('hasProjectPermission', () => {
    it('should return true for ProjectOwner with any permission', () => {
      expect(hasProjectPermission('ProjectOwner', ProjectPermission.ChangesCreate)).toBe(true);
      expect(hasProjectPermission('ProjectOwner', ProjectPermission.ChangesApprove)).toBe(true);
      expect(hasProjectPermission('ProjectOwner', ProjectPermission.ChangesExecute)).toBe(true);
    });

    it('should return true for ProjectDeveloper with allowed permissions', () => {
      expect(hasProjectPermission('ProjectDeveloper', ProjectPermission.ChangesCreate)).toBe(true);
      expect(hasProjectPermission('ProjectDeveloper', ProjectPermission.ChangesRead)).toBe(true);
    });

    it('should return false for ProjectDeveloper without permission', () => {
      // Note: ProjectDeveloper now has changes.approve, changes.reject, and changes.execute per YAML config
      expect(
        hasProjectPermission('ProjectDeveloper', ProjectPermission.ProjectSettingsUpdate),
      ).toBe(false);
      expect(hasProjectPermission('ProjectDeveloper', ProjectPermission.ChangesDelete)).toBe(false);
    });

    it('should return true for ProjectApprover with approval permissions', () => {
      expect(hasProjectPermission('ProjectApprover', ProjectPermission.ChangesApprove)).toBe(true);
      expect(hasProjectPermission('ProjectApprover', ProjectPermission.ChangesReject)).toBe(true);
    });

    it('should return false for ProjectApprover without permission', () => {
      expect(hasProjectPermission('ProjectApprover', ProjectPermission.ChangesCreate)).toBe(false);
      expect(hasProjectPermission('ProjectApprover', ProjectPermission.ChangesExecute)).toBe(false);
    });

    it('should return true for ProjectViewer with read permissions', () => {
      expect(hasProjectPermission('ProjectViewer', ProjectPermission.ChangesRead)).toBe(true);
    });

    it('should return false for ProjectViewer without permission', () => {
      expect(hasProjectPermission('ProjectViewer', ProjectPermission.ChangesCreate)).toBe(false);
      expect(hasProjectPermission('ProjectViewer', ProjectPermission.ChangesApprove)).toBe(false);
    });

    it('should return false for undefined role', () => {
      expect(hasProjectPermission(undefined, ProjectPermission.ChangesRead)).toBe(false);
    });

    it('should return false for invalid role', () => {
      expect(
        hasProjectPermission('InvalidRole' as ProjectRole, ProjectPermission.ChangesRead),
      ).toBe(false);
    });
  });

  describe('hasOrganizationRole', () => {
    it('should return true for matching role', () => {
      expect(hasOrganizationRole('OrganizationAdmin', 'OrganizationAdmin')).toBe(true);
      expect(hasOrganizationRole('OrganizationDBA', 'OrganizationDBA')).toBe(true);
    });

    it('should return false for non-matching role', () => {
      expect(hasOrganizationRole('OrganizationAdmin', 'OrganizationDBA')).toBe(false);
    });

    it('should return false for undefined role', () => {
      expect(hasOrganizationRole(undefined, 'OrganizationAdmin')).toBe(false);
    });
  });

  describe('hasAnyOrganizationRole', () => {
    it('should return true if user has one of the required roles', () => {
      expect(
        hasAnyOrganizationRole('OrganizationAdmin', ['OrganizationAdmin', 'OrganizationDBA']),
      ).toBe(true);
      expect(
        hasAnyOrganizationRole('OrganizationDBA', ['OrganizationAdmin', 'OrganizationDBA']),
      ).toBe(true);
    });

    it('should return false if user does not have any required role', () => {
      expect(
        hasAnyOrganizationRole('OrganizationDeveloper', ['OrganizationAdmin', 'OrganizationDBA']),
      ).toBe(false);
    });

    it('should return false for undefined role', () => {
      expect(hasAnyOrganizationRole(undefined, ['OrganizationAdmin'])).toBe(false);
    });
  });

  describe('hasProjectRole', () => {
    it('should return true for matching role', () => {
      expect(hasProjectRole('ProjectOwner', 'ProjectOwner')).toBe(true);
      expect(hasProjectRole('ProjectDeveloper', 'ProjectDeveloper')).toBe(true);
    });

    it('should return false for non-matching role', () => {
      expect(hasProjectRole('ProjectOwner', 'ProjectDeveloper')).toBe(false);
    });

    it('should return false for undefined role', () => {
      expect(hasProjectRole(undefined, 'ProjectOwner')).toBe(false);
    });
  });

  describe('hasAnyProjectRole', () => {
    it('should return true if user has one of the required roles', () => {
      expect(hasAnyProjectRole('ProjectOwner', ['ProjectOwner', 'ProjectApprover'])).toBe(true);
      expect(hasAnyProjectRole('ProjectApprover', ['ProjectOwner', 'ProjectApprover'])).toBe(true);
    });

    it('should return false if user does not have any required role', () => {
      expect(hasAnyProjectRole('ProjectViewer', ['ProjectOwner', 'ProjectApprover'])).toBe(false);
    });

    it('should return false for undefined role', () => {
      expect(hasAnyProjectRole(undefined, ['ProjectOwner'])).toBe(false);
    });
  });

  describe('isValidOrganizationRole', () => {
    it('should return true for valid roles', () => {
      expect(isValidOrganizationRole('OrganizationAdmin')).toBe(true);
      expect(isValidOrganizationRole('OrganizationDBA')).toBe(true);
      expect(isValidOrganizationRole('OrganizationDeveloper')).toBe(true);
    });

    it('should return false for invalid roles', () => {
      expect(isValidOrganizationRole('InvalidRole')).toBe(false);
      expect(isValidOrganizationRole('')).toBe(false);
    });
  });

  describe('isValidProjectRole', () => {
    it('should return true for valid roles', () => {
      expect(isValidProjectRole('ProjectOwner')).toBe(true);
      expect(isValidProjectRole('ProjectDeveloper')).toBe(true);
      expect(isValidProjectRole('ProjectApprover')).toBe(true);
      expect(isValidProjectRole('ProjectViewer')).toBe(true);
    });

    it('should return false for invalid roles', () => {
      expect(isValidProjectRole('InvalidRole')).toBe(false);
      expect(isValidProjectRole('')).toBe(false);
    });
  });

  describe('isAdmin', () => {
    it('should return true for admin role', () => {
      expect(isAdmin('OrganizationAdmin')).toBe(true);
    });

    it('should return false for non-admin roles', () => {
      expect(isAdmin('OrganizationDBA')).toBe(false);
      expect(isAdmin('OrganizationDeveloper')).toBe(false);
    });

    it('should return false for undefined role', () => {
      expect(isAdmin(undefined)).toBe(false);
    });
  });

  describe('isDBA', () => {
    it('should return true for DBA role', () => {
      expect(isDBA('OrganizationDBA')).toBe(true);
    });

    it('should return false for non-DBA roles', () => {
      expect(isDBA('OrganizationAdmin')).toBe(false);
      expect(isDBA('OrganizationDeveloper')).toBe(false);
    });

    it('should return false for undefined role', () => {
      expect(isDBA(undefined)).toBe(false);
    });
  });

  describe('isDeveloper', () => {
    it('should return true for developer role', () => {
      expect(isDeveloper('OrganizationDeveloper')).toBe(true);
    });

    it('should return false for non-developer roles', () => {
      expect(isDeveloper('OrganizationAdmin')).toBe(false);
      expect(isDeveloper('OrganizationDBA')).toBe(false);
    });

    it('should return false for undefined role', () => {
      expect(isDeveloper(undefined)).toBe(false);
    });
  });

  describe('getOrganizationPermissions', () => {
    it('should return all permissions for admin', () => {
      const permissions = getOrganizationPermissions('OrganizationAdmin');
      expect(permissions).toContain(OrganizationPermission.All);
    });

    it('should return correct permissions for DBA', () => {
      const permissions = getOrganizationPermissions('OrganizationDBA');
      expect(permissions).toContain(OrganizationPermission.DatabasesCreate);
      expect(permissions).toContain(OrganizationPermission.DatabasesRead);
      expect(permissions).not.toContain(OrganizationPermission.UsersCreate);
    });

    it('should return correct permissions for Developer', () => {
      const permissions = getOrganizationPermissions('OrganizationDeveloper');
      expect(permissions).toContain(OrganizationPermission.ProjectsCreate);
      expect(permissions).toContain(OrganizationPermission.ProjectsRead);
      // Note: OrganizationDeveloper now has databases.create and databases.update permission per YAML config
      expect(permissions).toContain(OrganizationPermission.DatabasesCreate);
      expect(permissions).toContain(OrganizationPermission.DatabasesUpdate);
      expect(permissions).not.toContain(OrganizationPermission.DatabasesDelete);
    });
  });

  describe('getProjectPermissions', () => {
    it('should return all permissions for ProjectOwner', () => {
      const permissions = getProjectPermissions('ProjectOwner');
      expect(permissions).toContain(ProjectPermission.ChangesCreate);
      expect(permissions).toContain(ProjectPermission.ChangesApprove);
      expect(permissions).toContain(ProjectPermission.ChangesExecute);
    });

    it('should return correct permissions for ProjectDeveloper', () => {
      const permissions = getProjectPermissions('ProjectDeveloper');
      expect(permissions).toContain(ProjectPermission.ChangesCreate);
      expect(permissions).toContain(ProjectPermission.ChangesRead);
      // Note: ProjectDeveloper now has approve/reject/execute permissions per YAML config
      expect(permissions).toContain(ProjectPermission.ChangesApprove);
      expect(permissions).toContain(ProjectPermission.ChangesReject);
      expect(permissions).toContain(ProjectPermission.ChangesExecute);
      expect(permissions).not.toContain(ProjectPermission.ChangesDelete);
      expect(permissions).not.toContain(ProjectPermission.ProjectSettingsUpdate);
    });

    it('should return correct permissions for ProjectApprover', () => {
      const permissions = getProjectPermissions('ProjectApprover');
      expect(permissions).toContain(ProjectPermission.ChangesApprove);
      expect(permissions).toContain(ProjectPermission.ChangesReject);
      expect(permissions).not.toContain(ProjectPermission.ChangesCreate);
    });

    it('should return correct permissions for ProjectViewer', () => {
      const permissions = getProjectPermissions('ProjectViewer');
      expect(permissions).toContain(ProjectPermission.ChangesRead);
      expect(permissions).not.toContain(ProjectPermission.ChangesCreate);
      expect(permissions).not.toContain(ProjectPermission.ChangesApprove);
    });
  });
});
