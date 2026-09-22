package transport

import "strings"

// Defaults and environment variables, kept in step with the Python SDK's
// ucloud_sandbox/domain_config.py so the two behave the same when neither is
// configured explicitly.
const (
	// DefaultRegion is the region used when none is configured.
	DefaultRegion = "cn-wlcb"

	// DomainSuffix is appended to a region to form a sandbox domain.
	DomainSuffix = "sandbox.ucloudai.com"

	// DefaultDomain is DefaultRegion joined to DomainSuffix.
	DefaultDomain = DefaultRegion + "." + DomainSuffix
)

// Environment variables read when the corresponding Config field is empty.
const (
	EnvAPIKey       = "UCLOUD_SANDBOX_API_KEY"
	EnvRegion       = "UCLOUD_SANDBOX_REGION"
	EnvDomain       = "UCLOUD_SANDBOX_DOMAIN"
	EnvAPIURL       = "UCLOUD_SANDBOX_API_URL"
	EnvSandboxURL   = "UCLOUD_SANDBOX_URL"
	EnvDebug        = "UCLOUD_SANDBOX_DEBUG"
	EnvInsecureHTTP = "UCLOUD_SANDBOX_INSECURE_HTTP"
)

// cnRegionPrefixes are the region prefixes served from mainland China, which
// need images pulled from a registry reachable there. Mirrors the Python SDK's
// CN_REGION_PREFIXES.
var cnRegionPrefixes = []string{"cn", "gray"}

// IsRegionCN reports whether region is served from mainland China. Callers use
// it to pick between the CN and the default base image; see pkg/template.
func IsRegionCN(region string) bool {
	r := strings.ToLower(strings.TrimSpace(region))
	for _, prefix := range cnRegionPrefixes {
		if strings.HasPrefix(r, prefix) {
			return true
		}
	}
	return false
}

// domainForRegion expands a region into its sandbox domain.
func domainForRegion(region string) string {
	return region + "." + DomainSuffix
}

// regionFromDomain recovers the region from a sandbox domain, returning "" if
// domain does not look like one. Used to keep the region consistent when a
// caller configures Domain but not Region.
func regionFromDomain(domain string) string {
	suffix := "." + DomainSuffix
	if !strings.HasSuffix(domain, suffix) {
		return ""
	}
	region := strings.TrimSuffix(domain, suffix)
	if region == "" || strings.Contains(region, ".") {
		return ""
	}
	return region
}
