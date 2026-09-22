package template

import "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"

// The platform's own base images. A template that names no base starts from one
// of these.
//
// Which one depends on the region: the CN image is served from a registry
// reachable inside mainland China, and pulling the other one from there is slow
// at best. These values and the choice between them match the Python SDK's
// ucloud_sandbox/domain_config.py.
const (
	// CNBaseImage is the base image for regions in mainland China.
	CNBaseImage = "uhub.service.ucloud.cn/agent-sandbox-public/base_cn:latest"

	// DefaultBaseImage is the base image for every other region.
	DefaultBaseImage = "uhub.service.ucloud.cn/agent-sandbox-public/base:latest"
)

// BaseImageForRegion returns the base image a template should start from in
// region.
func BaseImageForRegion(region string) string {
	if transport.IsRegionCN(region) {
		return CNBaseImage
	}
	return DefaultBaseImage
}
