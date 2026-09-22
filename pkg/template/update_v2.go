package template

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// UpdateV2 changes a template's visibility and returns its names.
//
// PATCH /v2/templates/{templateID}
func (s *Service) UpdateV2(ctx context.Context, templateID string, public bool) ([]string, error) {
	body := api.PatchV2TemplatesTemplateIDJSONRequestBody{Public: &public}

	resp, err := s.t.API().PatchV2TemplatesTemplateIDWithResponse(ctx, templateID, body)
	if err != nil {
		return nil, err
	}
	updated, err := transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}
	return updated.Names, nil
}
