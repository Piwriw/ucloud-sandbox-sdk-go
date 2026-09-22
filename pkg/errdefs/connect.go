package errdefs

import (
	"errors"

	"connectrpc.com/connect"
)

// FromConnect turns an error from envd's connect-rpc services into the same
// types FromHTTP produces, so a caller does not have to know which protocol a
// given operation used.
//
// Errors that did not come from connect are returned unchanged: a context
// cancellation or a dial failure is more useful in its original form than
// flattened into a SandboxError.
func FromConnect(err error) error {
	if err == nil {
		return nil
	}

	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return err
	}

	msg := connectErr.Message()
	switch connectErr.Code() {
	case connect.CodeNotFound:
		return &NotFoundError{SandboxError{Message: msg, Cause: err}}
	case connect.CodeAlreadyExists:
		return &ConflictError{SandboxError{Message: msg, Cause: err}}
	case connect.CodeInvalidArgument:
		return &InvalidArgumentError{SandboxError{Message: msg, Cause: err}}
	case connect.CodeUnauthenticated:
		return &AuthenticationError{SandboxError{Message: msg, Cause: err}}
	case connect.CodePermissionDenied:
		return &ForbiddenError{SandboxError{Message: msg, Cause: err}}
	case connect.CodeResourceExhausted:
		return &RateLimitError{SandboxError{Message: msg, Cause: err}}
	case connect.CodeDeadlineExceeded:
		return &TimeoutError{SandboxError{Message: msg, Cause: err}}
	case connect.CodeUnavailable:
		// envd is not answering. Same situation the control plane reports as a
		// 502, so it maps to the same type.
		return &TimeoutError{SandboxError{Message: msg, Cause: err}}
	default:
		return &SandboxError{Message: msg, Cause: err}
	}
}
