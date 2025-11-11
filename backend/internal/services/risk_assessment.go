package services

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// RiskAssessmentService provides risk assessment functionality for change plans
type RiskAssessmentService struct {
	databaseRepo         repository.DatabaseRepository
	environmentRepo      repository.EnvironmentRepository
	databaseInstanceRepo repository.DatabaseInstanceRepository
	logger               *zap.Logger
}

// NewRiskAssessmentService creates a new risk assessment service
func NewRiskAssessmentService(
	databaseRepo repository.DatabaseRepository,
	environmentRepo repository.EnvironmentRepository,
	databaseInstanceRepo repository.DatabaseInstanceRepository,
	logger *zap.Logger,
) *RiskAssessmentService {
	return &RiskAssessmentService{
		databaseRepo:         databaseRepo,
		environmentRepo:      environmentRepo,
		databaseInstanceRepo: databaseInstanceRepo,
		logger:               logger,
	}
}

// AssessRisk calculates the risk level for a change plan based on:
// - SQL content (destructive operations)
// - Environment (Production = higher risk)
// - Database type (for future extensions)
func (s *RiskAssessmentService) AssessRisk(
	ctx context.Context,
	sqlContent string,
	changeType models.ChangeType,
	targetDatabaseID uuid.UUID,
) (models.RiskLevel, error) {
	riskScore := 0

	// Get database to access environment and database instance
	database, err := s.databaseRepo.GetByID(ctx, targetDatabaseID)
	if err != nil {
		s.logger.Error("Failed to get database for risk assessment", zap.Error(err))
		return models.RiskLevelMedium, err
	}

	// Analyze SQL content for destructive operations
	sqlRiskScore := s.analyzeSQLContent(sqlContent, changeType)
	riskScore += sqlRiskScore

	// Analyze environment (Production = higher risk)
	environment, err := s.environmentRepo.GetByID(ctx, database.EnvironmentID)
	if err != nil {
		s.logger.Error("Failed to get environment for risk assessment", zap.Error(err))
		return models.RiskLevelMedium, err
	}
	envRiskScore := s.analyzeEnvironment(environment.Name)
	riskScore += envRiskScore

	// Analyze database type (for future extensions)
	instance, err := s.databaseInstanceRepo.GetByID(ctx, database.DatabaseInstanceID)
	if err != nil {
		s.logger.Error("Failed to get database instance for risk assessment", zap.Error(err))
		return models.RiskLevelMedium, err
	}
	dbTypeRiskScore := s.analyzeDatabaseEngine(instance.DatabaseEngine)
	riskScore += dbTypeRiskScore

	// Calculate final risk level
	riskLevel := s.calculateRiskLevel(riskScore)

	s.logger.Debug("Risk assessment completed",
		zap.String("target_database_id", targetDatabaseID.String()),
		zap.Int("risk_score", riskScore),
		zap.String("risk_level", string(riskLevel)),
		zap.Int("sql_risk_score", sqlRiskScore),
		zap.Int("env_risk_score", envRiskScore),
		zap.Int("db_type_risk_score", dbTypeRiskScore),
	)

	return riskLevel, nil
}

// analyzeSQLContent analyzes SQL content for destructive operations and returns a risk score
func (s *RiskAssessmentService) analyzeSQLContent(sqlContent string, changeType models.ChangeType) int {
	riskScore := 0
	sqlUpper := strings.ToUpper(strings.TrimSpace(sqlContent))

	// DDL operations are generally higher risk
	if changeType == models.ChangeTypeDDL {
		riskScore += 2
	}

	// Check for highly destructive operations (High risk)
	highRiskKeywords := []string{
		"DROP DATABASE",
		"DROP SCHEMA",
		"TRUNCATE TABLE",
		"DROP TABLE",
		"DROP COLUMN",
		"DROP INDEX",
		"DROP CONSTRAINT",
		"ALTER TABLE ... DROP",
		"DELETE FROM", // Without WHERE clause or with dangerous WHERE
	}

	for _, keyword := range highRiskKeywords {
		if strings.Contains(sqlUpper, keyword) {
			// Check if DELETE has WHERE clause
			if keyword == "DELETE FROM" {
				if !strings.Contains(sqlUpper, "WHERE") {
					riskScore += 5 // DELETE without WHERE is very dangerous
				} else {
					riskScore += 3 // DELETE with WHERE is still risky
				}
			} else {
				riskScore += 5
			}
			break // Only count the first high-risk operation found
		}
	}

	// Check for moderate risk operations
	moderateRiskKeywords := []string{
		"ALTER TABLE",
		"CREATE INDEX",
		"DROP INDEX",
		"UPDATE",
		"INSERT INTO",
	}

	if riskScore < 5 { // Only check moderate risks if no high-risk found
		for _, keyword := range moderateRiskKeywords {
			if strings.Contains(sqlUpper, keyword) {
				if keyword == "UPDATE" {
					if !strings.Contains(sqlUpper, "WHERE") {
						riskScore += 3 // UPDATE without WHERE is dangerous
					} else {
						riskScore += 1
					}
				} else {
					riskScore += 1
				}
				break
			}
		}
	}

	return riskScore
}

// analyzeEnvironment analyzes the environment name and returns a risk score
func (s *RiskAssessmentService) analyzeEnvironment(envName string) int {
	envNameUpper := strings.ToUpper(strings.TrimSpace(envName))

	// Production environment is high risk
	if strings.Contains(envNameUpper, "PROD") || strings.Contains(envNameUpper, "PRODUCTION") {
		return 3
	}

	// Staging environment is moderate risk
	if strings.Contains(envNameUpper, "STAGING") || strings.Contains(envNameUpper, "STAGE") {
		return 1
	}

	// Development/Test environments are low risk
	if strings.Contains(envNameUpper, "DEV") || strings.Contains(envNameUpper, "DEVELOPMENT") ||
		strings.Contains(envNameUpper, "TEST") || strings.Contains(envNameUpper, "LOCAL") {
		return 0
	}

	// Default to medium risk for unknown environments
	return 1
}

// analyzeDatabaseEngine analyzes the database type and returns a risk score
// This can be extended in the future for database-specific risk factors
func (s *RiskAssessmentService) analyzeDatabaseEngine(dbType models.DatabaseEngine) int {
	// For now, all database types have the same base risk
	// This can be extended in the future to differentiate between MySQL and PostgreSQL
	// based on specific features or known issues
	return 0
}

// calculateRiskLevel converts a risk score to a RiskLevel
// Risk score ranges:
// - 0-2: Low
// - 3-5: Medium
// - 6+: High
func (s *RiskAssessmentService) calculateRiskLevel(riskScore int) models.RiskLevel {
	if riskScore <= 2 {
		return models.RiskLevelLow
	} else if riskScore <= 5 {
		return models.RiskLevelMedium
	}
	return models.RiskLevelHigh
}
