package handlers

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubEnvironmentRepo is a minimal EnvironmentRepository for validation tests.
type stubEnvironmentRepo struct {
	env *models.Environment
	err error
}

func (s *stubEnvironmentRepo) Create(context.Context, *models.Environment) error { return nil }
func (s *stubEnvironmentRepo) GetByID(context.Context, uuid.UUID) (*models.Environment, error) {
	return s.env, s.err
}
func (s *stubEnvironmentRepo) List(context.Context, repository.EnvironmentFilters, int, int) ([]*models.Environment, error) {
	return nil, nil
}
func (s *stubEnvironmentRepo) Count(context.Context, repository.EnvironmentFilters) (int64, error) {
	return 0, nil
}
func (s *stubEnvironmentRepo) Update(context.Context, *models.Environment) error { return nil }
func (s *stubEnvironmentRepo) Delete(context.Context, uuid.UUID) error           { return nil }

func TestValidateGhostSelection(t *testing.T) {
	envID := uuid.New()
	dbID := uuid.New()

	nonProdEnv := &models.Environment{ID: envID, Name: "Staging", IsProduction: false}
	prodEnv := &models.Environment{ID: envID, Name: "Production", IsProduction: true}

	tests := []struct {
		name        string
		backend     models.ExecutionBackend
		changeType  models.ChangeType
		envID       *uuid.UUID
		envRepo     repository.EnvironmentRepository
		wantBackend models.ExecutionBackend
		wantErr     string
	}{
		{
			name:        "empty backend defaults to direct",
			backend:     "",
			changeType:  models.ChangeTypeDDL,
			envID:       &envID,
			envRepo:     &stubEnvironmentRepo{env: nonProdEnv},
			wantBackend: models.ExecutionBackendDirect,
		},
		{
			name:        "explicit direct passes",
			backend:     models.ExecutionBackendDirect,
			changeType:  models.ChangeTypeDDL,
			envID:       &envID,
			envRepo:     &stubEnvironmentRepo{env: nonProdEnv},
			wantBackend: models.ExecutionBackendDirect,
		},
		{
			name:        "ghost allowed on DDL + non-production",
			backend:     models.ExecutionBackendGhost,
			changeType:  models.ChangeTypeDDL,
			envID:       &envID,
			envRepo:     &stubEnvironmentRepo{env: nonProdEnv},
			wantBackend: models.ExecutionBackendGhost,
		},
		{
			name:       "ghost rejected for DML",
			backend:    models.ExecutionBackendGhost,
			changeType: models.ChangeTypeDML,
			envID:      &envID,
			envRepo:    &stubEnvironmentRepo{env: nonProdEnv},
			wantErr:    "only supported for DDL",
		},
		{
			name:       "ghost rejected on production environment",
			backend:    models.ExecutionBackendGhost,
			changeType: models.ChangeTypeDDL,
			envID:      &envID,
			envRepo:    &stubEnvironmentRepo{env: prodEnv},
			wantErr:    "not permitted on production environments",
		},
		{
			name:       "ghost rejected when database has no environment",
			backend:    models.ExecutionBackendGhost,
			changeType: models.ChangeTypeDDL,
			envID:      nil,
			envRepo:    &stubEnvironmentRepo{env: nonProdEnv},
			wantErr:    "assigned to an environment",
		},
		{
			name:       "ghost rejected when environment lookup fails",
			backend:    models.ExecutionBackendGhost,
			changeType: models.ChangeTypeDDL,
			envID:      &envID,
			envRepo:    &stubEnvironmentRepo{err: repository.ErrNotFound},
			wantErr:    "failed to validate environment",
		},
		{
			name:       "invalid backend value rejected",
			backend:    models.ExecutionBackend("turbo"),
			changeType: models.ChangeTypeDDL,
			envID:      &envID,
			envRepo:    &stubEnvironmentRepo{env: nonProdEnv},
			wantErr:    "invalid execution_backend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ChangePlanHandler{environmentRepo: tt.envRepo}
			db := &models.Database{ID: dbID, EnvironmentID: tt.envID}

			got, err := h.validateGhostSelection(context.Background(), tt.backend, tt.changeType, db)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantBackend, got)
		})
	}
}
