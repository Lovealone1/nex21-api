package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Lovealone1/nex21-api/internal/modules/auth/application"
	errors "github.com/Lovealone1/nex21-api/internal/platform/apperrors"
	// "github.com/Lovealone1/nex21-api/internal/platform/observability"
)

// SupabaseClient implements the AuthProvider interface using GoTrue endpoints.
type SupabaseClient struct {
	baseURL        string
	anonKey        string
	serviceRoleKey string
	client         *http.Client
}

type supabaseTokenRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type supabaseError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Message          string `json:"message"` // sometimes API uses message instead
}

// NewSupabaseClient provisions the client with connection configs.
func NewSupabaseClient(url, anonKey, serviceRoleKey string) *SupabaseClient {
	return &SupabaseClient{
		baseURL:        url,
		anonKey:        anonKey,
		serviceRoleKey: serviceRoleKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Login invokes the GoTrue /auth/v1/token endpoint with password grant.
func (s *SupabaseClient) Login(ctx context.Context, email, password string) (*application.LoginResponse, error) {
	endpoint := fmt.Sprintf("%s/auth/v1/token?grant_type=password", s.baseURL)

	reqBody, _ := json.Marshal(supabaseTokenRequest{
		Email:    email,
		Password: password,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, errors.New(errors.Internal, "SupabaseClient.Login", "Failed to build request")
	}

	req.Header.Set("apikey", s.anonKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, errors.New(errors.Unavailable, "SupabaseClient.Login", "Failed to reach identity provider")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var supErr supabaseError
		if err := json.NewDecoder(resp.Body).Decode(&supErr); err == nil {
			// Map specific 400/401 errors
			msg := supErr.ErrorDescription
			if msg == "" {
				msg = supErr.Message
			}
			return nil, errors.New(errors.Unauthenticated, "SupabaseClient.Login", msg)
		}
		return nil, errors.Errorf(errors.Internal, "SupabaseClient.Login", "IdentityProvider returned status: %d", resp.StatusCode)
	}

	var parsedResp application.LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsedResp); err != nil {
		return nil, errors.New(errors.Internal, "SupabaseClient.Login", "Failed to decode auth response")
	}

	return &parsedResp, nil
}

// GetUser invokes the GoTrue /auth/v1/user endpoint with the provided bearer token.
func (s *SupabaseClient) GetUser(ctx context.Context, token string) (*application.UserDTO, error) {
	endpoint := fmt.Sprintf("%s/auth/v1/user", s.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.New(errors.Internal, "SupabaseClient.GetUser", "Failed to build request")
	}

	req.Header.Set("apikey", s.anonKey)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, errors.New(errors.Unavailable, "SupabaseClient.GetUser", "Failed to reach identity provider")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, errors.New(errors.Unauthenticated, "SupabaseClient.GetUser", "Invalid or expired token")
	}

	var parsedResp application.UserDTO
	if err := json.NewDecoder(resp.Body).Decode(&parsedResp); err != nil {
		return nil, errors.New(errors.Internal, "SupabaseClient.GetUser", "Failed to decode auth response")
	}

	return &parsedResp, nil
}

type supabaseAdminCreateRequest struct {
	Email        string                 `json:"email"`
	Password     string                 `json:"password"`
	EmailConfirm bool                   `json:"email_confirm"`
	UserMetadata map[string]interface{} `json:"user_metadata"`
}

type supabaseAdminCreateResponse struct {
	ID string `json:"id"`
}

// AdminCreateUser invokes GoTrue Admin API to create a user and map their UID.
// Requires serviceRoleKey.
func (s *SupabaseClient) AdminCreateUser(ctx context.Context, email, password string, metadata map[string]interface{}) (string, error) {
	if s.serviceRoleKey == "" {
		return "", errors.New(errors.Internal, "SupabaseClient.AdminCreateUser", "Service role key not configured")
	}

	endpoint := fmt.Sprintf("%s/auth/v1/admin/users", s.baseURL)

	reqBody, _ := json.Marshal(supabaseAdminCreateRequest{
		Email:        email,
		Password:     password,
		EmailConfirm: true, // Auto confirm for admin-created profiles
		UserMetadata: metadata,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", errors.New(errors.Internal, "SupabaseClient.AdminCreateUser", "Failed to build request")
	}

	// Admin operations require the service_role key to bypass RLS and Auth rules
	req.Header.Set("apikey", s.serviceRoleKey)
	req.Header.Set("Authorization", "Bearer "+s.serviceRoleKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", errors.New(errors.Unavailable, "SupabaseClient.AdminCreateUser", "Failed to reach identity provider")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var supErr supabaseError
		if err := json.NewDecoder(resp.Body).Decode(&supErr); err == nil {
			msg := supErr.ErrorDescription
			if msg == "" {
				msg = supErr.Message
			}
			return "", errors.New(errors.Internal, "SupabaseClient.AdminCreateUser", msg)
		}
		return "", errors.Errorf(errors.Internal, "SupabaseClient.AdminCreateUser", "IdentityProvider returned status: %d", resp.StatusCode)
	}

	var parsedResp supabaseAdminCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsedResp); err != nil {
		return "", errors.New(errors.Internal, "SupabaseClient.AdminCreateUser", "Failed to decode auth response")
	}

	return parsedResp.ID, nil
}

// AdminDeleteUser invokes GoTrue Admin API to permanently delete a user by UID.
// Requires serviceRoleKey.
func (s *SupabaseClient) AdminDeleteUser(ctx context.Context, uid string) error {
	if s.serviceRoleKey == "" {
		return errors.New(errors.Internal, "SupabaseClient.AdminDeleteUser", "Service role key not configured")
	}

	endpoint := fmt.Sprintf("%s/auth/v1/admin/users/%s", s.baseURL, uid)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return errors.New(errors.Internal, "SupabaseClient.AdminDeleteUser", "Failed to build request")
	}

	req.Header.Set("apikey", s.serviceRoleKey)
	req.Header.Set("Authorization", "Bearer "+s.serviceRoleKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return errors.New(errors.Unavailable, "SupabaseClient.AdminDeleteUser", "Failed to reach identity provider")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var supErr supabaseError
		if err := json.NewDecoder(resp.Body).Decode(&supErr); err == nil {
			msg := supErr.ErrorDescription
			if msg == "" {
				msg = supErr.Message
			}
			return errors.New(errors.Internal, "SupabaseClient.AdminDeleteUser", msg)
		}
		return errors.Errorf(errors.Internal, "SupabaseClient.AdminDeleteUser", "IdentityProvider returned status: %d", resp.StatusCode)
	}

	return nil
}

