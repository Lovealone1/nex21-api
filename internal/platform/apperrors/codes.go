package errors

// Code defines the standardized error code.
type Code string

const (
	// InvalidArgument indicates client specified an invalid argument.
	InvalidArgument Code = "invalid_argument"

	// NotFound indicates some requested entity (e.g., file or directory) was not found.
	NotFound Code = "not_found"

	// AlreadyExists indicates an attempt to create an entity failed because one already exists.
	AlreadyExists Code = "already_exists"

	// Conflict indicates a concurrency conflict or other state conflict (e.g. optimistic lock failure).
	Conflict Code = "conflict"

	// PermissionDenied indicates the caller does not have permission to execute the specified operation.
	PermissionDenied Code = "permission_denied"

	// Unauthenticated indicates the request does not have valid authentication credentials for the operation.
	Unauthenticated Code = "unauthenticated"

	// Internal errors. Means some invariants expected by underlying system has been broken.
	Internal Code = "internal"

	// Unavailable indicates the service is currently unavailable.
	Unavailable Code = "unavailable"

	// FailedPrecondition indicates operation was rejected because the system is not in a state required for the operation's execution.
	FailedPrecondition Code = "failed_precondition"
)
