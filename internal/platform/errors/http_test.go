package errors_test

import (
	"encoding/json"
	stderrs "errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lovealone1/nex21-api/internal/platform/errors"
)

func TestHTTPStatusCode(t *testing.T) {
	tests := []struct {
		name string
		code errors.Code
		want int
	}{
		{"invalid argument", errors.InvalidArgument, http.StatusBadRequest},
		{"not found", errors.NotFound, http.StatusNotFound},
		{"already exists", errors.AlreadyExists, http.StatusConflict},
		{"permission denied", errors.PermissionDenied, http.StatusForbidden},
		{"unauthenticated", errors.Unauthenticated, http.StatusUnauthorized},
		{"unavailable", errors.Unavailable, http.StatusServiceUnavailable},
		{"internal", errors.Internal, http.StatusInternalServerError},
		{"unknown code", errors.Code("unknown"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.HTTPStatusCode(tt.code); got != tt.want {
				t.Errorf("HTTPStatusCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   errors.HTTPErrorResponse
	}{
		{
			name:       "domain error",
			err:        errors.New(errors.NotFound, "op", "User not found"),
			wantStatus: http.StatusNotFound,
			wantBody: errors.HTTPErrorResponse{
				Error:   string(errors.NotFound),
				Message: "User not found",
			},
		},
		{
			name:       "standard error",
			err:        stderrs.New("db connection failed"),
			wantStatus: http.StatusInternalServerError,
			wantBody: errors.HTTPErrorResponse{
				Error:   string(errors.Internal),
				Message: "An internal error occurred.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			errors.WriteHTTPError(w, tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("WriteHTTPError() status = %v, want %v", w.Code, tt.wantStatus)
			}

			var body errors.HTTPErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Fatalf("Failed to decode response body: %v", err)
			}

			if body != tt.wantBody {
				t.Errorf("WriteHTTPError() body = %v, want %v", body, tt.wantBody)
			}
		})
	}
}
