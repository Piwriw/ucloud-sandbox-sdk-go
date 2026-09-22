package errdefs

import "fmt"

// FromHTTP turns a non-2xx response into the matching error type. body is the
// raw response body, which becomes the error message: the control plane returns
// a JSON Error object there, and passing it through unchanged keeps whatever
// detail the server chose to send.
//
// Statuses with no specific type map to *SandboxError carrying the status code,
// so an unrecognised failure is still distinguishable from success.
func FromHTTP(statusCode int, body string) error {
	switch statusCode {
	case 400:
		return &InvalidArgumentError{SandboxError{Message: body}}
	case 401:
		return &AuthenticationError{SandboxError{Message: body}}
	case 403:
		return &ForbiddenError{SandboxError{Message: body}}
	case 404:
		return &NotFoundError{SandboxError{Message: body}}
	case 409:
		return &ConflictError{SandboxError{Message: body}}
	case 429:
		return &RateLimitError{SandboxError{Message: body}}
	case 502:
		// The proxy answers 502 when no sandbox is listening behind it, which
		// in practice means the sandbox is not running rather than a gateway
		// fault. Reported as a timeout to match the platform's own wording.
		return &TimeoutError{SandboxError{Message: "sandbox is likely not running"}}
	case 507:
		return &NotEnoughSpaceError{SandboxError{Message: body}}
	default:
		return &SandboxError{Message: fmt.Sprintf("HTTP %d: %s", statusCode, body)}
	}
}
