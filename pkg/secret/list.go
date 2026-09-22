package secret

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// List returns a paginator over the project's secrets, newest cursor first. No
// request is made until the paginator is walked.
//
//	all, err := c.Secrets().List(ctx, secret.ListOptions{}).All(ctx)
//
// GET /secrets
func (s *Service) List(ctx context.Context, opts ListOptions) *transport.Paginator[Info] {
	return transport.NewPaginator(func(ctx context.Context, token string) ([]Info, string, error) {
		params := &api.GetSecretsParams{}
		if token != "" {
			params.NextToken = &token
		}
		if opts.Limit > 0 {
			limit := api.PaginationLimit(opts.Limit)
			params.Limit = &limit
		}

		resp, err := s.t.API().GetSecretsWithResponse(ctx, params)
		if err != nil {
			return nil, "", err
		}
		page, err := transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
		if err != nil {
			return nil, "", err
		}

		items := make([]Info, 0, len(*page))
		for _, secret := range *page {
			items = append(items, *infoFrom(secret))
		}
		return items, transport.NextTokenFrom(resp.HTTPResponse.Header), nil
	})
}
