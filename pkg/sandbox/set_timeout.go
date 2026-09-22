package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// SetTimeout resets how long a sandbox will live, counted from now rather than
// added to what is left.
//
// POST /sandboxes/{sandboxID}/timeout
func (s *Service) SetTimeout(ctx context.Context, sandboxID string, timeoutSeconds int) error {
	body := api.PostSandboxesSandboxIDTimeoutJSONRequestBody{Timeout: int32(timeoutSeconds)}

	resp, err := s.t.API().PostSandboxesSandboxIDTimeoutWithResponse(ctx, sandboxID, body)
	if err != nil {
		return err
	}
	return transport.Check(resp.HTTPResponse, resp.Body)
}
