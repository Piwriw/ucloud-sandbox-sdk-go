package template

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// AssignTags points tags at a build.
//
// target names the build, as a template name optionally carrying a tag, for
// example "my-template" or "my-template:v1".
//
// POST /templates/tags
func (s *Service) AssignTags(ctx context.Context, target string, tags []string) (*AssignedTags, error) {
	body := api.PostTemplatesTagsJSONRequestBody{Target: target, Tags: tags}

	resp, err := s.t.API().PostTemplatesTagsWithResponse(ctx, body)
	if err != nil {
		return nil, err
	}
	assigned, err := transport.Parsed(resp.JSON201, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}
	return &AssignedTags{BuildID: assigned.BuildID.String(), Tags: assigned.Tags}, nil
}
