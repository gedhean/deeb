package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

// PermissionsYAML represents the structure of permissions.yaml
type PermissionsYAML struct {
	Permissions []string `yaml:"permissions"`
}

// RoleYAML represents a single role in roles.yaml
type RoleYAML struct {
	Name        string   `yaml:"name"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Permissions []string `yaml:"permissions"`
}

// RolesYAML represents the structure of roles.yaml
type RolesYAML struct {
	Roles []RoleYAML `yaml:"roles"`
}

// PermissionInfo holds information about a permission for code generation
type PermissionInfo struct {
	ConstName  string // e.g., "PermissionUsersCreate"
	Value      string // e.g., "users.create"
	IsWildcard bool   // true if permission is "*"
	IsOrgLevel bool   // true if organization-level permission
}

// RoleInfo holds information about a role for code generation
type RoleInfo struct {
	Name        string   // e.g., "OrganizationAdmin"
	ConstValue  string   // e.g., "OrganizationAdmin" (PascalCase)
	Description string   // e.g., "Full system access"
	Permissions []string // Permission constant names
	IsOrgRole   bool     // true if organization role
}

func main() {
	var (
		permissionsFile = flag.String("permissions", "backend/config/iam/permissions.yaml", "Path to permissions.yaml")
		rolesFile       = flag.String("roles", "backend/config/iam/roles.yaml", "Path to roles.yaml")
		outputDir       = flag.String("output", "backend/internal/rbac", "Output directory for generated files")
	)
	flag.Parse()

	// Read and parse YAML files
	permissions, err := readPermissions(*permissionsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading permissions: %v\n", err)
		os.Exit(1)
	}

	roles, err := readRoles(*rolesFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading roles: %v\n", err)
		os.Exit(1)
	}

	// Validate that all permissions in roles exist in permissions.yaml
	if err := validateRolePermissions(permissions, roles); err != nil {
		fmt.Fprintf(os.Stderr, "Validation error: %v\n", err)
		os.Exit(1)
	}

	// Generate permission info
	permInfos := generatePermissionInfos(permissions)

	// Generate role info
	orgRoles, projectRoles := generateRoleInfos(roles, permInfos)

	// Generate files
	if err := generatePermissionsFile(*outputDir, permInfos); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating permissions file: %v\n", err)
		os.Exit(1)
	}

	if err := generateRolesFile(*outputDir, orgRoles, projectRoles); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating roles file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Generated RBAC code successfully")
}

func readPermissions(path string) (*PermissionsYAML, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var perms PermissionsYAML
	if err := yaml.Unmarshal(data, &perms); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	return &perms, nil
}

func readRoles(path string) (*RolesYAML, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var roles RolesYAML
	if err := yaml.Unmarshal(data, &roles); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	return &roles, nil
}

func validateRolePermissions(perms *PermissionsYAML, roles *RolesYAML) error {
	permSet := make(map[string]bool)
	for _, p := range perms.Permissions {
		permSet[p] = true
	}

	for _, role := range roles.Roles {
		for _, perm := range role.Permissions {
			if !permSet[perm] {
				return fmt.Errorf("role %s references unknown permission: %s", role.Name, perm)
			}
		}
	}

	return nil
}

func generatePermissionInfos(perms *PermissionsYAML) []PermissionInfo {
	var infos []PermissionInfo

	for _, perm := range perms.Permissions {
		info := PermissionInfo{
			Value:      perm,
			IsWildcard: perm == "*",
		}

		if perm == "*" {
			info.ConstName = "PermissionAll"
		} else {
			info.ConstName = permissionToConstName(perm)
		}

		// Determine if it's organization-level or project-level
		// Project-level permissions start with "project.", "changes.", or "policy_rules."
		info.IsOrgLevel = !strings.HasPrefix(perm, "project.") &&
			!strings.HasPrefix(perm, "changes.") &&
			!strings.HasPrefix(perm, "policy_rules.") &&
			perm != "*"

		infos = append(infos, info)
	}

	return infos
}

func permissionToConstName(perm string) string {
	// Convert "users.create" -> "PermissionUsersCreate"
	// Convert "project.members.create" -> "PermissionProjectMembersCreate"
	// Convert "policy_rules.create" -> "PermissionPolicyRulesCreate"

	parts := strings.Split(perm, ".")
	var builder strings.Builder
	builder.WriteString("Permission")

	for _, part := range parts {
		// Handle underscores: "policy_rules" -> "PolicyRules"
		subparts := strings.Split(part, "_")
		for _, subpart := range subparts {
			// Capitalize first letter manually to avoid deprecated strings.Title
			if len(subpart) > 0 {
				builder.WriteString(strings.ToUpper(subpart[:1]) + subpart[1:])
			}
		}
	}

	return builder.String()
}

func generateRoleInfos(roles *RolesYAML, permInfos []PermissionInfo) ([]RoleInfo, []RoleInfo) {
	// Create permission value -> const name map
	permMap := make(map[string]string)
	for _, info := range permInfos {
		permMap[info.Value] = info.ConstName
	}

	var orgRoles, projectRoles []RoleInfo

	for _, role := range roles.Roles {
		info := RoleInfo{
			Name:        role.Name,
			ConstValue:  role.Name, // Use PascalCase directly from YAML
			Description: role.Description,
		}

		// Convert permission values to constant names
		for _, perm := range role.Permissions {
			constName, ok := permMap[perm]
			if !ok {
				fmt.Fprintf(os.Stderr, "Warning: unknown permission %s in role %s\n", perm, role.Name)
				continue
			}
			info.Permissions = append(info.Permissions, constName)
		}

		// Determine if organization or project role
		if strings.HasPrefix(role.Name, "Organization") {
			info.IsOrgRole = true
			orgRoles = append(orgRoles, info)
		} else if strings.HasPrefix(role.Name, "Project") {
			info.IsOrgRole = false
			projectRoles = append(projectRoles, info)
		}
	}

	return orgRoles, projectRoles
}

func generatePermissionsFile(outputDir string, permInfos []PermissionInfo) error {
	tmpl := `// Code generated by cmd/generate-rbac. DO NOT EDIT.
// Generated from: backend/config/iam/permissions.yaml
// Generated at: {{ .Timestamp }}

package rbac

// Permission represents an action that can be performed on a resource.
// Permissions are granted through organization and project roles.
type Permission string

// Organization-level permissions
const (
{{- range .OrgPermissions }}
	{{ .ConstName }} Permission = "{{ .Value }}"
{{- end }}
)

// Project-level permissions
const (
{{- range .ProjectPermissions }}
	{{ .ConstName }} Permission = "{{ .Value }}"
{{- end }}
)
`

	// Separate org and project permissions
	var orgPerms, projPerms []PermissionInfo
	for _, info := range permInfos {
		if info.IsWildcard {
			orgPerms = append(orgPerms, info)
		} else if info.IsOrgLevel {
			orgPerms = append(orgPerms, info)
		} else {
			projPerms = append(projPerms, info)
		}
	}

	// Sort for consistent output
	sort.Slice(orgPerms, func(i, j int) bool {
		// Wildcard first
		if orgPerms[i].IsWildcard {
			return true
		}
		if orgPerms[j].IsWildcard {
			return false
		}
		return orgPerms[i].ConstName < orgPerms[j].ConstName
	})
	sort.Slice(projPerms, func(i, j int) bool {
		return projPerms[i].ConstName < projPerms[j].ConstName
	})

	data := struct {
		Timestamp          string
		OrgPermissions     []PermissionInfo
		ProjectPermissions []PermissionInfo
	}{
		Timestamp:          time.Now().Format(time.RFC3339),
		OrgPermissions:     orgPerms,
		ProjectPermissions: projPerms,
	}

	return writeFormattedGoFile(filepath.Join(outputDir, "permissions_generated.go"), tmpl, data)
}

func generateRolesFile(outputDir string, orgRoles, projectRoles []RoleInfo) error {
	tmpl := `// Code generated by cmd/generate-rbac. DO NOT EDIT.
// Generated from: backend/config/iam/roles.yaml
// Generated at: {{ .Timestamp }}

package rbac

// OrganizationRole represents a user's role at the organization level.
type OrganizationRole string

const (
{{- range .OrgRoles }}
	// {{ .Name }} - {{ .Description }}
	{{ .Name }} OrganizationRole = "{{ .ConstValue }}"
{{- end }}
)

// ProjectRole represents a user's role within a specific project.
type ProjectRole string

const (
{{- range .ProjectRoles }}
	// {{ .Name }} - {{ .Description }}
	{{ .Name }} ProjectRole = "{{ .ConstValue }}"
{{- end }}
)

// organizationRolePermissions maps organization roles to their permissions
var organizationRolePermissions = map[OrganizationRole][]Permission{
{{- range .OrgRoles }}
	{{ .Name }}: {
	{{- range .Permissions }}
		{{ . }},
	{{- end }}
	},
{{- end }}
}

// projectRolePermissions maps project roles to their permissions
var projectRolePermissions = map[ProjectRole][]Permission{
{{- range .ProjectRoles }}
	{{ .Name }}: {
	{{- range .Permissions }}
		{{ . }},
	{{- end }}
	},
{{- end }}
}

// GetOrganizationPermissions returns all permissions for a given organization role.
// Returns an empty slice if the role is not recognized.
func GetOrganizationPermissions(role OrganizationRole) []Permission {
	permissions, exists := organizationRolePermissions[role]
	if !exists {
		return []Permission{}
	}
	return permissions
}

// GetProjectPermissions returns all permissions for a given project role.
// Returns an empty slice if the role is not recognized.
func GetProjectPermissions(role ProjectRole) []Permission {
	permissions, exists := projectRolePermissions[role]
	if !exists {
		return []Permission{}
	}
	return permissions
}

// IsValidOrganizationRole checks if the given role is a valid organization role.
func IsValidOrganizationRole(role OrganizationRole) bool {
	_, exists := organizationRolePermissions[role]
	return exists
}

// IsValidProjectRole checks if the given role is a valid project role.
func IsValidProjectRole(role ProjectRole) bool {
	_, exists := projectRolePermissions[role]
	return exists
}

// hasPermission checks if a permission is in the given list
func hasPermission(permissions []Permission, permission Permission) bool {
	// Check for wildcard permission (admin)
	for _, p := range permissions {
		if p == PermissionAll {
			return true
		}
		if p == permission {
			return true
		}
	}
	return false
}
`

	data := struct {
		Timestamp    string
		OrgRoles     []RoleInfo
		ProjectRoles []RoleInfo
	}{
		Timestamp:    time.Now().Format(time.RFC3339),
		OrgRoles:     orgRoles,
		ProjectRoles: projectRoles,
	}

	return writeFormattedGoFile(filepath.Join(outputDir, "roles_generated.go"), tmpl, data)
}

func writeFormattedGoFile(path, tmplStr string, data interface{}) error {
	// Parse and execute template
	t, err := template.New("").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Write unformatted code for debugging
		if writeErr := os.WriteFile(path+".unformatted", buf.Bytes(), 0o644); writeErr != nil {
			return fmt.Errorf("formatting code: %w (also failed to write unformatted code: %v)", err, writeErr)
		}
		return fmt.Errorf("formatting code: %w (unformatted code written to %s.unformatted)", err, path)
	}

	// Write to file
	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}
