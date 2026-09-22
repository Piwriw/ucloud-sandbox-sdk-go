package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// UpdateNetwork replaces a running sandbox's egress policy. Fields left nil in
// update are cleared, not kept.
//
// PUT /sandboxes/{sandboxID}/network
func (s *Service) UpdateNetwork(ctx context.Context, sandboxID string, update NetworkUpdate) error {
	resp, err := s.t.API().PutSandboxesSandboxIDNetworkWithResponse(ctx, sandboxID, update.toAPI())
	if err != nil {
		return err
	}
	return transport.Check(resp.HTTPResponse, resp.Body)
}
