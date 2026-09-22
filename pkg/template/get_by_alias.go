package template

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// GetByAlias resolves a template alias to the template it names.
//
// GET /templates/aliases/{alias}
func (s *Service) GetByAlias(ctx context.Context, alias string) (*Alias, error) {
	resp, err := s.t.API().GetTemplatesAliasesAliasWithResponse(ctx, alias)
	if err != nil {
		return nil, err
	}
	resolved, err := transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}
	return &Alias{TemplateID: resolved.TemplateID, Public: resolved.Public}, nil
}
