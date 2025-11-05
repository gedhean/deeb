package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/workflow"
	"go.uber.org/zap"
)

// ApprovalService provides approval workflow functionality for change plans
type ApprovalService struct {
	approvalRepo      repository.ApprovalRepository
	changePlanRepo    repository.ChangePlanRepository
	policyFindingRepo repository.PolicyFindingRepository
	projectMemberRepo repository.ProjectMemberRepository
	rbacChecker       *rbac.CheckerService
	logger            *zap.Logger
}

// NewApprovalService creates a new approval service
func NewApprovalService(
	approvalRepo repository.ApprovalRepository,
	changePlanRepo repository.ChangePlanRepository,
	policyFindingRepo repository.PolicyFindingRepository,
	projectMemberRepo repository.ProjectMemberRepository,
	rbacChecker *rbac.CheckerService,
	logger *zap.Logger,
) *ApprovalService {
	return &ApprovalService{
		approvalRepo:      approvalRepo,
		changePlanRepo:    changePlanRepo,
		policyFindingRepo: policyFindingRepo,
		projectMemberRepo: projectMemberRepo,
		rbacChecker:       rbacChecker,
		logger:            logger,
	}
}

// Approval errors
var (
	ErrChangePlanNotFound        = fmt.Errorf("change plan not found")
	ErrInvalidChangePlanStatus   = fmt.Errorf("change plan is not in Open or Failed status")
	ErrUnauthorizedApprover      = fmt.Errorf("user does not have permission to approve change plans")
	ErrCannotApproveWithErrors   = fmt.Errorf("cannot approve change plan with SQL Review errors")
	ErrChangePlanAlreadyApproved = fmt.Errorf("change plan is already approved")
	ErrChangePlanAlreadyClosed   = fmt.Errorf("change plan is already closed")
)

// Approve approves a Change Plan, moving it from Open or Failed to Approved status
// Requires: no SQL Review errors, valid approver permissions
// Allows retry of Failed change plans by transitioning them back to Approved
func (s *ApprovalService) Approve(ctx context.Context, changePlanID uuid.UUID, approverID uuid.UUID, comments *string) error {
	// Get Change Plan
	changePlan, err := s.changePlanRepo.GetByID(ctx, changePlanID)
	if err != nil {
		if err == repository.ErrNotFound {
			return ErrChangePlanNotFound
		}
		return fmt.Errorf("failed to get change plan: %w", err)
	}

	// Verify Change Plan can transition to Approved status
	// Allow transitions from Open or Failed status (Failed can be retried)
	if changePlan.Status != models.ChangePlanStatusOpen && changePlan.Status != models.ChangePlanStatusFailed {
		if changePlan.Status == models.ChangePlanStatusApproved {
			return ErrChangePlanAlreadyApproved
		}
		if changePlan.Status == models.ChangePlanStatusClosed {
			return ErrChangePlanAlreadyClosed
		}
		return fmt.Errorf("%w: current status is %s", ErrInvalidChangePlanStatus, changePlan.Status)
	}

	// Validate the transition using workflow validation
	if err := workflow.ValidateTransition(changePlan.Status, models.ChangePlanStatusApproved); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidChangePlanStatus, err)
	}

	// Verify approver has permission
	if err := s.verifyApproverPermission(ctx, changePlan.ProjectID, approverID); err != nil {
		return fmt.Errorf("%w: %v", ErrUnauthorizedApprover, err)
	}

	// Check for SQL Review errors
	hasErrors, errorFindings, err := s.hasSQLReviewErrors(ctx, changePlanID)
	if err != nil {
		return fmt.Errorf("failed to check SQL Review errors: %w", err)
	}

	if hasErrors {
		// Build error message with details
		errorMessages := make([]string, 0, len(errorFindings))
		for _, finding := range errorFindings {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", finding.RuleName, finding.Message))
		}
		return fmt.Errorf("%w: %s", ErrCannotApproveWithErrors, strings.Join(errorMessages, "; "))
	}

	// Check if approval already exists for this user and change plan
	existingApproval, err := s.approvalRepo.GetByChangePlanIDAndApproverID(ctx, changePlanID, approverID)
	if err != nil && err != repository.ErrNotFound {
		return fmt.Errorf("failed to check existing approval: %w", err)
	}

	now := time.Now()

	if existingApproval != nil {
		// Update existing approval
		existingApproval.Status = models.ApprovalStatusApproved
		existingApproval.Comments = comments
		existingApproval.ApprovedAt = &now

		if err := s.approvalRepo.Update(ctx, existingApproval); err != nil {
			return fmt.Errorf("failed to update approval record: %w", err)
		}

		s.logger.Info("Approval record updated",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("approver_id", approverID.String()),
			zap.String("approval_id", existingApproval.ID.String()),
		)
	} else {
		// Create new approval record
		approval := &models.Approval{
			ID:           uuid.New(),
			ChangePlanID: changePlanID,
			ApproverID:   approverID,
			Status:       models.ApprovalStatusApproved,
			Comments:     comments,
			ApprovedAt:   &now,
			CreatedAt:    now,
		}

		if err := s.approvalRepo.Create(ctx, approval); err != nil {
			return fmt.Errorf("failed to create approval record: %w", err)
		}

		s.logger.Info("Approval record created",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("approver_id", approverID.String()),
			zap.String("approval_id", approval.ID.String()),
		)
	}

	// Update Change Plan status to Approved
	changePlan.Status = models.ChangePlanStatusApproved
	changePlan.UpdatedAt = time.Now()

	if err := s.changePlanRepo.Update(ctx, changePlan); err != nil {
		return fmt.Errorf("failed to update change plan status: %w", err)
	}

	s.logger.Info("Change plan approved",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("approver_id", approverID.String()),
		zap.String("project_id", changePlan.ProjectID.String()),
	)

	return nil
}

