package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAPITokenRepo is a minimal in-memory APITokenRepository for tests.
type fakeAPITokenRepo struct {
	mu    sync.Mutex
	token *models.APIToken // returned by GetByTokenHash when hash matches token.TokenHash and not revoked
}

func (f *fakeAPITokenRepo) Create(context.Context, *models.APIToken) error { return nil }

func (f *fakeAPITokenRepo) GetByTokenHash(_ context.Context, hash string) (*models.APIToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.token == nil || f.token.TokenHash != hash || f.token.RevokedAt != nil {
		return nil, repository.ErrNotFound
	}
	return f.token, nil
}

func (f *fakeAPITokenRepo) List(context.Context) ([]*models.APIToken, error) { return nil, nil }
func (f *fakeAPITokenRepo) Revoke(context.Context, uuid.UUID) error          { return nil }
func (f *fakeAPITokenRepo) UpdateLastUsed(context.Context, uuid.UUID) error  { return nil }

// fakeUserRepo is a minimal in-memory UserRepository for tests.
type fakeUserRepo struct {
	user *models.User
}

func (f *fakeUserRepo) Create(context.Context, *models.User) error { return nil }
func (f *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if f.user == nil || f.user.ID != id {
		return nil, repository.ErrNotFound
	}
	return f.user, nil
}

func (f *fakeUserRepo) GetByEmail(context.Context, string) (*models.User, error) {
	return nil, repository.ErrNotFound
}
func (f *fakeUserRepo) Update(context.Context, *models.User) error { return nil }
func (f *fakeUserRepo) Delete(context.Context, uuid.UUID) error    { return nil }
func (f *fakeUserRepo) List(context.Context, repository.UserFilters, int, int) ([]*models.User, error) {
	return nil, nil
}
func (f *fakeUserRepo) Count(context.Context, repository.UserFilters) (int64, error) {
	return 0, nil
}
func (f *fakeUserRepo) UpdateLastLogin(context.Context, uuid.UUID) error { return nil }

// newAPITokenFixture builds a middleware wired with fakes plus a valid plaintext token
// bound to an active OrganizationAdmin user.
func newAPITokenFixture(t *testing.T) (*AuthMiddleware, string, *fakeAPITokenRepo, *models.User) {
	t.Helper()
	tm, err := auth.NewTokenManager(testJWTSecret, 0)
	require.NoError(t, err)

	plaintext, hash, prefix, err := auth.GenerateAPIToken()
	require.NoError(t, err)

	user := &models.User{
		ID:               uuid.New(),
		Email:            "svc@example.com",
		OrganizationRole: rbac.OrganizationAdmin,
		IsActive:         true,
	}
	tokRepo := &fakeAPITokenRepo{token: &models.APIToken{
		ID:          uuid.New(),
		UserID:      user.ID,
		Name:        "test",
		TokenPrefix: prefix,
		TokenHash:   hash,
	}}
	userRepo := &fakeUserRepo{user: user}

	return NewAuthMiddleware(tm, tokRepo, userRepo), plaintext, tokRepo, user
}

func doAuth(mw *AuthMiddleware, authHeader string, handler echo.HandlerFunc) (*httptest.ResponseRecorder, error) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return rec, mw.Authenticate(handler)(c)
}

func TestAuthenticate_APIToken_Success(t *testing.T) {
	mw, plaintext, _, user := newAPITokenFixture(t)

	var captured *auth.Claims
	handler := func(c echo.Context) error {
		captured, _ = c.Get("user").(*auth.Claims)
		return c.NoContent(http.StatusOK)
	}

	rec, err := doAuth(mw, "Bearer "+plaintext, handler)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, captured)
	assert.Equal(t, user.ID, captured.UserID)
	assert.Equal(t, user.Email, captured.Email)
	assert.Equal(t, string(rbac.OrganizationAdmin), captured.OrganizationRole)
}

func TestAuthenticate_APIToken_UnknownHash(t *testing.T) {
	mw, _, _, _ := newAPITokenFixture(t)

	handler := func(c echo.Context) error {
		t.Fatal("handler should not be called")
		return nil
	}
	_, err := doAuth(mw, "Bearer deeb_pat_doesnotexist", handler)
	requireUnauthorized(t, err, "invalid token")
}

func TestAuthenticate_APIToken_Revoked(t *testing.T) {
	mw, plaintext, tokRepo, _ := newAPITokenFixture(t)
	now := time.Now()
	tokRepo.token.RevokedAt = &now

	handler := func(c echo.Context) error {
		t.Fatal("handler should not be called")
		return nil
	}
	// Revoked tokens are filtered at the repository layer -> not found -> invalid token.
	_, err := doAuth(mw, "Bearer "+plaintext, handler)
	requireUnauthorized(t, err, "invalid token")
}

func TestAuthenticate_APIToken_Expired(t *testing.T) {
	mw, plaintext, tokRepo, _ := newAPITokenFixture(t)
	past := time.Now().Add(-time.Hour)
	tokRepo.token.ExpiresAt = &past

	handler := func(c echo.Context) error {
		t.Fatal("handler should not be called")
		return nil
	}
	_, err := doAuth(mw, "Bearer "+plaintext, handler)
	requireUnauthorized(t, err, "expired or revoked")
}

func TestAuthenticate_APIToken_InactiveUser(t *testing.T) {
	mw, plaintext, _, user := newAPITokenFixture(t)
	user.IsActive = false

	handler := func(c echo.Context) error {
		t.Fatal("handler should not be called")
		return nil
	}
	_, err := doAuth(mw, "Bearer "+plaintext, handler)
	requireUnauthorized(t, err, "inactive")
}

// A deeb_pat_ value arriving via the cookie must NOT be treated as an API token; the
// cookie path is JWT-only, so it fails JWT validation.
func TestAuthenticate_APIToken_NotAcceptedViaCookie(t *testing.T) {
	mw, plaintext, _, _ := newAPITokenFixture(t)

	handler := func(c echo.Context) error {
		t.Fatal("handler should not be called")
		return nil
	}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "deeb_auth_token", Value: plaintext})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := mw.Authenticate(handler)(c)
	requireUnauthorized(t, err, "invalid token")
}

func requireUnauthorized(t *testing.T, err error, contains string) {
	t.Helper()
	require.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
	assert.Contains(t, httpErr.Message, contains)
}
