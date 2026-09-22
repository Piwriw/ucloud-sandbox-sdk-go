package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Metrics returns a sandbox's resource use over a window. The result is never
// nil.
//
// GET /sandboxes/{sandboxID}/metrics
func (s *Service) Metrics(ctx context.Context, sandboxID string, opts MetricsOptions) ([]Metrics, error) {
	params := &api.GetSandboxesSandboxIDMetricsParams{}
	if opts.StartUnix > 0 {
		params.Start = &opts.StartUnix
	}
	if opts.EndUnix > 0 {
		params.End = &opts.EndUnix
	}

	resp, err := s.t.API().GetSandboxesSandboxIDMetricsWithResponse(ctx, sandboxID, params)
	if err != nil {
		return nil, err
	}
	page, err := transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}

	samples := make([]Metrics, 0, len(*page))
	for _, m := range *page {
		samples = append(samples, metricsFrom(m))
	}
	return samples, nil
}
