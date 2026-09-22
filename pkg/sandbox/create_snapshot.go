package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// CreateSnapshot captures a sandbox as a template that can be booted later.
//
// POST /sandboxes/{sandboxID}/snapshots
func (s *Service) CreateSnapshot(ctx context.Context, sandboxID string, opts SnapshotOptions) (*SnapshotInfo, error) {
	body := api.PostSandboxesSandboxIDSnapshotsJSONRequestBody{}
	if opts.Name != "" {
		body.Name = &opts.Name
	}

	resp, err := s.t.API().PostSandboxesSandboxIDSnapshotsWithResponse(ctx, sandboxID, body)
	if err != nil {
		return nil, err
	}
	created, err := transport.Parsed(resp.JSON201, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}
	info := snapshotFrom(*created)
	return &info, nil
}
