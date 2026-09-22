// Package errdefs holds the error types the SDK returns, and the mappings that
// produce them from the two wire protocols in play: HTTP for the control plane
// and envd's files API, connect-rpc for envd's process and filesystem services.
//
// Both mappings land on the same types, so callers classify a failure the same
// way wherever it came from:
//
//	if errors.Is(err, errdefs.ErrNotFound) { ... }
//
//	var exit *errdefs.CommandExitError
//	if errors.As(err, &exit) { log.Print(exit.Stderr) }
package errdefs

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// SandboxError is the base for every error type below. It is also returned on
// its own when nothing more specific fits.
type SandboxError struct {
	Message string
	Cause   error
}

func (e *SandboxError) Error() string { return e.Message }
func (e *SandboxError) Unwrap() error { return e.Cause }

// Sentinels for errors.Is. Each type's Is method matches on type, not value, so
// these compare equal to any error of the same kind regardless of its message.
var (
	ErrTimeout         = &TimeoutError{SandboxError: SandboxError{Message: "timeout"}}
	ErrNotFound        = &NotFoundError{SandboxError: SandboxError{Message: "not found"}}
	ErrAuth            = &AuthenticationError{SandboxError: SandboxError{Message: "authentication failed"}}
	ErrRateLimit       = &RateLimitError{SandboxError: SandboxError{Message: "rate limit exceeded"}}
	ErrInvalidArgument = &InvalidArgumentError{SandboxError: SandboxError{Message: "invalid argument"}}
	ErrConflict        = &ConflictError{SandboxError: SandboxError{Message: "conflict"}}
	ErrForbidden       = &ForbiddenError{SandboxError: SandboxError{Message: "forbidden"}}
	ErrNotEnoughSpace  = &NotEnoughSpaceError{SandboxError: SandboxError{Message: "not enough space"}}
)

type TimeoutError struct{ SandboxError }

func (e *TimeoutError) Is(target error) bool { _, ok := target.(*TimeoutError); return ok }

type NotFoundError struct{ SandboxError }

func (e *NotFoundError) Is(target error) bool { _, ok := target.(*NotFoundError); return ok }

type AuthenticationError struct{ SandboxError }

func (e *AuthenticationError) Is(target error) bool {
	_, ok := target.(*AuthenticationError)
	return ok
}

type InvalidArgumentError struct{ SandboxError }

func (e *InvalidArgumentError) Is(target error) bool {
	_, ok := target.(*InvalidArgumentError)
	return ok
}

type NotEnoughSpaceError struct{ SandboxError }

func (e *NotEnoughSpaceError) Is(target error) bool {
	_, ok := target.(*NotEnoughSpaceError)
	return ok
}

type RateLimitError struct{ SandboxError }

func (e *RateLimitError) Is(target error) bool { _, ok := target.(*RateLimitError); return ok }

type ConflictError struct{ SandboxError }

func (e *ConflictError) Is(target error) bool { _, ok := target.(*ConflictError); return ok }

type ForbiddenError struct{ SandboxError }

func (e *ForbiddenError) Is(target error) bool { _, ok := target.(*ForbiddenError); return ok }

type FileUploadError struct{ SandboxError }

func (e *FileUploadError) Is(target error) bool { _, ok := target.(*FileUploadError); return ok }

type TemplateError struct{ SandboxError }

func (e *TemplateError) Is(target error) bool { _, ok := target.(*TemplateError); return ok }

// BuildError reports a template build that the platform rejected or failed.
type BuildError struct {
	SandboxError
	BuildID    string
	TemplateID string
}

func (e *BuildError) Is(target error) bool { _, ok := target.(*BuildError); return ok }

// CommandExitError reports a command that ran to completion with a non-zero
// exit code. The captured output is carried along, since it is usually the only
// explanation of what went wrong.
type CommandExitError struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Message  string
	Cause    error
}

func (e *CommandExitError) Error() string {
	return fmt.Sprintf("command exited with code %d: %s", e.ExitCode, e.Message)
}
func (e *CommandExitError) Unwrap() error { return e.Cause }

// IsConflict reports whether err is, or wraps, a ConflictError.
func IsConflict(err error) bool {
	var conflictErr *ConflictError
	return errors.As(err, &conflictErr)
}

// IsNotFound reports whether err is, or wraps, a NotFoundError.
func IsNotFound(err error) bool {
	var notFoundErr *NotFoundError
	return errors.As(err, &notFoundErr)
}

// IsTimeout reports whether err represents a timeout, including transport-level
// ones that never reached the mappings below. The string check is a fallback
// for errors that report a deadline only in their message.
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded")
}
