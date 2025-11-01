package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestProject_Validate tests Project.Validate() method
func TestProject_Validate(t *testing.T) {
	tests := []struct {
		name    string
		project *Project
		wantErr error
	}{
		{
			name: "valid project",
			project: &Project{
				Name:        "Test Project",
				Description: stringPtr("Test Description"),
			},
			wantErr: nil,
		},
		{
			name: "valid project with empty description",
			project: &Project{
				Name:        "Test Project",
				Description: nil,
			},
			wantErr: nil,
		},
		{
			name: "valid project with trimmed whitespace",
			project: &Project{
				Name:        "  Test Project  ",
				Description: stringPtr("  Test Description  "),
			},
			wantErr: nil,
		},
		{
			name: "empty name",
			project: &Project{
				Name:        "",
				Description: stringPtr("Test Description"),
			},
			wantErr: ErrProjectNameRequired,
		},
		{
			name: "name too long",
			project: &Project{
				Name:        string(make([]byte, 101)), // 101 characters
				Description: stringPtr("Test Description"),
			},
			wantErr: ErrProjectNameTooLong,
		},
		{
			name: "name exactly 100 characters",
			project: &Project{
				Name:        string(make([]byte, 100)),
				Description: stringPtr("Test Description"),
			},
			wantErr: nil,
		},
		{
			name: "description too long",
			project: &Project{
				Name:        "Test Project",
				Description: stringPtr(string(make([]byte, 501))), // 501 characters
			},
			wantErr: ErrDescriptionTooLong,
		},
		{
			name: "description exactly 500 characters",
			project: &Project{
				Name:        "Test Project",
				Description: stringPtr(string(make([]byte, 500))),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.project.Validate()
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateCreateRequest tests ValidateCreateRequest function
func TestValidateCreateRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *CreateProjectRequest
		wantErr error
	}{
		{
			name: "valid request",
			req: &CreateProjectRequest{
				Name:        "Test Project",
				Description: "Test Description",
			},
			wantErr: nil,
		},
		{
			name: "valid request with empty description",
			req: &CreateProjectRequest{
				Name:        "Test Project",
				Description: "",
			},
			wantErr: nil,
		},
		{
			name: "empty name",
			req: &CreateProjectRequest{
				Name:        "",
				Description: "Test Description",
			},
			wantErr: ErrProjectNameRequired,
		},
		{
			name: "name too long",
			req: &CreateProjectRequest{
				Name:        string(make([]byte, 101)),
				Description: "Test Description",
			},
			wantErr: ErrProjectNameTooLong,
		},
		{
			name: "description too long",
			req: &CreateProjectRequest{
				Name:        "Test Project",
				Description: string(make([]byte, 501)),
			},
			wantErr: ErrDescriptionTooLong,
		},
		{
			name: "trimmed whitespace",
			req: &CreateProjectRequest{
				Name:        "  Test Project  ",
				Description: "  Test Description  ",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateRequest(tt.req)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateUpdateRequest tests ValidateUpdateRequest function
func TestValidateUpdateRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *UpdateProjectRequest
		wantErr error
	}{
		{
			name: "valid request with name",
			req: &UpdateProjectRequest{
				Name: stringPtr("Updated Project"),
			},
			wantErr: nil,
		},
		{
			name: "valid request with description",
			req: &UpdateProjectRequest{
				Description: stringPtr("Updated Description"),
			},
			wantErr: nil,
		},
		{
			name: "valid request with both fields",
			req: &UpdateProjectRequest{
				Name:        stringPtr("Updated Project"),
				Description: stringPtr("Updated Description"),
			},
			wantErr: nil,
		},
		{
			name: "valid request with nil fields",
			req: &UpdateProjectRequest{
				Name:        nil,
				Description: nil,
			},
			wantErr: nil,
		},
		{
			name: "empty name",
			req: &UpdateProjectRequest{
				Name: stringPtr(""),
			},
			wantErr: ErrProjectNameRequired,
		},
		{
			name: "name too long",
			req: &UpdateProjectRequest{
				Name: stringPtr(string(make([]byte, 101))),
			},
			wantErr: ErrProjectNameTooLong,
		},
		{
			name: "description too long",
			req: &UpdateProjectRequest{
				Description: stringPtr(string(make([]byte, 501))),
			},
			wantErr: ErrDescriptionTooLong,
		},
		{
			name: "trimmed whitespace",
			req: &UpdateProjectRequest{
				Name:        stringPtr("  Updated Project  "),
				Description: stringPtr("  Updated Description  "),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateRequest(tt.req)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDatabaseInstance_Validate tests DatabaseInstance.Validate() method
func TestDatabaseInstance_Validate(t *testing.T) {
	tests := []struct {
		name    string
		instance *DatabaseInstance
		wantErr error
	}{
		{
			name: "valid MySQL instance",
			instance: &DatabaseInstance{
				Name:         "Test Instance",
				Host:         "localhost",
				Port:         3306,
				DatabaseType: DatabaseTypeMySQL,
				Username:     "testuser",
			},
			wantErr: nil,
		},
		{
			name: "valid PostgreSQL instance",
			instance: &DatabaseInstance{
				Name:         "Test Instance",
				Host:         "localhost",
				Port:         5432,
				DatabaseType: DatabaseTypePostgreSQL,
				Username:     "testuser",
			},
			wantErr: nil,
		},
		{
			name: "empty name",
			instance: &DatabaseInstance{
				Name:         "",
				Host:         "localhost",
				Port:         5432,
				DatabaseType: DatabaseTypePostgreSQL,
				Username:     "testuser",
			},
			wantErr: ErrDatabaseInstanceNameRequired,
		},
		{
			name: "name too long",
			instance: &DatabaseInstance{
				Name:         string(make([]byte, 101)),
				Host:         "localhost",
				Port:         5432,
				DatabaseType: DatabaseTypePostgreSQL,
				Username:     "testuser",
			},
			wantErr: ErrDatabaseInstanceNameTooLong,
		},
		{
			name: "empty host",
			instance: &DatabaseInstance{
				Name:         "Test Instance",
				Host:         "",
				Port:         5432,
				DatabaseType: DatabaseTypePostgreSQL,
				Username:     "testuser",
			},
			wantErr: ErrHostRequired,
		},
		{
			name: "host too long",
			instance: &DatabaseInstance{
				Name:         "Test Instance",
				Host:         string(make([]byte, 256)),
				Port:         5432,
				DatabaseType: DatabaseTypePostgreSQL,
				Username:     "testuser",
			},
			wantErr: ErrHostTooLong,
		},
		{
			name: "invalid port - zero",
			instance: &DatabaseInstance{
				Name:         "Test Instance",
				Host:         "localhost",
				Port:         0,
				DatabaseType: DatabaseTypePostgreSQL,
				Username:     "testuser",
			},
			wantErr: ErrPortInvalid,
		},
		{
			name: "invalid port - too high",
			instance: &DatabaseInstance{
				Name:         "Test Instance",
				Host:         "localhost",
				Port:         65536,
				DatabaseType: DatabaseTypePostgreSQL,
				Username:     "testuser",
			},
			wantErr: ErrPortInvalid,
		},
		{
			name: "valid port - minimum",
			instance: &DatabaseInstance{
				Name:         "Test Instance",
				Host:         "localhost",
				Port:         1,
				DatabaseType: DatabaseTypePostgreSQL,
				Username:     "testuser",
			},
			wantErr: nil,
		},
		{
			name: "valid port - maximum",
			instance: &DatabaseInstance{
				Name:         "Test Instance",
				Host:         "localhost",
				Port:         65535,
				DatabaseType: DatabaseTypePostgreSQL,
				Username:     "testuser",
			},
			wantErr: nil,
		},
		{
			name: "empty database type",
			instance: &DatabaseInstance{
				Name:         "Test Instance",
				Host:         "localhost",
				Port:         5432,
				DatabaseType: "",
				Username:     "testuser",
			},
			wantErr: ErrDatabaseTypeRequired,
		},
		{
			name: "invalid database type",
			instance: &DatabaseInstance{
				Name:         "Test Instance",
				Host:         "localhost",
				Port:         5432,
				DatabaseType: "invalid",
				Username:     "testuser",
			},
			wantErr: ErrDatabaseTypeInvalid,
		},
		{
			name: "empty username",
			instance: &DatabaseInstance{
				Name:         "Test Instance",
				Host:         "localhost",
				Port:         5432,
				DatabaseType: DatabaseTypePostgreSQL,
				Username:     "",
			},
			wantErr: ErrUsernameRequired,
		},
		{
			name: "trimmed whitespace",
			instance: &DatabaseInstance{
				Name:         "  Test Instance  ",
				Host:         "  localhost  ",
				Port:         5432,
				DatabaseType: DatabaseTypePostgreSQL,
				Username:     "testuser",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.instance.Validate()
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateCreateDatabaseInstanceRequest tests ValidateCreateDatabaseInstanceRequest function
func TestValidateCreateDatabaseInstanceRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *CreateDatabaseInstanceRequest
		wantErr error
	}{
		{
			name: "valid request",
			req: &CreateDatabaseInstanceRequest{
				Name:         "Test Instance",
				Host:         "localhost",
				Port:         5432,
				DatabaseType: DatabaseTypePostgreSQL,
				Username:     "testuser",
			},
			wantErr: nil,
		},
		{
			name: "empty name",
			req: &CreateDatabaseInstanceRequest{
				Name:         "",
				Host:         "localhost",
				Port:         5432,
				DatabaseType: DatabaseTypePostgreSQL,
				Username:     "testuser",
			},
			wantErr: ErrDatabaseInstanceNameRequired,
		},
		{
			name: "name too long",
			req: &CreateDatabaseInstanceRequest{
				Name:         string(make([]byte, 101)),
				Host:         "localhost",
				Port:         5432,
				DatabaseType: DatabaseTypePostgreSQL,
				Username:     "testuser",
			},
			wantErr: ErrDatabaseInstanceNameTooLong,
		},
		{
			name: "empty host",
			req: &CreateDatabaseInstanceRequest{
				Name:         "Test Instance",
				Host:         "",
				Port:         5432,
				DatabaseType: DatabaseTypePostgreSQL,
				Username:     "testuser",
			},
			wantErr: ErrHostRequired,
		},
		{
			name: "invalid port",
			req: &CreateDatabaseInstanceRequest{
				Name:         "Test Instance",
				Host:         "localhost",
				Port:         0,
				DatabaseType: DatabaseTypePostgreSQL,
				Username:     "testuser",
			},
			wantErr: ErrPortInvalid,
		},
		{
			name: "invalid database type",
			req: &CreateDatabaseInstanceRequest{
				Name:         "Test Instance",
				Host:         "localhost",
				Port:         5432,
				DatabaseType: "invalid",
				Username:     "testuser",
			},
			wantErr: ErrDatabaseTypeInvalid,
		},
		{
			name: "empty username",
			req: &CreateDatabaseInstanceRequest{
				Name:         "Test Instance",
				Host:         "localhost",
				Port:         5432,
				DatabaseType: DatabaseTypePostgreSQL,
				Username:     "",
			},
			wantErr: ErrUsernameRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateDatabaseInstanceRequest(tt.req)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateUpdateDatabaseInstanceRequest tests ValidateUpdateDatabaseInstanceRequest function
func TestValidateUpdateDatabaseInstanceRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *UpdateDatabaseInstanceRequest
		wantErr error
	}{
		{
			name: "valid request with name",
			req: &UpdateDatabaseInstanceRequest{
				Name: stringPtr("Updated Instance"),
			},
			wantErr: nil,
		},
		{
			name: "valid request with host",
			req: &UpdateDatabaseInstanceRequest{
				Host: stringPtr("updated-host"),
			},
			wantErr: nil,
		},
		{
			name: "valid request with port",
			req: &UpdateDatabaseInstanceRequest{
				Port: intPtr(3306),
			},
			wantErr: nil,
		},
		{
			name: "valid request with nil fields",
			req: &UpdateDatabaseInstanceRequest{
				Name: nil,
				Host: nil,
			},
			wantErr: nil,
		},
		{
			name: "empty name",
			req: &UpdateDatabaseInstanceRequest{
				Name: stringPtr(""),
			},
			wantErr: ErrDatabaseInstanceNameRequired,
		},
		{
			name: "name too long",
			req: &UpdateDatabaseInstanceRequest{
				Name: stringPtr(string(make([]byte, 101))),
			},
			wantErr: ErrDatabaseInstanceNameTooLong,
		},
		{
			name: "host too long",
			req: &UpdateDatabaseInstanceRequest{
				Host: stringPtr(string(make([]byte, 256))),
			},
			wantErr: ErrHostTooLong,
		},
		{
			name: "invalid port",
			req: &UpdateDatabaseInstanceRequest{
				Port: intPtr(0),
			},
			wantErr: ErrPortInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateDatabaseInstanceRequest(tt.req)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDatabase_Validate tests Database.Validate() method
func TestDatabase_Validate(t *testing.T) {
	tests := []struct {
		name    string
		database *Database
		wantErr error
	}{
		{
			name: "valid database",
			database: &Database{
				DatabaseName:       "test_db",
				DatabaseInstanceID: uuid.New(),
				ProjectID:          uuid.New(),
				EnvironmentID:      uuid.New(),
			},
			wantErr: nil,
		},
		{
			name: "empty database name",
			database: &Database{
				DatabaseName:       "",
				DatabaseInstanceID: uuid.New(),
				ProjectID:          uuid.New(),
				EnvironmentID:      uuid.New(),
			},
			wantErr: ErrDatabaseNameRequired,
		},
		{
			name: "database name too long",
			database: &Database{
				DatabaseName:       string(make([]byte, 256)),
				DatabaseInstanceID: uuid.New(),
				ProjectID:          uuid.New(),
				EnvironmentID:      uuid.New(),
			},
			wantErr: ErrDatabaseNameTooLong,
		},
		{
			name: "nil database instance ID",
			database: &Database{
				DatabaseName:       "test_db",
				DatabaseInstanceID: uuid.Nil,
				ProjectID:          uuid.New(),
				EnvironmentID:      uuid.New(),
			},
			wantErr: ErrDatabaseInstanceIDRequired,
		},
		{
			name: "nil project ID",
			database: &Database{
				DatabaseName:       "test_db",
				DatabaseInstanceID: uuid.New(),
				ProjectID:          uuid.Nil,
				EnvironmentID:      uuid.New(),
			},
			wantErr: ErrDatabaseProjectIDRequired,
		},
		{
			name: "nil environment ID",
			database: &Database{
				DatabaseName:       "test_db",
				DatabaseInstanceID: uuid.New(),
				ProjectID:          uuid.New(),
				EnvironmentID:      uuid.Nil,
			},
			wantErr: ErrDatabaseEnvironmentIDRequired,
		},
		{
			name: "trimmed whitespace",
			database: &Database{
				DatabaseName:       "  test_db  ",
				DatabaseInstanceID: uuid.New(),
				ProjectID:          uuid.New(),
				EnvironmentID:      uuid.New(),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.database.Validate()
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateCreateDatabaseRequest tests ValidateCreateDatabaseRequest function
func TestValidateCreateDatabaseRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *CreateDatabaseRequest
		wantErr error
	}{
		{
			name: "valid request",
			req: &CreateDatabaseRequest{
				DatabaseName:       "test_db",
				DatabaseInstanceID: uuid.New(),
				ProjectID:          uuid.New(),
				EnvironmentID:      uuid.New(),
			},
			wantErr: nil,
		},
		{
			name: "empty database name",
			req: &CreateDatabaseRequest{
				DatabaseName:       "",
				DatabaseInstanceID: uuid.New(),
				ProjectID:          uuid.New(),
				EnvironmentID:      uuid.New(),
			},
			wantErr: ErrDatabaseNameRequired,
		},
		{
			name: "database name too long",
			req: &CreateDatabaseRequest{
				DatabaseName:       string(make([]byte, 256)),
				DatabaseInstanceID: uuid.New(),
				ProjectID:          uuid.New(),
				EnvironmentID:      uuid.New(),
			},
			wantErr: ErrDatabaseNameTooLong,
		},
		{
			name: "nil database instance ID",
			req: &CreateDatabaseRequest{
				DatabaseName:       "test_db",
				DatabaseInstanceID: uuid.Nil,
				ProjectID:          uuid.New(),
				EnvironmentID:      uuid.New(),
			},
			wantErr: ErrDatabaseInstanceIDRequired,
		},
		{
			name: "nil project ID",
			req: &CreateDatabaseRequest{
				DatabaseName:       "test_db",
				DatabaseInstanceID: uuid.New(),
				ProjectID:          uuid.Nil,
				EnvironmentID:      uuid.New(),
			},
			wantErr: ErrDatabaseProjectIDRequired,
		},
		{
			name: "nil environment ID",
			req: &CreateDatabaseRequest{
				DatabaseName:       "test_db",
				DatabaseInstanceID: uuid.New(),
				ProjectID:          uuid.New(),
				EnvironmentID:      uuid.Nil,
			},
			wantErr: ErrDatabaseEnvironmentIDRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateDatabaseRequest(tt.req)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateUpdateDatabaseRequest tests ValidateUpdateDatabaseRequest function
func TestValidateUpdateDatabaseRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *UpdateDatabaseRequest
		wantErr error
	}{
		{
			name: "valid request with name",
			req: &UpdateDatabaseRequest{
				DatabaseName: stringPtr("updated_db"),
			},
			wantErr: nil,
		},
		{
			name: "valid request with nil fields",
			req: &UpdateDatabaseRequest{
				DatabaseName: nil,
			},
			wantErr: nil,
		},
		{
			name: "empty database name",
			req: &UpdateDatabaseRequest{
				DatabaseName: stringPtr(""),
			},
			wantErr: ErrDatabaseNameRequired,
		},
		{
			name: "database name too long",
			req: &UpdateDatabaseRequest{
				DatabaseName: stringPtr(string(make([]byte, 256))),
			},
			wantErr: ErrDatabaseNameTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateDatabaseRequest(tt.req)
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