// Close rejects/closes a Change Plan, moving it from Open to Closed status
// Requires: valid approver permissions
func (s *ApprovalService) Close(ctx context.Context, changePlanID uuid.UUID, approverID uuid.UUID, comments *string) error {
	// Get Change Plan
	changePlan, err := s.changePlanRepo.GetByID(ctx, changePlanID)
	if err != nil {
		if err == repository.ErrNotFound {
			return ErrChangePlanNotFound
		}
		return fmt.Errorf("failed to get change plan: %w", err)
	}

	// Verify Change Plan is in Open or Failed status
	if changePlan.Status != models.ChangePlanStatusOpen && changePlan.Status != models.ChangePlanStatusFailed {
		if changePlan.Status == models.ChangePlanStatusApproved {
			return ErrChangePlanAlreadyApproved
		}
		if changePlan.Status == models.ChangePlanStatusClosed {
			return ErrChangePlanAlreadyClosed
		}
		return fmt.Errorf("%w: current status is %s", ErrInvalidChangePlanStatus, changePlan.Status)
	}

	// Verify approver has permission
	if err := s.verifyApproverPermission(ctx, changePlan.ProjectID, approverID); err != nil {
		return fmt.Errorf("%w: %v", ErrUnauthorizedApprover, err)
	}

	// Create approval record with Rejected status
	approval := &models.Approval{
		ID:           uuid.New(),
		ChangePlanID: changePlanID,
		ApproverID:   approverID,
		Status:       models.ApprovalStatusRejected,
		Comments:     comments,
		CreatedAt:    time.Now(),
		// approved_at is nil for rejected approvals
	}

	if err := s.approvalRepo.Create(ctx, approval); err != nil {
		return fmt.Errorf("failed to create approval record: %w", err)
	}

	// Update Change Plan status to Closed
	changePlan.Status = models.ChangePlanStatusClosed
	changePlan.UpdatedAt = time.Now()

	if err := s.changePlanRepo.Update(ctx, changePlan); err != nil {
		return fmt.Errorf("failed to update change plan status: %w", err)
	}

	s.logger.Info("Change plan closed",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("approver_id", approverID.String()),
		zap.String("project_id", changePlan.ProjectID.String()),
	)

	return nil
}

// GetApprovalHistory retrieves approval history for a Change Plan
// Returns approvals ordered by created_at DESC (most recent first)
func (s *ApprovalService) GetApprovalHistory(ctx context.Context, changePlanID uuid.UUID) ([]*models.Approval, error) {
	approvals, err := s.approvalRepo.GetHistory(ctx, changePlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get approval history: %w", err)
	}

	return approvals, nil
}

// verifyApproverPermission checks if the user has permission to approve/reject change plans
// Returns nil if permission is granted, error otherwise
func (s *ApprovalService) verifyApproverPermission(ctx context.Context, projectID uuid.UUID, approverID uuid.UUID) error {
	// Get user from context (organization role)
	userIDStr, _, orgRole, ok := rbac.GetUserFromContext(ctx)
	if !ok {
		return fmt.Errorf("user information not found in context")
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fmt.Errorf("invalid user ID in context: %w", err)
	}

	// Verify the approver ID matches the user in context
	if userUUID != approverID {
		return fmt.Errorf("approver ID does not match authenticated user")
	}

	// Check organization-level permissions (Org Admin, Org DBA can approve)
	if s.rbacChecker.HasOrganizationPermission(orgRole, rbac.PermissionChangesApprove) {
		return nil
	}

	// Check project-level permissions
	projectMember, err := s.projectMemberRepo.GetMemberRole(ctx, projectID, approverID)
	if err != nil {
		if err == repository.ErrNotFound {
			return fmt.Errorf("user is not a project member")
		}
		return fmt.Errorf("failed to get project member role: %w", err)
	}

	// Convert models.ProjectRole to rbac.ProjectRole
	projectRole := s.toRBACProjectRole(projectMember.Role)
	if projectRole == "" {
		return fmt.Errorf("invalid project role: %s", projectMember.Role)
	}

	// Check if user has approve permission at project level
	if !s.rbacChecker.HasProjectPermission(projectRole, rbac.PermissionChangesApprove) {
		return fmt.Errorf("user does not have approve permission in project")
	}

	return nil
}

// hasSQLReviewErrors checks if the Change Plan has any SQL Review errors
// Returns: (hasErrors, errorFindings, error)
func (s *ApprovalService) hasSQLReviewErrors(ctx context.Context, changePlanID uuid.UUID) (bool, []*models.PolicyFinding, error) {
	findings, err := s.policyFindingRepo.GetByChangePlanID(ctx, changePlanID)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get policy findings: %w", err)
	}

	// Filter for errors only
	errorFindings := make([]*models.PolicyFinding, 0)
	for _, finding := range findings {
		if finding.Severity == "Error" {
			errorFindings = append(errorFindings, finding)
		}
	}

	return len(errorFindings) > 0, errorFindings, nil
}

// toRBACProjectRole converts models.ProjectRole to rbac.ProjectRole
func (s *ApprovalService) toRBACProjectRole(role models.ProjectRole) rbac.ProjectRole {
	switch role {
	case models.ProjectRoleOwner:
		return rbac.ProjectOwner
	case models.ProjectRoleDeveloper:
		return rbac.ProjectDeveloper
	case models.ProjectRoleApprover:
		return rbac.ProjectApprover
	case models.ProjectRoleViewer:
		return rbac.ProjectViewer
	default:
		return ""
	}
}
