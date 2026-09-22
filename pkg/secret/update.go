package secret

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Update stores value as the secret's next version and makes it the one served
// to readers that do not name a version. Earlier versions are kept.
//
// secret is either the identifier ("sec_...") or the secret's name. value is
// write-only.
//
// POST /secrets/{secretID}
func (s *Service) Update(ctx context.Context, secret, value string, opts ...UpdateOptions) (*Info, error) {
	opt := first(opts)

	body := api.PostSecretsSecretIDJSONRequestBody{
		Value:    value,
		Metadata: metadataFor(opt.Metadata),
	}

	resp, err := s.t.API().PostSecretsSecretIDWithResponse(ctx, secret, body)
	if err != nil {
		return nil, err
	}
	updated, err := transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}
	return infoFrom(*updated), nil
}
