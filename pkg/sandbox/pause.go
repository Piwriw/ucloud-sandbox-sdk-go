package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Pause stops a sandbox and keeps its state so it can be resumed later.
//
// A sandbox that is already paused is not an error: the platform answers 409,
// which is reported as success because the caller's intent is satisfied.
//
// POST /sandboxes/{sandboxID}/pause
func (s *Service) Pause(ctx context.Context, sandboxID string, opts PauseOptions) error {
	body := api.PostSandboxesSandboxIDPauseJSONRequestBody{Memory: opts.Memory}

	resp, err := s.t.API().PostSandboxesSandboxIDPauseWithResponse(ctx, sandboxID, body)
	if err != nil {
		return err
	}
	if err := transport.Check(resp.HTTPResponse, resp.Body); err != nil {
		if errdefs.IsConflict(err) {
			return nil
		}
		return err
	}
	return nil
}
