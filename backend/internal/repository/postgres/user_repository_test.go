package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	return sqlxDB, mock
}

func TestUserRepository_Create(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{
		Email:            "test@example.com",
		PasswordHash:     "hashedpassword",
		OrganizationRole: "OrganizationDeveloper",
		TwoFactorEnabled: false,
		TwoFactorSecret:  "",
		IsActive:         true,
	}

	mock.ExpectExec("INSERT INTO users").
		WithArgs(
			sqlmock.AnyArg(), // id
			user.Email,
			user.PasswordHash,
			user.OrganizationRole,
			user.TwoFactorEnabled,
			user.TwoFactorSecret,
			user.IsActive,
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, user)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, user.ID)
	assert.NotZero(t, user.CreatedAt)
	assert.NotZero(t, user.UpdatedAt)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByID(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "email", "password_hash", "display_name", "organization_role",
		"is_active", "oauth_provider", "oauth_id",
		"two_factor_enabled", "two_factor_secret",
		"last_login_at", "created_at", "updated_at",
	}).AddRow(
		userID, "test@example.com", "hashedpassword", "", "OrganizationDeveloper",
		true, "", "",
		false, "",
		nil, now, now,
	)

	mock.ExpectQuery("SELECT (.+) FROM users WHERE id").
		WithArgs(userID).
		WillReturnRows(rows)

	user, err := repo.GetByID(ctx, userID)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "OrganizationDeveloper", user.OrganizationRole)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectQuery("SELECT (.+) FROM users WHERE id").
		WithArgs(userID).
		WillReturnError(sql.ErrNoRows)

	user, err := repo.GetByID(ctx, userID)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "user not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	email := "test@example.com"
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "email", "password_hash", "display_name", "organization_role",
		"is_active", "oauth_provider", "oauth_id",
		"two_factor_enabled", "two_factor_secret",
		"last_login_at", "created_at", "updated_at",
	}).AddRow(
		userID, email, "hashedpassword", "", "OrganizationDeveloper",
		true, "", "",
		false, "",
		nil, now, now,
	)

	mock.ExpectQuery("SELECT (.+) FROM users WHERE email").
		WithArgs(email).
		WillReturnRows(rows)

	user, err := repo.GetByEmail(ctx, email)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, email, user.Email)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Update(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{
		ID:               uuid.New(),
		Email:            "updated@example.com",
		PasswordHash:     "newhashedpassword",
		OrganizationRole: "OrganizationAdmin",
		TwoFactorEnabled: true,
		TwoFactorSecret:  "secret",
		IsActive:         true,
		CreatedAt:        time.Now(),
	}

	mock.ExpectExec("UPDATE users").
		WithArgs(
			user.ID,
			user.Email,
			user.PasswordHash,
			user.OrganizationRole,
			user.TwoFactorEnabled,
			user.TwoFactorSecret,
			user.IsActive,
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Update(ctx, user)
	assert.NoError(t, err)
	assert.NotZero(t, user.UpdatedAt)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Update_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{
		ID:               uuid.New(),
		Email:            "test@example.com",
		PasswordHash:     "hashedpassword",
		OrganizationRole: "OrganizationDeveloper",
	}

	mock.ExpectExec("UPDATE users").
		WithArgs(
			user.ID,
			user.Email,
			user.PasswordHash,
			user.OrganizationRole,
			user.TwoFactorEnabled,
			user.TwoFactorSecret,
			user.IsActive,
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Update(ctx, user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Delete(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectExec("DELETE FROM users WHERE id").
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, userID)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Delete_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectExec("DELETE FROM users WHERE id").
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(ctx, userID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_List(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	now := time.Now()
	user1ID := uuid.New()
	user2ID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "email", "password_hash", "display_name", "organization_role",
		"is_active", "oauth_provider", "oauth_id",
		"two_factor_enabled", "two_factor_secret",
		"last_login_at", "created_at", "updated_at",
	}).
		AddRow(user1ID, "user1@example.com", "hash1", "", "OrganizationDeveloper", true, "", "", false, "", nil, now, now).
		AddRow(user2ID, "user2@example.com", "hash2", "", "OrganizationAdmin", true, "", "", false, "", nil, now, now)

	mock.ExpectQuery("SELECT (.+) FROM users ORDER BY created_at DESC").
		WithArgs(10, 0).
		WillReturnRows(rows)

	users, err := repo.List(ctx, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "user1@example.com", users[0].Email)
	assert.Equal(t, "user2@example.com", users[1].Email)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_UpdateLastLogin(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectExec("UPDATE users SET last_login_at").
		WithArgs(userID, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateLastLogin(ctx, userID)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}
