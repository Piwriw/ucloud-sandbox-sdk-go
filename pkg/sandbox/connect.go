package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Connect returns a handle to an existing sandbox, extending its life by
// TimeoutSeconds. A paused sandbox is resumed.
//
// POST /sandboxes/{sandboxID}/connect
func (s *Service) Connect(ctx context.Context, sandboxID string, opts ConnectOptions) (*Sandbox, error) {
	body := api.PostSandboxesSandboxIDConnectJSONRequestBody{
		Timeout: int32(orDefaultInt(opts.TimeoutSeconds, DefaultTimeoutSeconds)),
		Memory:  opts.Memory,
	}

	resp, err := s.t.API().PostSandboxesSandboxIDConnectWithResponse(ctx, sandboxID, body)
	if err != nil {
		return nil, err
	}
	if err := transport.Check(resp.HTTPResponse, resp.Body); err != nil {
		return nil, err
	}

	// The endpoint answers 200 for a running sandbox and 201 for one it had to
	// resume; both carry the same body.
	connected := resp.JSON200
	if connected == nil {
		connected = resp.JSON201
	}
	if connected == nil {
		return nil, transport.Check(resp.HTTPResponse, resp.Body)
	}
	return s.newSandbox(sandboxID, *connected)
}
