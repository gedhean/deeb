package services

import (
	"testing"

	"github.com/nsxbet/deeb/internal/database"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewSchemaMetadataService(t *testing.T) {
	dbInstanceRepo := new(MockDatabaseInstanceRepository)
	databaseRepo := new(MockDatabaseRepository)
	connectionMgr := database.NewConnectionManager("test-encryption-key-min-32-chars", zap.NewNop())
	logger := zap.NewNop()

	service := NewSchemaMetadataService(
		dbInstanceRepo,
		databaseRepo,
		nil, // database schema repo
		connectionMgr,
		logger,
	)

	assert.NotNil(t, service)
	assert.Equal(t, dbInstanceRepo, service.dbInstanceRepo)
	assert.Equal(t, databaseRepo, service.databaseRepo)
	assert.Nil(t, service.databaseSchemaRepo)
	assert.Equal(t, connectionMgr, service.connectionMgr)
	assert.Equal(t, logger, service.logger)
}

// TODO: Add integration tests for ExtractSchemaMetadata once we have test database setup
// TODO: Add integration tests for ExtractTableDetail once we have test database setup
