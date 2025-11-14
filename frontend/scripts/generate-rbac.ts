/**
 * RBAC Code Generator
 *
 * Generates TypeScript RBAC utilities from YAML configuration files.
 * This ensures frontend and backend stay in sync with a single source of truth.
 *
 * Usage: tsx scripts/generate-rbac.ts
 */

/* eslint-disable no-undef */

import * as fs from 'fs';
import * as path from 'path';
import { fileURLToPath } from 'url';
import * as yaml from 'js-yaml';

// ES module __dirname equivalent
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Interfaces matching the YAML structure
interface PermissionsYaml {
  permissions: string[];
}

interface Role {
  name: string;
  title: string;
  description: string;
  permissions: string[];
}

interface RolesYaml {
  roles: Role[];
}

// Configuration
const BACKEND_IAM_DIR = path.resolve(__dirname, '../../backend/config/iam');
const PERMISSIONS_YAML = path.join(BACKEND_IAM_DIR, 'permissions.yaml');
const ROLES_YAML = path.join(BACKEND_IAM_DIR, 'roles.yaml');
const OUTPUT_FILE = path.resolve(__dirname, '../src/lib/rbac.generated.ts');

/**
 * Convert permission string to constant name
 * Examples:
 *   "users.create" -> "UsersCreate"
 *   "database_instances.read" -> "DatabaseInstancesRead"
 *   "project.members.create" -> "ProjectMembersCreate"
 *   "*" -> "All"
 */
function toConstantName(permission: string): string {
  // Handle wildcard permission
  if (permission === '*') {
    return 'All';
  }

  return permission
    .split('.')
    .map((part) =>
      part
        .split('_')
        .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
        .join(''),
    )
    .join('');
}

/**
 * Check if permission is organization-level or project-level
 */
function isProjectPermission(permission: string): boolean {
  return (
    permission.startsWith('project.') ||
    permission.startsWith('changes.') ||
    permission.startsWith('policy_rules.')
  );
}

/**
 * Check if role is organization-level or project-level
 */
function isProjectRole(roleName: string): boolean {
  return roleName.startsWith('Project');
}

/**
 * Load and parse YAML files
 */
function loadYamlFiles(): { permissions: string[]; roles: Role[] } {
  // Load permissions.yaml
  const permissionsContent = fs.readFileSync(PERMISSIONS_YAML, 'utf8');
  const permissionsData = yaml.load(permissionsContent) as PermissionsYaml;

  // Load roles.yaml
  const rolesContent = fs.readFileSync(ROLES_YAML, 'utf8');
  const rolesData = yaml.load(rolesContent) as RolesYaml;

  return {
    permissions: permissionsData.permissions,
    roles: rolesData.roles,
  };
}

/**
 * Validate YAML structure
 */
function validateYaml(permissions: string[], roles: Role[]): void {
  const errors: string[] = [];

  // Validate permissions format
  permissions.forEach((perm) => {
    if (perm !== '*' && !perm.match(/^[a-z_]+\.[a-z_]+(\.[a-z_]+)?$/)) {
      errors.push(`Invalid permission format: ${perm}`);
    }
  });

  // Validate roles
  roles.forEach((role) => {
    if (!role.name || !role.title || !role.description || !role.permissions) {
      errors.push(`Invalid role structure for: ${role.name || 'unknown'}`);
    }

    // Validate role permissions exist
    role.permissions.forEach((perm) => {
      if (perm !== '*' && !permissions.includes(perm)) {
        errors.push(`Role ${role.name} references unknown permission: ${perm}`);
      }
    });
  });

  if (errors.length > 0) {
    console.error('YAML Validation Errors:');
    errors.forEach((err) => console.error(`  - ${err}`));
    process.exit(1);
  }
}

/**
 * Generate TypeScript code for permissions
 */
function generatePermissions(permissions: string[]): string {
  const orgPermissions = permissions.filter((p) => !isProjectPermission(p));
  const projPermissions = permissions.filter((p) => isProjectPermission(p));

  let code = '';

  // Generate OrganizationPermissionValues
  code += '// Organization-level permissions\n';
  code += 'const OrganizationPermissionValues = {\n';
  orgPermissions.forEach((perm) => {
    const constName = toConstantName(perm);
    code += `  ${constName}: '${perm}',\n`;
  });
  code += '} as const;\n\n';

  code += 'export const OrganizationPermission = OrganizationPermissionValues;\n';
  code += '// eslint-disable-next-line no-redeclare\n';
  code += 'export type OrganizationPermission =\n';
  code += '  (typeof OrganizationPermissionValues)[keyof typeof OrganizationPermissionValues];\n\n';

  // Generate ProjectPermissionValues
  code += '// Project-level permissions\n';
  code += 'const ProjectPermissionValues = {\n';
  projPermissions.forEach((perm) => {
    const constName = toConstantName(perm);
    code += `  ${constName}: '${perm}',\n`;
  });
  code += '} as const;\n\n';

  code += 'export const ProjectPermission = ProjectPermissionValues;\n';
  code += '// eslint-disable-next-line no-redeclare\n';
  code += 'export type ProjectPermission =\n';
  code += '  (typeof ProjectPermissionValues)[keyof typeof ProjectPermissionValues];\n\n';

  return code;
}