type supabaseAdminUpdateRequest struct {
	Email        string                 `json:"email,omitempty"`
	UserMetadata map[string]interface{} `json:"user_metadata,omitempty"`
}

// AdminUpdateUser invokes GoTrue Admin API to update email and/or user_metadata for a given UID.
// Requires serviceRoleKey. Pass only the fields you want to change inside `updates`.
// Recognized keys: "email", "full_name".
func (s *SupabaseClient) AdminUpdateUser(ctx context.Context, uid string, updates map[string]interface{}) error {
	if s.serviceRoleKey == "" {
		return errors.New(errors.Internal, "SupabaseClient.AdminUpdateUser", "Service role key not configured")
	}

	endpoint := fmt.Sprintf("%s/auth/v1/admin/users/%s", s.baseURL, uid)

	payload := supabaseAdminUpdateRequest{}
	if email, ok := updates["email"].(string); ok && email != "" {
		payload.Email = email
	}

	// Separate out user_metadata fields (everything that is not email)
	meta := map[string]interface{}{}
	for k, v := range updates {
		if k != "email" {
			meta[k] = v
		}
	}
	if len(meta) > 0 {
		payload.UserMetadata = meta
	}

	reqBody, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return errors.New(errors.Internal, "SupabaseClient.AdminUpdateUser", "Failed to build request")
	}

	req.Header.Set("apikey", s.serviceRoleKey)
	req.Header.Set("Authorization", "Bearer "+s.serviceRoleKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return errors.New(errors.Unavailable, "SupabaseClient.AdminUpdateUser", "Failed to reach identity provider")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var supErr supabaseError
		if err := json.NewDecoder(resp.Body).Decode(&supErr); err == nil {
			msg := supErr.ErrorDescription
			if msg == "" {
				msg = supErr.Message
			}
			return errors.New(errors.Internal, "SupabaseClient.AdminUpdateUser", msg)
		}
		return errors.Errorf(errors.Internal, "SupabaseClient.AdminUpdateUser", "IdentityProvider returned status: %d", resp.StatusCode)
	}

	return nil
}
