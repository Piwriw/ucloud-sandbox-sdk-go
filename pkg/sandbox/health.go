package sandbox

import (
	"context"
	"net/http"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

// IsRunning reports whether envd inside the sandbox is answering.
//
// A sandbox that is paused or gone answers 502 through the proxy, which is
// reported as false rather than as an error: asking whether something is
// running and learning that it is not is a successful answer.
func (s *Sandbox) IsRunning(ctx context.Context) (bool, error) {
	resp, err := s.conn.files.GetHealth(ctx)
	if err != nil {
		if errdefs.IsTimeout(err) || ctx.Err() != nil {
			return false, &errdefs.TimeoutError{SandboxError: errdefs.SandboxError{
				Message: "health check timed out", Cause: err,
			}}
		}
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadGateway {
		return false, nil
	}
	return true, nil
}

// Health checks that the control plane is reachable and healthy.
//
// GET /health
func (s *Service) Health(ctx context.Context) error {
	resp, err := s.t.API().GetHealthWithResponse(ctx)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode >= 200 && resp.HTTPResponse.StatusCode < 300 {
		return nil
	}
	return errdefs.FromHTTP(resp.HTTPResponse.StatusCode, string(resp.Body))
}
