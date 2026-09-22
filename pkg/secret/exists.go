package secret

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

// Exists reports whether a secret exists. secret is either the identifier
// ("sec_...") or the secret's name.
//
// Errors other than "not found" are returned, so a network failure is not
// mistaken for a missing secret.
func (s *Service) Exists(ctx context.Context, secret string) (bool, error) {
	if _, err := s.GetInfo(ctx, secret); err != nil {
		if errdefs.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
