package secret

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Delete revokes a secret and schedules every version of it for cleanup.
//
// secret is either the identifier ("sec_...") or the secret's name. A secret
// that does not exist is reported as false rather than as an error, so deleting
// twice is not a failure.
//
// DELETE /secrets/{secretID}
func (s *Service) Delete(ctx context.Context, secret string) (bool, error) {
	resp, err := s.t.API().DeleteSecretsSecretIDWithResponse(ctx, secret)
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
