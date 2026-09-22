package template

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// GetOptions are the optional arguments to Get.
type GetOptions struct {
	// NextToken starts the builds list from a cursor returned by an earlier
	// call.
	NextToken string

	// Limit caps how many builds come back per page, 1 to 100. Zero lets the
	// platform choose.
	Limit int
}

// Get returns a template together with one page of its builds.
//
// GET /templates/{templateID}
func (s *Service) Get(ctx context.Context, templateID string, opts GetOptions) (*WithBuilds, error) {
	params := &api.GetTemplatesTemplateIDParams{}
	if opts.NextToken != "" {
		params.NextToken = &opts.NextToken
	}
	if opts.Limit > 0 {
		limit := int32(opts.Limit)
		params.Limit = &limit
	}

	resp, err := s.t.API().GetTemplatesTemplateIDWithResponse(ctx, templateID, params)
	if err != nil {
		return nil, err
	}
	found, err := transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}

	return withBuildsFrom(*found, transport.NextTokenFrom(resp.HTTPResponse.Header)), nil
}
