package template

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// DeleteTags removes tags from a template.
//
// DELETE /templates/tags
func (s *Service) DeleteTags(ctx context.Context, name string, tags []string) error {
	body := api.DeleteTemplatesTagsJSONRequestBody{Name: name, Tags: tags}

	resp, err := s.t.API().DeleteTemplatesTagsWithResponse(ctx, body)
	if err != nil {
		return err
	}
	return transport.Check(resp.HTTPResponse, resp.Body)
}
