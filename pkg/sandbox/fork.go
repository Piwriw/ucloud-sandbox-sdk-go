package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Fork starts copies of a running sandbox.
//
// All forks boot from one snapshot, which is captured once however many are
// asked for. Each fork then succeeds or fails on its own, so the returned
// slice has one entry per requested fork, each carrying either a Sandbox or an
// error. A failure of one fork is not an error from Fork itself.
//
// POST /sandboxes/{sandboxID}/fork
func (s *Service) Fork(ctx context.Context, sandboxID string, opts ForkOptions) ([]ForkResult, error) {
	body := api.PostSandboxesSandboxIDForkJSONRequestBody{}
	if opts.Count > 0 {
		count := int32(opts.Count)
		body.Count = &count
	}
	timeout := int32(orDefaultInt(opts.TimeoutSeconds, DefaultTimeoutSeconds))
	body.Timeout = &timeout

	resp, err := s.t.API().PostSandboxesSandboxIDForkWithResponse(ctx, sandboxID, body)
	if err != nil {
		return nil, err
	}
	forked, err := transport.Parsed(resp.JSON201, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}

	results := make([]ForkResult, 0, len(*forked))
	for _, result := range *forked {
		results = append(results, s.forkResult(result))
	}
	return results, nil
}

// forkResult turns one entry of the fork response into a usable result.
func (s *Service) forkResult(result api.SandboxForkResult) ForkResult {
	if result.Sandbox == nil {
		message := "fork failed"
		if result.Error != nil {
			message = result.Error.Message
		}
		return ForkResult{Err: &errdefs.SandboxError{Message: message}}
	}

	handle, err := s.newSandbox(result.Sandbox.SandboxID, *result.Sandbox)
	if err != nil {
		return ForkResult{Err: err}
	}
	return ForkResult{Sandbox: handle}
}
