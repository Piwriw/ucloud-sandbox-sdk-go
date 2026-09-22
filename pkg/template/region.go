package template

import (
	"os"
	"strings"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// defaultRegion is the region New assumes when it is given none.
//
// Service.NewBuilder does not come through here: it knows the client's resolved
// region. This is only for a Builder made without a client, and it follows the
// Python SDK, which reads the same variable and falls back to the same default.
func defaultRegion() string {
	if region := strings.TrimSpace(os.Getenv(transport.EnvRegion)); region != "" {
		return region
	}
	return transport.DefaultRegion
}
