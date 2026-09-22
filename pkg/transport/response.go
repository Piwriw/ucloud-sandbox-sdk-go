package transport

import (
	"fmt"
	"net/http"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

// Check turns a non-2xx response into an error from errdefs.
//
// resp and body are the HTTPResponse and Body fields of a generated
// ...WithResponse result. The generated code parses each documented status into
// its own JSONxxx field, but the SDK reports failures through one error
// hierarchy instead, so the raw body becomes the error message.
func Check(resp *http.Response, body []byte) error {
	if resp == nil {
		return &errdefs.SandboxError{Message: "no response"}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errdefs.FromHTTP(resp.StatusCode, string(body))
	}
	return nil
}

// Parsed checks the response and returns the decoded body.
//
// parsed is the generated JSONxxx field for the success status the caller
// expects. A success status with nothing decoded there means the server
// answered in a shape the spec does not describe, which is reported rather than
// passed on as a nil pointer.
func Parsed[T any](parsed *T, resp *http.Response, body []byte) (*T, error) {
	if err := Check(resp, body); err != nil {
		return nil, err
	}
	if parsed == nil {
		return nil, &errdefs.SandboxError{
			Message: fmt.Sprintf("HTTP %d: response body was empty or did not match the spec", resp.StatusCode),
		}
	}
	return parsed, nil
}

// IsNotFound reports whether a generated response is a 404. Used by the calls
// that report a missing object as a false return rather than an error.
func IsNotFound(resp *http.Response) bool {
	return resp != nil && resp.StatusCode == http.StatusNotFound
}