/**
 * Generate TypeScript code for role types
 */
function generateRoleTypes(roles: Role[]): string {
  const orgRoles = roles.filter((r) => !isProjectRole(r.name));
  const projRoles = roles.filter((r) => isProjectRole(r.name));

  let code = '';

  // Generate OrganizationRole type
  code += '// Organization roles\n';
  code += `export type OrganizationRole = ${orgRoles.map((r) => `'${r.name}'`).join(' | ')};\n\n`;

  // Generate ProjectRole type
  code += '// Project roles\n';
  code += `export type ProjectRole = ${projRoles.map((r) => `'${r.name}'`).join(' | ')};\n\n`;

  return code;
}

/**
 * Generate TypeScript code for role permission mappings
 */
function generateRoleMappings(roles: Role[]): string {
  const orgRoles = roles.filter((r) => !isProjectRole(r.name));
  const projRoles = roles.filter((r) => isProjectRole(r.name));

  let code = '';

  // Generate organizationRolePermissions mapping
  code += '// Organization role permissions mapping\n';
  code +=
    'export const organizationRolePermissions: Record<OrganizationRole, OrganizationPermission[]> = {\n';
  orgRoles.forEach((role) => {
    code += `  ${role.name}: [\n`;
    role.permissions.forEach((perm) => {
      const constName = toConstantName(perm);
      code += `    OrganizationPermission.${constName},\n`;
    });
    code += `  ],\n`;
  });
  code += '};\n\n';

  // Generate projectRolePermissions mapping
  code += '// Project role permissions mapping\n';
  code += 'export const projectRolePermissions: Record<ProjectRole, ProjectPermission[]> = {\n';
  projRoles.forEach((role) => {
    code += `  ${role.name}: [\n`;
    role.permissions.forEach((perm) => {
      const constName = toConstantName(perm);
      code += `    ProjectPermission.${constName},\n`;
    });
    code += `  ],\n`;
  });
  code += '};\n';

  return code;
}

/**
 * Generate the complete TypeScript file
 */
function generateTypeScriptFile(permissions: string[], roles: Role[]): string {
  let code = '';

  // File header
  code += '/**\n';
  code += ' * RBAC Generated Code\n';
  code += ' *\n';
  code += ' * ⚠️  DO NOT EDIT THIS FILE MANUALLY ⚠️\n';
  code += ' *\n';
  code += ' * This file is automatically generated from:\n';
  code += ` *   - ${path.relative(process.cwd(), PERMISSIONS_YAML)}\n`;
  code += ` *   - ${path.relative(process.cwd(), ROLES_YAML)}\n`;
  code += ' *\n';
  code += ' * To update permissions or roles:\n';
  code += ' *   1. Edit the YAML files\n';
  code += ' *   2. Run: pnpm generate:rbac\n';
  code += ' *\n';
  code += ` * Generated at: ${new Date().toISOString()}\n`;
  code += ' */\n\n';

  // Generate role types
  code += generateRoleTypes(roles);

  // Generate permissions
  code += generatePermissions(permissions);

  // Generate role mappings
  code += generateRoleMappings(roles);

  return code;
}

/**
 * Main function
 */
function main() {
  console.log('🔧 Generating RBAC TypeScript code from YAML files...\n');

  // Check if YAML files exist
  if (!fs.existsSync(PERMISSIONS_YAML)) {
    console.error(`❌ Permissions file not found: ${PERMISSIONS_YAML}`);
    process.exit(1);
  }
  if (!fs.existsSync(ROLES_YAML)) {
    console.error(`❌ Roles file not found: ${ROLES_YAML}`);
    process.exit(1);
  }

  // Load YAML files
  console.log(`📖 Loading YAML files...`);
  const { permissions, roles } = loadYamlFiles();
  console.log(`   - Found ${permissions.length} permissions`);
  console.log(`   - Found ${roles.length} roles`);

  // Validate YAML structure
  console.log(`\n✅ Validating YAML structure...`);
  validateYaml(permissions, roles);
  console.log(`   - All validation checks passed`);

  // Generate TypeScript code
  console.log(`\n📝 Generating TypeScript code...`);
  const code = generateTypeScriptFile(permissions, roles);

  // Write output file
  console.log(`\n💾 Writing to ${path.relative(process.cwd(), OUTPUT_FILE)}...`);
  fs.writeFileSync(OUTPUT_FILE, code, 'utf8');

  console.log(`\n✨ RBAC code generation completed successfully!\n`);

  // Print summary
  const orgRoles = roles.filter((r) => !isProjectRole(r.name));
  const projRoles = roles.filter((r) => isProjectRole(r.name));
  const orgPerms = permissions.filter((p) => !isProjectPermission(p));
  const projPerms = permissions.filter((p) => isProjectPermission(p));

  console.log('📊 Summary:');
  console.log(`   - Organization roles: ${orgRoles.length}`);
  console.log(`   - Project roles: ${projRoles.length}`);
  console.log(`   - Organization permissions: ${orgPerms.length}`);
  console.log(`   - Project permissions: ${projPerms.length}`);
  console.log(`   - Total permissions: ${permissions.length}`);
  console.log('');
}

// Run the generator
main();
