package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID               uuid.UUID  `json:"id"                   db:"id"`
	Email            string     `json:"email"                db:"email"`
	PasswordHash     string     `json:"-"                    db:"password_hash"`
	DisplayName      *string    `json:"display_name"         db:"display_name"`
	OrganizationRole string     `json:"organization_role"    db:"organization_role"`
	IsActive         bool       `json:"is_active"            db:"is_active"`
	OAuthProvider    *string    `json:"oauth_provider"       db:"oauth_provider"`
	OAuthID          *string    `json:"oauth_id"             db:"oauth_id"`
	TwoFactorEnabled bool       `json:"two_factor_enabled"   db:"two_factor_enabled"`
	TwoFactorSecret  *string    `json:"-"                    db:"two_factor_secret"`
	LastLogin        *time.Time `json:"last_login,omitempty" db:"last_login_at"`
	CreatedAt        time.Time  `json:"created_at"           db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"           db:"updated_at"`
}

// CreateUserRequest represents the request to create a new user
type CreateUserRequest struct {
	Email            string `json:"email"             validate:"required,email"`
	Password         string `json:"password"          validate:"required,min=8"`
	OrganizationRole string `json:"organization_role" validate:"required,oneof=OrganizationAdmin OrganizationDBA OrganizationDeveloper"`
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	OrganizationRole *string `json:"organization_role,omitempty"  validate:"omitempty,oneof=OrganizationAdmin OrganizationDBA OrganizationDeveloper"`
	TwoFactorEnabled *bool   `json:"two_factor_enabled,omitempty"`
	IsActive         *bool   `json:"is_active,omitempty"`
}

// UserResponse represents a user response (without sensitive fields)
type UserResponse struct {
	ID               uuid.UUID  `json:"id"`
	Email            string     `json:"email"`
	OrganizationRole string     `json:"organization_role"`
	TwoFactorEnabled bool       `json:"two_factor_enabled"`
	IsActive         bool       `json:"is_active"`
	LastLogin        *time.Time `json:"last_login,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// ToResponse converts a User to UserResponse
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:               u.ID,
		Email:            u.Email,
		OrganizationRole: u.OrganizationRole,
		TwoFactorEnabled: u.TwoFactorEnabled,
		IsActive:         u.IsActive,
		LastLogin:        u.LastLogin,
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
	}
}
