package template

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// ListTags returns every tag pointing at a build of the template. The result is
// never nil.
//
// GET /templates/{templateID}/tags
func (s *Service) ListTags(ctx context.Context, templateID string) ([]Tag, error) {
	resp, err := s.t.API().GetTemplatesTemplateIDTagsWithResponse(ctx, templateID)
	if err != nil {
		return nil, err
	}
	page, err := transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}

	tags := make([]Tag, 0, len(*page))
	for _, t := range *page {
		tags = append(tags, tagFrom(t))
	}
	return tags, nil
}
