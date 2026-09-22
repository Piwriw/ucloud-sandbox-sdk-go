package sandbox

import (
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

// errdefsFromConnect maps an envd RPC failure onto the SDK's error types.
func errdefsFromConnect(err error) error { return errdefs.FromConnect(err) }

// errStreamEndedEarly reports a process stream that closed without saying how
// the process ended.
func errStreamEndedEarly() error {
	return &errdefs.SandboxError{
		Message: "process stream ended without reporting an exit status",
	}
}
