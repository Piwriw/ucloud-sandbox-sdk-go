package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Get returns a sandbox's current state.
//
// GET /sandboxes/{sandboxID}
func (s *Service) Get(ctx context.Context, sandboxID string) (*Info, error) {
	resp, err := s.t.API().GetSandboxesSandboxIDWithResponse(ctx, sandboxID)
	if err != nil {
		return nil, err
	}
	detail, err := transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}
	return infoFromDetail(*detail), nil
}
