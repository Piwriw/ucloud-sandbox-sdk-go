package sandbox

import (
	"errors"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

// assertNotFound is the error a fake handler wraps in a connect code; the code
// is what matters, not the message.
var assertNotFound = errors.New("not found")

// apiSandbox is the control-plane response newSandbox consumes.
func apiSandbox(envdVersion string) api.Sandbox {
	return api.Sandbox{
		SandboxID:   "sbx-1",
		TemplateID:  "base",
		ClientID:    "client-1",
		EnvdVersion: envdVersion,
	}
}
