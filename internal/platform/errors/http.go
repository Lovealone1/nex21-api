package errors

import (
	"encoding/json"
	"net/http"
)

// HTTPErrorResponse is the standard response body for API errors.
type HTTPErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// HTTPStatusCode returns the appropriate HTTP status code for a domain error code.
func HTTPStatusCode(code Code) int {
	switch code {
	case InvalidArgument, FailedPrecondition:
		return http.StatusBadRequest
	case NotFound:
		return http.StatusNotFound
	case AlreadyExists, Conflict:
		return http.StatusConflict
	case PermissionDenied:
		return http.StatusForbidden
	case Unauthenticated:
		return http.StatusUnauthorized
	case Unavailable:
		return http.StatusServiceUnavailable
	case Internal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// WriteHTTPError writes a standard JSON error response based on a domain error.
// If err is not a domain Error, it treats it as an Internal error.
func WriteHTTPError(w http.ResponseWriter, err error) {
	code := ErrorCode(err)
	status := HTTPStatusCode(code)
	msg := ErrorMessage(err)

	response := HTTPErrorResponse{
		Error:   string(code),
		Message: msg,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Since we are writing an error response, we ignore json encoding errors if any
	_ = json.NewEncoder(w).Encode(response)
}
