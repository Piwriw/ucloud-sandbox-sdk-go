package volume

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Create makes a new volume and returns a handle carrying its content token.
//
// POST /volumes
func (s *Service) Create(ctx context.Context, name string) (*Volume, error) {
	resp, err := s.t.API().PostVolumesWithResponse(ctx, api.PostVolumesJSONRequestBody{Name: name})
	if err != nil {
		return nil, err
	}
	created, err := transport.Parsed(resp.JSON201, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}
	return s.newVolume(*created), nil
}
