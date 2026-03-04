package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Lovealone1/nex21-api/internal/modules/auth/application"
	"github.com/Lovealone1/nex21-api/internal/platform/errors"
)

// SupabaseClient implements the AuthProvider interface using GoTrue endpoints.
type SupabaseClient struct {
	baseURL string
	anonKey string
	client  *http.Client
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
func NewSupabaseClient(url, anonKey string) *SupabaseClient {
	return &SupabaseClient{
		baseURL: url,
		anonKey: anonKey,
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
