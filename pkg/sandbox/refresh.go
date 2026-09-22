package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Refresh keeps a sandbox alive for another durationSeconds, up to an hour.
//
// Unlike SetTimeout, this is meant to be called repeatedly while work is in
// progress: it extends the sandbox rather than resetting it to a fixed life.
//
// POST /sandboxes/{sandboxID}/refreshes
func (s *Service) Refresh(ctx context.Context, sandboxID string, durationSeconds int) error {
	body := api.PostSandboxesSandboxIDRefreshesJSONRequestBody{}
	if durationSeconds > 0 {
		body.Duration = &durationSeconds
	}

	resp, err := s.t.API().PostSandboxesSandboxIDRefreshesWithResponse(ctx, sandboxID, body)
	if err != nil {
		return err
	}
	return transport.Check(resp.HTTPResponse, resp.Body)
}
