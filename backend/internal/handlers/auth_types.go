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

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}
