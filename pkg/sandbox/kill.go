package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Kill shuts a sandbox down. A sandbox that is already gone is reported as
// false rather than as an error, so killing twice is not a failure.
//
// DELETE /sandboxes/{sandboxID}
func (s *Service) Kill(ctx context.Context, sandboxID string) (bool, error) {
	resp, err := s.t.API().DeleteSandboxesSandboxIDWithResponse(ctx, sandboxID)
	if err != nil {
		return false, err
	}
	if transport.IsNotFound(resp.HTTPResponse) {
		return false, nil
	}
	if err := transport.Check(resp.HTTPResponse, resp.Body); err != nil {
		return false, err
	}
	return true, nil
}
