package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Resume restarts a paused sandbox and returns a fresh handle to it.
//
// POST /sandboxes/{sandboxID}/resume
func (s *Service) Resume(ctx context.Context, sandboxID string, opts ResumeOptions) (*Sandbox, error) {
	timeout := int32(orDefaultInt(opts.TimeoutSeconds, DefaultTimeoutSeconds))
	body := api.PostSandboxesSandboxIDResumeJSONRequestBody{
		Timeout: &timeout,
		Memory:  opts.Memory,
	}

	resp, err := s.t.API().PostSandboxesSandboxIDResumeWithResponse(ctx, sandboxID, body)
	if err != nil {
		return nil, err
	}
	resumed, err := transport.Parsed(resp.JSON201, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}
	return s.newSandbox(sandboxID, *resumed)
}
