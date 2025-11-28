package handlers

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockProjectAssignmentService is a mock implementation of ProjectAssignmentService
type MockProjectAssignmentService struct {
	mock.Mock
}

func (m *MockProjectAssignmentService) AssignUserToExistingProjects(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}
