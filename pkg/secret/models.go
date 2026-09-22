// Package secret manages project secrets.
//
// A secret's value is write-only: Create and Update accept one, and no read
// ever returns it. Sandboxes reach the value by referring to the secret by
// name, using the placeholder Fill produces:
//
//	secrets := c.Secrets()
//	if _, err := secrets.Create(ctx, "openai-key", apiKey, secret.CreateOptions{}); err != nil { ... }
//
//	sbx, err := c.Sandboxes().Create(ctx, sandbox.CreateOptions{
//	    EnvVars: map[string]string{
//	        "OPENAI_API_KEY": secret.MustFill("openai-key"),
//	    },
//	})
//
// The runtime substitutes the real value on its way out of the sandbox, so the
// value is never present in the sandbox's own environment.
package secret

import (
	"time"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

// Info is a secret's metadata. It never carries the secret's value.
type Info struct {
	// SecretID identifies the secret, prefixed with "sec_".
	SecretID string

	// Name is unique within the project, stored lower-cased.
	Name string

	// Version is the version served to readers that do not name one. A newly
	// created secret is at version 1, and each Update adds one.
	Version int64

	// Metadata is whatever the caller stored alongside the secret. Always
	// non-nil, empty when nothing was stored.
	Metadata map[string]string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// infoFrom converts a generated Secret into an Info.
func infoFrom(s api.Secret) *Info {
	metadata := make(map[string]string, len(s.Metadata))
	for k, v := range s.Metadata {
		metadata[k] = v
	}
	return &Info{
		SecretID:  s.SecretID,
		Name:      s.Name,
		Version:   s.CurrentVersion,
		Metadata:  metadata,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
