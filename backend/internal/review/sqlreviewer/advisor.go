package sqlreviewer

import (
	"context"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/sql-reviewer/pkg/advisor"
	"github.com/nsxbet/sql-reviewer/pkg/types"
	"go.uber.org/zap"
)

// Advisor wraps the sql-reviewer Advisor interface
type Advisor struct {
	logger *zap.Logger
}

// NewAdvisor creates a new Advisor instance
func NewAdvisor(logger *zap.Logger) *Advisor {
	return &Advisor{
		logger: logger,
	}
}

// Check runs SQL review using sql-reviewer for all provided rules
func (a *Advisor) Check(
	ctx context.Context,
	sqlContent string,
	dbType models.DatabaseType,
	rules []*types.SQLReviewRule,
) ([]*types.Advice, error) {
	// Convert database type to sql-reviewer engine type
	engine := convertDatabaseTypeToEngine(dbType)

	allAdvice := make([]*types.Advice, 0)

	// Run each rule check individually
	for _, rule := range rules {
		// Create advisor context for this rule
		checkCtx := advisor.Context{
			DBType:     engine,
			Statements: sqlContent,
			Rule:       rule,
		}

		// Determine advisor type from rule type
		advType := advisor.Type(rule.Type)

		// Run the advisor check for this rule
		adviceList, err := advisor.Check(ctx, engine, advType, checkCtx)
		if err != nil {
			a.logger.Warn("Failed to run sql-reviewer check for rule",
				zap.String("rule", rule.Type),
				zap.Error(err))
			continue
		}

		allAdvice = append(allAdvice, adviceList...)
	}

	return allAdvice, nil
}

// convertDatabaseTypeToEngine converts models.DatabaseType to sql-reviewer engine type
func convertDatabaseTypeToEngine(dbType models.DatabaseType) types.Engine {
	switch dbType {
	case models.DatabaseTypeMySQL:
		return types.Engine_MYSQL
	case models.DatabaseTypePostgreSQL:
		return types.Engine_POSTGRES
	default:
		return types.Engine_MYSQL // Default to MySQL
	}
}
