package template

import "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"

// Service is the entry point for template operations. Get one from
// client.Client.Templates, or build it directly on a transport client.
//
// It is safe for concurrent use.
type Service struct {
	t *transport.Client
}

// NewService returns a Service backed by t.
func NewService(t *transport.Client) *Service {
	return &Service{t: t}
}

// NewBuilder returns a Builder that defaults to the base image for the client's
// region. Prefer it over New, which has to guess the region.
func (s *Service) NewBuilder(opts BuilderOptions) *Builder {
	if opts.Region == "" {
		opts.Region = s.t.Region()
	}
	return New(opts)
}
