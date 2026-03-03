package errors_test

import (
	stderrs "errors"
	"fmt"
	"testing"

	"github.com/Lovealone1/nex21-api/internal/platform/errors"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *errors.Error
		want string
	}{
		{
			name: "all fields",
			err: &errors.Error{
				Code:    errors.NotFound,
				Message: "Item not found",
				Op:      "UserService.GetUser",
				Err:     stderrs.New("db record missing"),
			},
			want: "UserService.GetUser: not_found: Item not found - db record missing",
		},
		{
			name: "only message",
			err: &errors.Error{
				Message: "Just a message",
			},
			want: "Just a message",
		},
		{
			name: "code and message",
			err: &errors.Error{
				Code:    errors.InvalidArgument,
				Message: "Invalid input",
			},
			want: "invalid_argument: Invalid input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want errors.Code
	}{
		{
			name: "nil error",
			err:  nil,
			want: "",
		},
		{
			name: "standard error",
			err:  stderrs.New("standard error"),
			want: errors.Internal,
		},
		{
			name: "domain error",
			err:  errors.New(errors.NotFound, "op", "msg"),
			want: errors.NotFound,
		},
		{
			name: "wrapped domain error",
			err:  fmt.Errorf("wrap context: %w", errors.New(errors.PermissionDenied, "op", "msg")),
			want: errors.PermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.ErrorCode(tt.err); got != tt.want {
				t.Errorf("ErrorCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil error",
			err:  nil,
			want: "",
		},
		{
			name: "standard error",
			err:  stderrs.New("db error"),
			want: "An internal error occurred.", // should be masked
		},
		{
			name: "domain error",
			err:  errors.New(errors.NotFound, "op", "User not found"),
			want: "User not found",
		},
		{
			name: "wrapped domain error",
			err:  fmt.Errorf("wrap context: %w", errors.New(errors.PermissionDenied, "op", "Not allowed to do this")),
			want: "Not allowed to do this",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.ErrorMessage(tt.err); got != tt.want {
				t.Errorf("ErrorMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}
