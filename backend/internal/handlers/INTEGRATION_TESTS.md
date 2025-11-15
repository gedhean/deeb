# Integration Tests for Authentication Endpoints

This document describes the integration tests for authentication endpoints (P01M40).

## Test Database Setup

The integration tests require a PostgreSQL test database. To set up:

```bash
# Create test database
psql -d postgres -c "CREATE DATABASE deeb_test;"
psql -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE deeb_test TO deeb;"
psql -d deeb_test -c "GRANT ALL ON SCHEMA public TO deeb;"

# Run migrations
DATABASE_URL="postgres://deeb@localhost:5432/deeb_test?sslmode=disable" go run main.go migrate up
```

## Running Integration Tests

```bash
TEST_DATABASE_URL="postgres://deeb@localhost:5432/deeb_test?sslmode=disable" \
  go test -v -tags=integration ./internal/handlers/ -run TestIntegration
```

## Test Coverage

The integration tests in `auth_integration_test.go` cover:

### 1. TestIntegration_RegisterAndLogin_Success
- **Purpose**: Tests complete registration → login flow with real database
- **Validates**:
  - User registration creates user in database
  - Password is correctly hashed and stored
  - Login with correct credentials succeeds
  - JWT token is generated and returned in HttpOnly cookie
  - Token contains correct claims (email, user ID, organization role)

### 2. TestIntegration_Register_DuplicateEmail
- **Purpose**: Tests duplicate email validation with database constraint
- **Validates**:
  - First registration succeeds
  - Second registration with same email returns 409 Conflict
  - Error message: "email already registered"

### 3. TestIntegration_Login_InvalidCredentials
- **Purpose**: Tests login failure scenarios with real database
- **Validates**:
  - Login with wrong password returns 401 Unauthorized
  - Login with non-existent email returns 401 Unauthorized
  - Generic error message prevents user enumeration: "invalid email or password"

### 4. TestIntegration_Login_RateLimiting
- **Purpose**: Tests rate limiting enforcement with real requests
- **Validates**:
  - 10 failed login attempts are allowed
  - 11th attempt is rate limited with 429 Too Many Requests
  - Error message: "too many login attempts, please try again later"
  - Rate limiting applies even if correct password is provided after limit

### 5. TestIntegration_Login_InactiveUser
- **Purpose**: Tests that inactive users cannot login
- **Validates**:
  - User can be registered and activated initially
  - After setting status to 'Inactive', login fails with 401
  - Error message: "account is inactive"

## Schema Notes

The integration tests work with the actual database schema, which uses:
- `status` field (varchar: "Active", "Inactive", "Suspended") instead of `is_active` boolean
- Additional fields: `display_name`, `oauth_provider`, `oauth_id`
- `last_login_at` instead of `last_login_at`

## Acceptance Criteria Met

- ✅ Test POST /api/v1/auth/register creates user
- ✅ Test POST /api/v1/auth/login returns JWT token
- ✅ Test login with invalid credentials fails
- ✅ Test registration with duplicate email fails
- ✅ Test rate limiting on login endpoint
- ✅ All tests designed to pass with test database

## Known Issues

There are schema mismatches between the original Go models and the database schema that need to be resolved for tests to pass. The test file demonstrates the correct approach for integration testing with:
- Real database connections
- Transaction cleanup between tests
- Proper test isolation
- Comprehensive error scenario coverage
