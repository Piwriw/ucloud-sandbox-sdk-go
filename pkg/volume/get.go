package volume

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Get returns a volume and the token its content operations need.
//
// GET /volumes/{volumeID}
func (s *Service) Get(ctx context.Context, volumeID string) (*api.VolumeAndToken, error) {
	resp, err := s.t.API().GetVolumesVolumeIDWithResponse(ctx, volumeID)
	if err != nil {
		return nil, err
	}
	return transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
}

// Connect returns a handle to an existing volume, ready for content operations.
func (s *Service) Connect(ctx context.Context, volumeID string) (*Volume, error) {
	found, err := s.Get(ctx, volumeID)
	if err != nil {
		return nil, err
	}
	return s.newVolume(*found), nil
}
