package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// ProjectAssignmentService provides functionality for automatically assigning users to projects
type ProjectAssignmentService struct {
	projectRepo       repository.ProjectRepository
	projectMemberRepo repository.ProjectMemberRepository
	userRepo          repository.UserRepository
	enabled           bool // Feature flag: enable/disable auto-assignment
	logger            *zap.Logger
}

// NewProjectAssignmentService creates a new project assignment service
func NewProjectAssignmentService(
	projectRepo repository.ProjectRepository,
	projectMemberRepo repository.ProjectMemberRepository,
	userRepo repository.UserRepository,
	enabled bool,
	logger *zap.Logger,
) *ProjectAssignmentService {
	return &ProjectAssignmentService{
		projectRepo:       projectRepo,
		projectMemberRepo: projectMemberRepo,
		userRepo:          userRepo,
		enabled:           enabled,
		logger:            logger,
	}
}

// Service errors
var (
	ErrSystemUserNotFound = fmt.Errorf("system user not found")
)

const (
	systemUserEmail     = "system@deeb1.dev"
	maxProjectsToAssign = 1000 // Maximum number of projects to auto-assign (performance limit)
)

// AssignUserToExistingProjects assigns a new user to all existing projects
// with ProjectDeveloper role, using the system user as the added_by_id.
// Returns the count of projects assigned or error.
// If no projects exist, returns (0, nil) without error.
func (s *ProjectAssignmentService) AssignUserToExistingProjects(ctx context.Context, userID uuid.UUID) (int, error) {
	// Check if feature is enabled
	if !s.enabled {
		s.logger.Debug("Auto-assignment feature is disabled",
			zap.String("user_id", userID.String()))
		return 0, nil
	}

	// Get system user by email
	systemUser, err := s.userRepo.GetByEmail(ctx, systemUserEmail)
	if err != nil {
		if err == repository.ErrNotFound {
			s.logger.Error("System user not found - run migration 000032_create_system_user.up.sql",
				zap.String("email", systemUserEmail))
			return 0, ErrSystemUserNotFound
		}
		return 0, fmt.Errorf("failed to get system user: %w", err)
	}

	// Validate system user exists (nil check is sufficient - UUID can be all zeros)
	if systemUser == nil {
		s.logger.Error("System user is nil - run migration 000032_create_system_user.up.sql",
			zap.String("email", systemUserEmail))
		return 0, ErrSystemUserNotFound
	}

	// Fetch projects (with reasonable limit for performance)
	projects, err := s.projectRepo.List(ctx, maxProjectsToAssign, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to list projects: %w", err)
	}

	// If no projects exist, return early
	if len(projects) == 0 {
		s.logger.Debug("No projects exist, skipping auto-assignment",
			zap.String("user_id", userID.String()))
		return 0, nil
	}

	// Check if we hit the limit (there might be more projects)
	if len(projects) == maxProjectsToAssign {
		s.logger.Warn("Reached maximum project limit for auto-assignment",
			zap.String("user_id", userID.String()),
			zap.Int("limit", maxProjectsToAssign),
			zap.Int("assigned", len(projects)))
	}

	// Prepare batch of members to add (use transaction for atomic assignment)
	members := make([]*models.ProjectMember, len(projects))
	for i, project := range projects {
		members[i] = &models.ProjectMember{
			ID:        uuid.New(),
			ProjectID: project.ID,
			UserID:    userID,
			Role:      rbac.ProjectDeveloper,
			AddedByID: systemUser.ID,
			AddedAt:   time.Now(),
		}
	}

	// Add all members in a single transaction (atomic - all or nothing)
	if err := s.projectMemberRepo.AddMembersBatch(ctx, members); err != nil {
		// Check if it's a unique constraint violation
		if isUniqueConstraintError(err) {
			s.logger.Warn("User already member of one or more projects - partial assignment may have occurred",
				zap.Error(err),
				zap.String("user_id", userID.String()),
				zap.Int("total_projects", len(projects)))
			return 0, fmt.Errorf("user already member of project(s): %w", err)
		}

		s.logger.Error("Failed to assign user to projects",
			zap.Error(err),
			zap.String("user_id", userID.String()),
			zap.Int("total_projects", len(projects)))
		return 0, fmt.Errorf("failed to assign user to projects: %w", err)
	}

	s.logger.Info("Auto-assigned user to projects",
		zap.String("user_id", userID.String()),
		zap.Int("project_count", len(projects)))

	return len(projects), nil
}

// isUniqueConstraintError checks if the error is a PostgreSQL unique constraint violation
func isUniqueConstraintError(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		// 23505 is the PostgreSQL error code for unique_violation
		if pqErr.Code == "23505" {
			// Check if the constraint name matches our unique constraint
			if strings.Contains(pqErr.Constraint, "unique_project_user") {
				return true
			}
		}
	}
	return false
}
