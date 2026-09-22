package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// BatchMetrics returns the latest sample for several sandboxes at once, keyed
// by sandbox ID.
//
// One call instead of one per sandbox, which is what makes it usable for a
// dashboard over many sandboxes. A sandbox with no sample is absent from the
// result rather than present with a zero value.
//
// GET /sandboxes/metrics
func (s *Service) BatchMetrics(ctx context.Context, sandboxIDs []string) (map[string]Metrics, error) {
	if len(sandboxIDs) == 0 {
		return map[string]Metrics{}, nil
	}

	params := &api.GetSandboxesMetricsParams{SandboxIds: sandboxIDs}

	resp, err := s.t.API().GetSandboxesMetricsWithResponse(ctx, params)
	if err != nil {
		return nil, err
	}
	batch, err := transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}

	metrics := make(map[string]Metrics, len(batch.Sandboxes))
	for sandboxID, m := range batch.Sandboxes {
		metrics[sandboxID] = metricsFrom(m)
	}
	return metrics, nil
}
