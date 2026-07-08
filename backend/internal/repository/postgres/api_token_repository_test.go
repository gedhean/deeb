package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func apiTokenRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "user_id", "name", "token_prefix", "token_hash",
		"last_used_at", "expires_at", "revoked_at", "created_at",
	})
}

func TestAPITokenRepository_Create(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewAPITokenRepository(db)
	ctx := context.Background()

	token := &models.APIToken{
		UserID:      uuid.New(),
		Name:        "rds-onboarder",
		TokenPrefix: "deeb_pat_ab12cd34",
		TokenHash:   "deadbeef",
	}

	mock.ExpectExec("INSERT INTO api_tokens").
		WithArgs(
			sqlmock.AnyArg(), // id
			token.UserID,
			token.Name,
			token.TokenPrefix,
			token.TokenHash,
			token.LastUsedAt,
			token.ExpiresAt,
			token.RevokedAt,
			sqlmock.AnyArg(), // created_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, token)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, token.ID)
	assert.NotZero(t, token.CreatedAt)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAPITokenRepository_GetByTokenHash(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewAPITokenRepository(db)
	ctx := context.Background()

	id := uuid.New()
	userID := uuid.New()
	now := time.Now()

	rows := apiTokenRows().AddRow(
		id, userID, "rds-onboarder", "deeb_pat_ab12cd34", "hash123",
		nil, nil, nil, now,
	)

	mock.ExpectQuery("SELECT (.+) FROM api_tokens WHERE token_hash").
		WithArgs("hash123").
		WillReturnRows(rows)

	token, err := repo.GetByTokenHash(ctx, "hash123")
	assert.NoError(t, err)
	require.NotNil(t, token)
	assert.Equal(t, id, token.ID)
	assert.Equal(t, userID, token.UserID)
	assert.True(t, token.IsActive())

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAPITokenRepository_GetByTokenHash_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewAPITokenRepository(db)
	ctx := context.Background()

	mock.ExpectQuery("SELECT (.+) FROM api_tokens WHERE token_hash").
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	token, err := repo.GetByTokenHash(ctx, "missing")
	assert.Nil(t, token)
	assert.True(t, errors.Is(err, repository.ErrNotFound))

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAPITokenRepository_List(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewAPITokenRepository(db)
	ctx := context.Background()

	now := time.Now()
	rows := apiTokenRows().
		AddRow(uuid.New(), uuid.New(), "one", "deeb_pat_1", "h1", nil, nil, nil, now).
		AddRow(uuid.New(), uuid.New(), "two", "deeb_pat_2", "h2", nil, nil, &now, now)

	mock.ExpectQuery("SELECT (.+) FROM api_tokens ORDER BY created_at DESC").
		WillReturnRows(rows)

	tokens, err := repo.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, tokens, 2)
	assert.False(t, tokens[1].IsActive(), "revoked token is not active")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAPITokenRepository_Revoke(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewAPITokenRepository(db)
	ctx := context.Background()

	id := uuid.New()
	mock.ExpectExec("UPDATE api_tokens SET revoked_at").
		WithArgs(id, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Revoke(ctx, id)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAPITokenRepository_Revoke_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewAPITokenRepository(db)
	ctx := context.Background()

	id := uuid.New()
	mock.ExpectExec("UPDATE api_tokens SET revoked_at").
		WithArgs(id, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Revoke(ctx, id)
	assert.True(t, errors.Is(err, repository.ErrNotFound))

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAPITokenRepository_UpdateLastUsed(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewAPITokenRepository(db)
	ctx := context.Background()

	id := uuid.New()
	mock.ExpectExec("UPDATE api_tokens SET last_used_at").
		WithArgs(id, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateLastUsed(ctx, id)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}
