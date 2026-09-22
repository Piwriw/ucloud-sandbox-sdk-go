package secret

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Create stores value as the first version of a new secret.
//
// name must be unique within the project and is lower-cased before storage, so
// the returned Info may carry a different spelling than the one passed in.
// value is write-only: it is never returned by this or any other call.
//
// POST /secrets
func (s *Service) Create(ctx context.Context, name, value string, opts ...CreateOptions) (*Info, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	opt := first(opts)

	body := api.PostSecretsJSONRequestBody{
		Name:     name,
		Value:    value,
		Metadata: metadataFor(opt.Metadata),
	}

	resp, err := s.t.API().PostSecretsWithResponse(ctx, body)
	if err != nil {
		return nil, err
	}
	created, err := transport.Parsed(resp.JSON201, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}
	return infoFrom(*created), nil
}

// metadataFor converts a caller's map into the generated optional type. A nil
// map is left unset so the server keeps its own default, while an empty
// non-nil map is sent through and clears whatever was stored.
func metadataFor(metadata map[string]string) *api.SecretMetadata {
	if metadata == nil {
		return nil
	}
	converted := make(api.SecretMetadata, len(metadata))
	for k, v := range metadata {
		converted[k] = v
	}
	return &converted
}
