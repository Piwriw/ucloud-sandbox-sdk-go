package volume

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// List returns every volume the team owns.
//
// The endpoint is not paginated, so this returns a slice rather than a
// paginator. The result is never nil.
//
// GET /volumes
func (s *Service) List(ctx context.Context) ([]Info, error) {
	resp, err := s.t.API().GetVolumesWithResponse(ctx)
	if err != nil {
		return nil, err
	}
	page, err := transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}

	volumes := make([]Info, 0, len(*page))
	for _, v := range *page {
		volumes = append(volumes, infoFrom(v))
	}
	return volumes, nil
}
