package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// ListSnapshots returns a paginator over the team's snapshots. No request is
// made until the paginator is walked.
//
// GET /snapshots
func (s *Service) ListSnapshots(ctx context.Context, opts ListSnapshotsOptions) *transport.Paginator[SnapshotInfo] {
	return transport.NewPaginator(func(ctx context.Context, token string) ([]SnapshotInfo, string, error) {
		params := &api.GetSnapshotsParams{}
		if token != "" {
			params.NextToken = &token
		}
		if opts.Limit > 0 {
			limit := int32(opts.Limit)
			params.Limit = &limit
		}
		if opts.SandboxID != "" {
			params.SandboxID = &opts.SandboxID
		}
		if opts.Name != "" {
			params.Name = &opts.Name
		}

		resp, err := s.t.API().GetSnapshotsWithResponse(ctx, params)
		if err != nil {
			return nil, "", err
		}
		page, err := transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
		if err != nil {
			return nil, "", err
		}

		snapshots := make([]SnapshotInfo, 0, len(*page))
		for _, snapshot := range *page {
			snapshots = append(snapshots, snapshotFrom(snapshot))
		}
		return snapshots, transport.NextTokenFrom(resp.HTTPResponse.Header), nil
	})
}
