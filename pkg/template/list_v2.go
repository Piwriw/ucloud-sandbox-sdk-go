package template

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// ListV2Options are the optional arguments to ListV2.
type ListV2Options struct {
	// TeamID lists another team's templates. Rarely needed; the API key
	// already identifies a team.
	TeamID string
}

// ListV2 returns every template the team can see. The endpoint is not
// paginated, so this returns a slice. The result is never nil.
//
// GET /v2/templates
func (s *Service) ListV2(ctx context.Context, opts ListV2Options) ([]Info, error) {
	params := &api.GetV2TemplatesParams{}
	if opts.TeamID != "" {
		params.TeamID = &opts.TeamID
	}

	resp, err := s.t.API().GetV2TemplatesWithResponse(ctx, params)
	if err != nil {
		return nil, err
	}
	page, err := transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}

	templates := make([]Info, 0, len(*page))
	for _, t := range *page {
		templates = append(templates, templateInfoFrom(t))
	}
	return templates, nil
}
