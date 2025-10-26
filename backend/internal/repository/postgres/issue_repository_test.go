package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupIssueRepositoryMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *IssueRepository) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewIssueRepository(db)
	return db, mock, repo
}

func TestIssueRepository_Create(t *testing.T) {
	projectID := uuid.New()
	authorID := uuid.New()
	databaseID := uuid.New()

	tests := []struct {
		name      string
		issue     *models.Issue
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name: "successful create",
			issue: &models.Issue{
				ProjectID:        projectID,
				Title:            "Test Issue",
				Description:      "Test Description",
				ChangeType:       models.ChangeTypeDDL,
				SQLContent:       "SELECT 1;",
				AuthorID:         authorID,
				TargetDatabaseID: databaseID,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO issues").
					WithArgs(
						sqlmock.AnyArg(), // id (UUID)
						projectID,
						databaseID,
						"Test Issue",
						"Test Description",
						models.ChangeTypeDDL,
						"SELECT 1;",
						sqlmock.AnyArg(), // rollback_sql
						models.IssueStatusDraft,
						models.RiskLevelMedium,
						authorID,
						sqlmock.AnyArg(), // assigned_reviewer_id
						sqlmock.AnyArg(), // scheduled_for
						sqlmock.AnyArg(), // executed_at
						sqlmock.AnyArg(), // execution_duration_ms
						sqlmock.AnyArg(), // affected_rows
						sqlmock.AnyArg(), // error_message
						sqlmock.AnyArg(), // created_at
						sqlmock.AnyArg(), // updated_at
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "validation error - missing title",
			issue: &models.Issue{
				ProjectID:        projectID,
				Description:      "Test Description",
				ChangeType:       models.ChangeTypeDDL,
				SQLContent:       "SELECT 1;",
				AuthorID:         authorID,
				TargetDatabaseID: databaseID,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				// No mock setup needed - validation fails before query
			},
			wantErr: true,
		},
		{
			name: "database error",
			issue: &models.Issue{
				ProjectID:        projectID,
				Title:            "Test Issue",
				Description:      "Test Description",
				ChangeType:       models.ChangeTypeDDL,
				SQLContent:       "SELECT 1;",
				AuthorID:         authorID,
				TargetDatabaseID: databaseID,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO issues").
					WithArgs(
						sqlmock.AnyArg(), // id
						sqlmock.AnyArg(), // project_id
						sqlmock.AnyArg(), // target_database_id
						sqlmock.AnyArg(), // title
						sqlmock.AnyArg(), // description
						sqlmock.AnyArg(), // change_type
						sqlmock.AnyArg(), // sql_content
						sqlmock.AnyArg(), // rollback_sql
						sqlmock.AnyArg(), // status
						sqlmock.AnyArg(), // risk_level
						sqlmock.AnyArg(), // author_id
						sqlmock.AnyArg(), // assigned_reviewer_id
						sqlmock.AnyArg(), // scheduled_for
						sqlmock.AnyArg(), // executed_at
						sqlmock.AnyArg(), // execution_duration_ms
						sqlmock.AnyArg(), // affected_rows
						sqlmock.AnyArg(), // error_message
						sqlmock.AnyArg(), // created_at
						sqlmock.AnyArg(), // updated_at
					).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, repo := setupIssueRepositoryMockDB(t)
			defer db.Close()

			tt.setupMock(mock)

			// Make a copy of the issue for testing to avoid mutation
			issueToCreate := &models.Issue{
				ID:               tt.issue.ID,
				ProjectID:        tt.issue.ProjectID,
				TargetDatabaseID: tt.issue.TargetDatabaseID,
				Title:            tt.issue.Title,
				Description:      tt.issue.Description,
				ChangeType:       tt.issue.ChangeType,
				SQLContent:       tt.issue.SQLContent,
				AuthorID:         tt.issue.AuthorID,
			}

			err := repo.Create(context.Background(), issueToCreate)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, issueToCreate.ID)
				assert.False(t, issueToCreate.CreatedAt.IsZero())
				assert.False(t, issueToCreate.UpdatedAt.IsZero())
				assert.Equal(t, models.IssueStatusDraft, issueToCreate.Status)
				assert.Equal(t, models.RiskLevelMedium, issueToCreate.RiskLevel)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestIssueRepository_GetByID(t *testing.T) {
	db, mock, repo := setupIssueRepositoryMockDB(t)
	defer db.Close()

	now := time.Now().UTC()
	issueID := uuid.New()
	projectID := uuid.New()
	authorID := uuid.New()
	databaseID := uuid.New()

	tests := []struct {
		name      string
		id        uuid.UUID
		setupMock func(sqlmock.Sqlmock)
		wantIssue *models.Issue
		wantErr   bool
	}{
		{
			name: "successful get",
			id:   issueID,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "project_id", "target_database_id", "title", "description", "change_type",
					"sql_content", "rollback_sql", "status", "risk_level", "author_id",
					"assigned_reviewer_id", "scheduled_for", "executed_at", "execution_duration_ms",
					"affected_rows", "error_message", "created_at", "updated_at",
				}).AddRow(
					issueID, projectID, databaseID, "Test Issue", "Test Description", models.ChangeTypeDDL,
					"SELECT 1;", nil, models.IssueStatusDraft, models.RiskLevelMedium, authorID,
					nil, nil, nil, nil, nil, nil, now, now,
				)
				mock.ExpectQuery("SELECT (.+) FROM issues WHERE id").
					WithArgs(issueID).
					WillReturnRows(rows)
			},
			wantIssue: &models.Issue{
				ID:               issueID,
				ProjectID:        projectID,
				TargetDatabaseID: databaseID,
				Title:            "Test Issue",
				Description:      "Test Description",
				ChangeType:       models.ChangeTypeDDL,
				SQLContent:       "SELECT 1;",
				Status:           models.IssueStatusDraft,
				RiskLevel:        models.RiskLevelMedium,
				AuthorID:         authorID,
				CreatedAt:        now,
				UpdatedAt:        now,
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   uuid.New(),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT (.+) FROM issues WHERE id").
					WithArgs(sqlmock.AnyArg()).
					WillReturnError(sql.ErrNoRows)
			},
			wantIssue: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			issue, err := repo.GetByID(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.name == "not found" {
					_, ok := err.(models.ErrNotFound)
					assert.True(t, ok, "Expected ErrNotFound error type")
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantIssue.ID, issue.ID)
				assert.Equal(t, tt.wantIssue.Title, issue.Title)
				assert.Equal(t, tt.wantIssue.Status, issue.Status)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestIssueRepository_Update(t *testing.T) {
	db, mock, repo := setupIssueRepositoryMockDB(t)
	defer db.Close()

	now := time.Now().UTC()
	issueID := uuid.New()
	projectID := uuid.New()
	authorID := uuid.New()
	databaseID := uuid.New()

	tests := []struct {
		name      string
		issue     *models.Issue
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name: "successful update",
			issue: &models.Issue{
				ID:               issueID,
				ProjectID:        projectID,
				Title:            "Updated Title",
				Description:      "Updated Description",
				ChangeType:       models.ChangeTypeDDL,
				SQLContent:       "SELECT 2;",
				Status:           models.IssueStatusPendingReview,
				RiskLevel:        models.RiskLevelMedium,
				AuthorID:         authorID,
				TargetDatabaseID: databaseID,
				CreatedAt:        now,
				UpdatedAt:        now,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE issues").
					WithArgs(
						issueID,
						"Updated Title",
						"Updated Description",
						models.ChangeTypeDDL,
						"SELECT 2;",
						sqlmock.AnyArg(), // rollback_sql
						models.IssueStatusPendingReview,
						models.RiskLevelMedium,
						databaseID,
						sqlmock.AnyArg(), // assigned_reviewer_id
						sqlmock.AnyArg(), // scheduled_for
						sqlmock.AnyArg(), // updated_at
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "not found",
			issue: &models.Issue{
				ID:               uuid.New(),
				ProjectID:        projectID,
				Title:            "Updated Title",
				Description:      "Updated Description",
				ChangeType:       models.ChangeTypeDDL,
				SQLContent:       "SELECT 2;",
				Status:           models.IssueStatusPendingReview,
				RiskLevel:        models.RiskLevelMedium,
				AuthorID:         authorID,
				TargetDatabaseID: databaseID,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("UPDATE issues").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			err := repo.Update(context.Background(), tt.issue)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, tt.issue.UpdatedAt.After(now) || tt.issue.UpdatedAt.Equal(now))
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestIssueRepository_Delete(t *testing.T) {
	db, mock, repo := setupIssueRepositoryMockDB(t)
	defer db.Close()

	tests := []struct {
		name      string
		id        uuid.UUID
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name: "successful delete",
			id:   uuid.New(),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM issues WHERE id").
					WithArgs(sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   uuid.New(),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM issues WHERE id").
					WithArgs(sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			err := repo.Delete(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestIssueRepository_List(t *testing.T) {
	db, mock, repo := setupIssueRepositoryMockDB(t)
	defer db.Close()

	now := time.Now().UTC()
	projectID := uuid.New()
	authorID := uuid.New()
	databaseID := uuid.New()

	tests := []struct {
		name      string
		filters   repository.ListFilters
		setupMock func(sqlmock.Sqlmock)
		wantLen   int
		wantErr   bool
	}{
		{
			name: "list all with default pagination",
			filters: repository.ListFilters{
				Limit:  10,
				Offset: 0,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "project_id", "target_database_id", "title", "description", "change_type",
					"sql_content", "rollback_sql", "status", "risk_level", "author_id",
					"assigned_reviewer_id", "scheduled_for", "executed_at", "execution_duration_ms",
					"affected_rows", "error_message", "created_at", "updated_at",
				}).
					AddRow(uuid.New(), projectID, databaseID, "Title 1", "Desc 1", models.ChangeTypeDDL, "SQL 1", nil, models.IssueStatusDraft, models.RiskLevelMedium, authorID, nil, nil, nil, nil, nil, nil, now, now).
					AddRow(uuid.New(), projectID, databaseID, "Title 2", "Desc 2", models.ChangeTypeDDL, "SQL 2", nil, models.IssueStatusApproved, models.RiskLevelMedium, authorID, nil, nil, nil, nil, nil, nil, now, now)

				mock.ExpectQuery("SELECT (.+) FROM issues ORDER BY").
					WillReturnRows(rows)
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name: "filter by status",
			filters: repository.ListFilters{
				Status: models.IssueStatusDraft,
				Limit:  10,
				Offset: 0,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "project_id", "target_database_id", "title", "description", "change_type",
					"sql_content", "rollback_sql", "status", "risk_level", "author_id",
					"assigned_reviewer_id", "scheduled_for", "executed_at", "execution_duration_ms",
					"affected_rows", "error_message", "created_at", "updated_at",
				}).
					AddRow(uuid.New(), projectID, databaseID, "Title 1", "Desc 1", models.ChangeTypeDDL, "SQL 1", nil, models.IssueStatusDraft, models.RiskLevelMedium, authorID, nil, nil, nil, nil, nil, nil, now, now)

				mock.ExpectQuery("SELECT (.+) FROM issues WHERE status").
					WithArgs(models.IssueStatusDraft).
					WillReturnRows(rows)
			},
			wantLen: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			issues, err := repo.List(context.Background(), tt.filters)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, issues, tt.wantLen)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestIssueRepository_Search(t *testing.T) {
	db, mock, repo := setupIssueRepositoryMockDB(t)
	defer db.Close()

	now := time.Now().UTC()
	projectID := uuid.New()
	authorID := uuid.New()
	databaseID := uuid.New()

	tests := []struct {
		name      string
		query     string
		filters   repository.ListFilters
		setupMock func(sqlmock.Sqlmock)
		wantLen   int
		wantErr   bool
	}{
		{
			name:  "search in title",
			query: "test",
			filters: repository.ListFilters{
				Limit:  10,
				Offset: 0,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "project_id", "target_database_id", "title", "description", "change_type",
					"sql_content", "rollback_sql", "status", "risk_level", "author_id",
					"assigned_reviewer_id", "scheduled_for", "executed_at", "execution_duration_ms",
					"affected_rows", "error_message", "created_at", "updated_at",
				}).
					AddRow(uuid.New(), projectID, databaseID, "Test Issue", "Description", "DDL", "SQL", nil, models.IssueStatusDraft, models.RiskLevelMedium, authorID, nil, nil, nil, nil, nil, nil, now, now)

				mock.ExpectQuery("SELECT (.+) FROM issues WHERE (.+) ILIKE").
					WithArgs("%test%").
					WillReturnRows(rows)
			},
			wantLen: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			issues, err := repo.Search(context.Background(), tt.query, tt.filters)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, issues, tt.wantLen)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestIssueRepository_Count(t *testing.T) {
	db, mock, repo := setupIssueRepositoryMockDB(t)
	defer db.Close()

	tests := []struct {
		name      string
		filters   repository.ListFilters
		setupMock func(sqlmock.Sqlmock)
		wantCount int64
		wantErr   bool
	}{
		{
			name:    "count all",
			filters: repository.ListFilters{},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(42)
				mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)
			},
			wantCount: 42,
			wantErr:   false,
		},
		{
			name: "count with filter",
			filters: repository.ListFilters{
				Status: models.IssueStatusDraft,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(10)
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM issues WHERE status").
					WithArgs(models.IssueStatusDraft).
					WillReturnRows(rows)
			},
			wantCount: 10,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			count, err := repo.Count(context.Background(), tt.filters)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, count)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
