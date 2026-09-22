package template

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// ListBuilds returns a paginator over every build of a template. No request is
// made until the paginator is walked.
//
// GET /templates/{templateID}
func (s *Service) ListBuilds(ctx context.Context, templateID string, opts GetOptions) *transport.Paginator[Build] {
	return transport.NewPaginator(func(ctx context.Context, token string) ([]Build, string, error) {
		pageOpts := GetOptions{Limit: opts.Limit, NextToken: token}
		if token == "" {
			pageOpts.NextToken = opts.NextToken
		}

		page, err := s.Get(ctx, templateID, pageOpts)
		if err != nil {
			return nil, "", err
		}
		return page.Builds, page.NextToken, nil
	})
}
