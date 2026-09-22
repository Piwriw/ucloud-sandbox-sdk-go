package secret

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// GetInfo returns a secret's metadata, never its value.
//
// secret is either the identifier ("sec_...") or the secret's name.
//
// GET /secrets/{secretID}
func (s *Service) GetInfo(ctx context.Context, secret string) (*Info, error) {
	resp, err := s.t.API().GetSecretsSecretIDWithResponse(ctx, secret)
	if err != nil {
		return nil, err
	}
	found, err := transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}
	return infoFrom(*found), nil
}
