package sandbox

import (
	"context"
	"net/url"
	"strings"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// ListV2 returns a paginator over the team's sandboxes. No request is made
// until the paginator is walked.
//
//	p := c.Sandboxes().ListV2(ctx, sandbox.ListV2Options{State: []sandbox.State{"running"}})
//	all, err := p.All(ctx)
//
// GET /v2/sandboxes
func (s *Service) ListV2(ctx context.Context, opts ListV2Options) *transport.Paginator[Info] {
	return transport.NewPaginator(func(ctx context.Context, token string) ([]Info, string, error) {
		params := &api.GetV2SandboxesParams{}
		if token != "" {
			params.NextToken = &token
		}
		if opts.Limit > 0 {
			limit := int32(opts.Limit)
			params.Limit = &limit
		}
		if query := metadataQuery(opts.Metadata); query != "" {
			params.Metadata = &query
		}
		if len(opts.State) > 0 {
			state := opts.State
			params.State = &state
		}
		if opts.Template != "" {
			params.Template = &opts.Template
		}
		if !opts.StartedAfter.IsZero() {
			params.StartedAfter = &opts.StartedAfter
		}
		if opts.OrderDescending {
			order := api.OrderDirection("desc")
			params.Order = &order
		}

		resp, err := s.t.API().GetV2SandboxesWithResponse(ctx, params)
		if err != nil {
			return nil, "", err
		}
		page, err := transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
		if err != nil {
			return nil, "", err
		}

		sandboxes := make([]Info, 0, len(*page))
		for _, listed := range *page {
			sandboxes = append(sandboxes, infoFromListed(listed))
		}
		return sandboxes, transport.NextTokenFrom(resp.HTTPResponse.Header), nil
	})
}

// metadataQuery renders a metadata filter as the "k=v&k=v" string the endpoint
// expects, with each key and value URL-encoded.
func metadataQuery(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(metadata))
	for key, value := range metadata {
		pairs = append(pairs, url.QueryEscape(key)+"="+url.QueryEscape(value))
	}
	return strings.Join(pairs, "&")
}
