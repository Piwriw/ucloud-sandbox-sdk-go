package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// DeleteSnapshot removes a snapshot and every build of it. A snapshot that does
// not exist is reported as false rather than as an error, so deleting twice is
// not a failure.
//
// A snapshot is a template, which is why this reaches the template endpoint.
// template.Service.Delete does the same thing; this exists because snapshots
// are made from sandboxes, and looking for the call next to CreateSnapshot is
// the natural place.
//
// DELETE /templates/{templateID}
func (s *Service) DeleteSnapshot(ctx context.Context, snapshotID string) (bool, error) {
	resp, err := s.t.API().DeleteTemplatesTemplateIDWithResponse(ctx, snapshotID)
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
