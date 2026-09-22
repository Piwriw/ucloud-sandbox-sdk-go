package sandbox

import (
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Service is the entry point for sandbox operations. Get one from
// client.Client.Sandboxes, or build it directly on a transport client.
//
// It is safe for concurrent use.
type Service struct {
	t *transport.Client
}

// NewService returns a Service backed by t.
func NewService(t *transport.Client) *Service {
	return &Service{t: t}
}

// newSandbox wraps a control-plane response into a usable handle, building the
// envd clients it needs.
func (s *Service) newSandbox(sandboxID string, sbx api.Sandbox) (*Sandbox, error) {
	domain := s.t.SandboxDomain(valueOr(sbx.Domain, ""))

	handle := &Sandbox{
		ID:          sandboxID,
		Domain:      domain,
		EnvdVersion: sbx.EnvdVersion,

		svc:         s,
		envdVersion: parseEnvdVersion(sbx.EnvdVersion),
	}

	conn, err := newEnvdConn(
		s.t.HTTPClient(),
		s.t.SandboxURL(sandboxID, domain),
		sandboxID,
		valueOr(sbx.EnvdAccessToken, ""),
		valueOr(sbx.TrafficAccessToken, ""),
		s.t.APIKey(),
	)
	if err != nil {
		return nil, err
	}
	handle.conn = conn

	handle.Commands = &Commands{sbx: handle}
	handle.Files = &Filesystem{sbx: handle}
	handle.Pty = &Pty{sbx: handle}

	return handle, nil
}

// valueOr dereferences an optional field, falling back to a default.
func valueOr[T any](ptr *T, fallback T) T {
	if ptr == nil {
		return fallback
	}
	return *ptr
}
