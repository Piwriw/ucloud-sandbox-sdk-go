package template

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Delete removes a template and all of its builds. A template that does not
// exist is reported as false rather than as an error.
//
// DELETE /templates/{templateID}
func (s *Service) Delete(ctx context.Context, templateID string) (bool, error) {
	resp, err := s.t.API().DeleteTemplatesTemplateIDWithResponse(ctx, templateID)
	if err != nil {
		return false, err
	}
	if transport.IsNotFound(resp.HTTPResponse) {
		return false, nil
	}
	if err := transport.Check(resp.HTTPResponse, resp.Body); err != nil {
		return false, err
	}
	return true, nil
}
