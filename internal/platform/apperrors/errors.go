package errors

import (
	"bytes"
	stderrs "errors"
	"fmt"
)

// Error defines a standard application error.
type Error struct {
	// Code is a machine-readable error code.
	Code Code
	// Message is a human-readable message.
	Message string
	// Err is the underlying error.
	Err error
	// Op is the logical operation during which the error occurred.
	Op string
}

// Error implements the standard error interface.
func (e *Error) Error() string {
	var buf bytes.Buffer

	if e.Op != "" {
		buf.WriteString(e.Op)
		buf.WriteString(": ")
	}

	if e.Code != "" {
		buf.WriteString(string(e.Code))
		buf.WriteString(": ")
	}

	if e.Message != "" {
		buf.WriteString(e.Message)
	}

	if e.Err != nil {
		if buf.Len() > 0 {
			buf.WriteString(" - ")
		}
		buf.WriteString(e.Err.Error())
	}

	return buf.String()
}

// Unwrap returns the wrapped error, if any, for errors.Is and errors.As.
func (e *Error) Unwrap() error {
	return e.Err
}

// Is reports whether any error in err's tree matches target.
// It delegates to the standard library errors.Is.
func Is(err, target error) bool {
	return stderrs.Is(err, target)
}

// As finds the first error in err's tree that matches target, and if one is found, sets
// target to that error value and returns true. Otherwise, it returns false.
// It delegates to the standard library errors.As.
func As(err error, target any) bool {
	return stderrs.As(err, target)
}

// Unwrap returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning error.
// It delegates to the standard library errors.Unwrap.
func Unwrap(err error) error {
	return stderrs.Unwrap(err)
}

// New creates a new application error with the given code and message.
func New(code Code, op, message string) *Error {
	return &Error{
		Code:    code,
		Op:      op,
		Message: message,
	}
}

// Errorf creates a new Error with formatting for the message.
func Errorf(code Code, op, format string, args ...any) *Error {
	return &Error{
		Code:    code,
		Op:      op,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap wraps an existing error with application context.
func Wrap(err error, code Code, op, message string) *Error {
	if err == nil {
		return nil
	}
	return &Error{
		Code:    code,
		Op:      op,
		Message: message,
		Err:     err,
	}
}

// Wrapf wraps an existing error with formatted application context.
func Wrapf(err error, code Code, op, format string, args ...any) *Error {
	if err == nil {
		return nil
	}
	return &Error{
		Code:    code,
		Op:      op,
		Message: fmt.Sprintf(format, args...),
		Err:     err,
	}
}

// ErrorCode unwraps an error and returns its standard code.
// If the error is not an *Error or does not have a code, it returns Internal.
func ErrorCode(err error) Code {
	if err == nil {
		return ""
	}

	var e *Error
	if As(err, &e) && e.Code != "" {
		return e.Code
	}

	return Internal
}

// ErrorMessage unwraps an error and returns its message.
// For domain errors it returns the human-readable message; if there is a root
// cause error it is appended so the actual failure is surfaced.
func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	var e *Error
	if As(err, &e) && e.Message != "" {
		if e.Err != nil {
			return fmt.Sprintf("%s: %v", e.Message, e.Err)
		}
		return e.Message
	}

	return "An internal error occurred."
}
