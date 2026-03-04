package application

import (
	"context"

	"github.com/Lovealone1/nex21-api/internal/platform/errors"
)

// AuthProvider defines the infra port for authenticating users.
type AuthProvider interface {
	Login(ctx context.Context, email, password string) (*LoginResponse, error)
	GetUser(ctx context.Context, token string) (*UserDTO, error)
}

// AuthService implements the application business rules for authentication.
type AuthService struct {
	authProvider AuthProvider
}

// NewAuthService creates a new authentication service.
func NewAuthService(provider AuthProvider) *AuthService {
	return &AuthService{
		authProvider: provider,
	}
}

// Login validates credentials and asks the provider for a token.
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, errors.New(errors.InvalidArgument, "AuthService.Login", "email and password are required")
	}

	// Any specific application rules (e.g. rate-limiting checks) would be applied here,
	// then delegate the actual token issuance to the AuthProvider.
	resp, err := s.authProvider.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, errors.Wrap(err, errors.Unauthenticated, "AuthService.Login", "login failed")
	}

	return resp, nil
}

// Me retrieves the current authenticated user's profile information.
func (s *AuthService) Me(ctx context.Context, token string) (*UserDTO, error) {
	if token == "" {
		return nil, errors.New(errors.Unauthenticated, "AuthService.Me", "No token provided")
	}

	user, err := s.authProvider.GetUser(ctx, token)
	if err != nil {
		return nil, errors.Wrap(err, errors.Unauthenticated, "AuthService.Me", "failed to resolve user from token")
	}

	return user, nil
}
