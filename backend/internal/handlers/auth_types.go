package handlers

import (
	"github.com/nsxbet/deeb/internal/models"
)

// RegisterRequest represents the request body for user registration
type RegisterRequest struct {
	Email            string  `json:"email"`
	Password         string  `json:"password"`
	OrganizationRole *string `json:"organization_role,omitempty"`
}

// RegisterResponse represents the response body for successful registration
type RegisterResponse struct {
	User *models.UserResponse `json:"user"`
}

// LoginRequest represents the request body for user login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents the response body for successful login
type LoginResponse struct {
	Token string               `json:"token,omitempty"` // Deprecated: token is now sent via HttpOnly cookie
	User  *models.UserResponse `json:"user"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}
