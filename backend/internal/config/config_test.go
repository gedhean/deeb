package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDatabaseURL(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
		want        DatabaseConfig
		wantErr     bool
	}{
		{
			name:        "valid postgresql URL",
			databaseURL: "postgresql://deeb:deeb_dev_password@localhost:5432/deeb?sslmode=disable",
			want: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "deeb",
				Password: "deeb_dev_password",
				Database: "deeb",
				SSLMode:  "disable",
			},
			wantErr: false,
		},
		{
			name:        "valid postgres URL",
			databaseURL: "postgres://testuser:testpass@db.example.com:5433/testdb?sslmode=require",
			want: DatabaseConfig{
				Host:     "db.example.com",
				Port:     5433,
				User:     "testuser",
				Password: "testpass",
				Database: "testdb",
				SSLMode:  "require",
			},
			wantErr: false,
		},
		{
			name:        "URL without password",
			databaseURL: "postgresql://deeb@localhost:5432/deeb?sslmode=disable",
			want: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "deeb",
				Password: "",
				Database: "deeb",
				SSLMode:  "disable",
			},
			wantErr: false,
		},
		{
			name:        "URL without port (defaults handled by config defaults)",
			databaseURL: "postgresql://deeb:pass@localhost/deeb",
			want: DatabaseConfig{
				Host:     "localhost",
				Port:     5432, // Will be from defaults
				User:     "deeb",
				Password: "pass",
				Database: "deeb",
				SSLMode:  "disable", // Will be from defaults
			},
			wantErr: false,
		},
		{
			name:        "invalid scheme",
			databaseURL: "mysql://user:pass@localhost:3306/db",
			wantErr:     true,
		},
		{
			name:        "invalid URL",
			databaseURL: "not a valid url",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			cfg.setDefaults()

			err := cfg.parseDatabaseURL(tt.databaseURL)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.Host, cfg.Database.Host)
			assert.Equal(t, tt.want.Port, cfg.Database.Port)
			assert.Equal(t, tt.want.User, cfg.Database.User)
			assert.Equal(t, tt.want.Password, cfg.Database.Password)
			assert.Equal(t, tt.want.Database, cfg.Database.Database)
			assert.Equal(t, tt.want.SSLMode, cfg.Database.SSLMode)
		})
	}
}

func TestLoadFromEnv_DatabaseURL(t *testing.T) {
	// Save original env vars
	originalVars := map[string]string{
		"DATABASE_URL": os.Getenv("DATABASE_URL"),
		"DB_HOST":      os.Getenv("DB_HOST"),
		"DB_PORT":      os.Getenv("DB_PORT"),
		"DB_USER":      os.Getenv("DB_USER"),
		"DB_PASSWORD":  os.Getenv("DB_PASSWORD"),
		"DB_NAME":      os.Getenv("DB_NAME"),
		"DB_SSLMODE":   os.Getenv("DB_SSLMODE"),
	}

	// Restore env vars after test
	defer func() {
		for k, v := range originalVars {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	t.Run("DATABASE_URL sets all database config", func(t *testing.T) {
		// Clear individual env vars
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_SSLMODE")

		// Set DATABASE_URL
		os.Setenv("DATABASE_URL", "postgresql://deeb:deeb_dev_password@localhost:5432/deeb?sslmode=disable")

		cfg := &Config{}
		cfg.setDefaults()
		cfg.loadFromEnv()

		assert.Equal(t, "localhost", cfg.Database.Host)
		assert.Equal(t, 5432, cfg.Database.Port)
		assert.Equal(t, "deeb", cfg.Database.User)
		assert.Equal(t, "deeb_dev_password", cfg.Database.Password)
		assert.Equal(t, "deeb", cfg.Database.Database)
		assert.Equal(t, "disable", cfg.Database.SSLMode)
	})

	t.Run("individual env vars override DATABASE_URL", func(t *testing.T) {
		// Set DATABASE_URL
		os.Setenv("DATABASE_URL", "postgresql://deeb:deeb_dev_password@localhost:5432/deeb?sslmode=disable")

		// Set individual override
		os.Setenv("DB_HOST", "overridden-host")
		os.Setenv("DB_PORT", "9999")

		cfg := &Config{}
		cfg.setDefaults()
		cfg.loadFromEnv()

		// These should be overridden
		assert.Equal(t, "overridden-host", cfg.Database.Host)
		assert.Equal(t, 9999, cfg.Database.Port)

		// These should come from DATABASE_URL
		assert.Equal(t, "deeb", cfg.Database.User)
		assert.Equal(t, "deeb_dev_password", cfg.Database.Password)
		assert.Equal(t, "deeb", cfg.Database.Database)
		assert.Equal(t, "disable", cfg.Database.SSLMode)
	})
}
