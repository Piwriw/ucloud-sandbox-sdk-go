package transport

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

// DefaultRetries is how many times a 429 carrying a usable Retry-After is
// retried when Config.Retries is nil.
const DefaultRetries = 3

// Config configures a Client. The zero value is usable: every field falls back
// to an environment variable and then to a default, except APIKey, which must
// resolve to something non-empty.
//
// Resolution order, highest first:
//
//	APIKey    Config.APIKey, UCLOUD_SANDBOX_API_KEY
//	domain    Config.Domain, Config.Region, UCLOUD_SANDBOX_REGION,
//	          UCLOUD_SANDBOX_DOMAIN, cn-wlcb.sandbox.ucloudai.com
//	API URL   Config.APIURL, UCLOUD_SANDBOX_API_URL, {scheme}://api.{domain}
//	scheme    http when InsecureHTTP or Debug is set, https otherwise
type Config struct {
	// APIKey authenticates every control-plane request, sent as X-API-Key.
	APIKey string

	// Region is expanded to {Region}.sandbox.ucloudai.com. Ignored when Domain
	// is set. It also selects the default template base image; see
	// pkg/template.
	Region string

	// Domain is the full sandbox domain, overriding Region.
	Domain string

	// APIURL is the full control-plane base URL, overriding Domain and Region.
	// Set it to reach a private deployment by address, for example
	// "http://10.10.0.5:8080".
	APIURL string

	// VolumeAPIURL is the base URL for volume content operations. Defaults to
	// the control-plane URL.
	VolumeAPIURL string

	// SandboxURL pins the URL used to reach sandboxes, instead of deriving one
	// per sandbox from its ID and domain.
	SandboxURL string

	// Headers are added to every control-plane and volume request.
	Headers map[string]string

	// RequestTimeout bounds a single control-plane request. Zero means the
	// client default of five minutes. This is a local deadline and is never
	// sent to the server; sandbox lifetimes are set per call in seconds.
	RequestTimeout time.Duration

	// Retries is how many times to retry a 429 that carries a usable
	// Retry-After header. nil means DefaultRetries; a pointer to 0 disables
	// retrying. Nothing else is retried, since the control plane's other
	// failures are not safe to repeat blindly.
	Retries *int

	// HTTPClient replaces the client the SDK would otherwise build. When set,
	// InsecureSkipTLS and RequestTimeout are left for the caller to configure.
	HTTPClient *http.Client

	// InsecureHTTP talks plain HTTP instead of HTTPS. For private deployments
	// without TLS; do not enable it across untrusted networks.
	InsecureHTTP bool

	// InsecureSkipTLS disables TLS certificate verification. Defaults to false;
	// earlier versions of this SDK disabled verification unconditionally.
	InsecureSkipTLS bool

	// Debug routes sandbox traffic to localhost, for running envd locally.
	Debug bool
}

// resolved is a Config with every fallback applied, computed once by New.
type resolved struct {
	apiKey          string
	region          string
	domain          string
	apiURL          string
	volumeAPIURL    string
	sandboxURL      string
	headers         map[string]string
	requestTimeout  time.Duration
	retries         int
	insecureHTTP    bool
	insecureSkipTLS bool
	debug           bool
}

// resolve applies the environment and default fallbacks documented on Config.
func (c Config) resolve() (resolved, error) {
	r := resolved{
		apiKey:          firstNonEmpty(c.APIKey, os.Getenv(EnvAPIKey)),
		sandboxURL:      firstNonEmpty(c.SandboxURL, os.Getenv(EnvSandboxURL)),
		headers:         c.Headers,
		requestTimeout:  c.RequestTimeout,
		insecureSkipTLS: c.InsecureSkipTLS,
		debug:           c.Debug || envBool(EnvDebug),
		insecureHTTP:    c.InsecureHTTP || envBool(EnvInsecureHTTP),
	}

	if r.apiKey == "" {
		return resolved{}, &errdefs.InvalidArgumentError{SandboxError: errdefs.SandboxError{
			Message: "API key is required: set Config.APIKey or the " + EnvAPIKey + " environment variable",
		}}
	}

	r.retries = DefaultRetries
	if c.Retries != nil {
		if *c.Retries < 0 {
			return resolved{}, &errdefs.InvalidArgumentError{SandboxError: errdefs.SandboxError{
				Message: "Retries must not be negative",
			}}
		}
		r.retries = *c.Retries
	}

	r.region, r.domain = resolveRegionDomain(c.Region, c.Domain)

	r.apiURL = firstNonEmpty(c.APIURL, os.Getenv(EnvAPIURL))
	if r.apiURL == "" {
		r.apiURL = r.scheme() + "://api." + r.domain
	}
	r.apiURL = strings.TrimRight(r.apiURL, "/")

	r.volumeAPIURL = strings.TrimRight(firstNonEmpty(c.VolumeAPIURL, r.apiURL), "/")

	return r, nil
}

// resolveRegionDomain settles on a region and a domain together, so the two
// never disagree.
//
// A caller that names only a domain still needs a region, because the region
// picks the default template base image. When the domain is a standard sandbox
// domain the region is read back out of it; otherwise — a private deployment,
// say — the region falls back as if no domain had been given.
func resolveRegionDomain(region, domain string) (string, string) {
	region = strings.TrimSpace(region)
	domain = strings.TrimSpace(domain)

	if domain != "" {
		if r := regionFromDomain(domain); r != "" {
			return r, domain
		}
		return fallbackRegion(region), domain
	}

	if region != "" {
		return region, domainForRegion(region)
	}

	if envRegion := strings.TrimSpace(os.Getenv(EnvRegion)); envRegion != "" {
		return envRegion, domainForRegion(envRegion)
	}

	if envDomain := strings.TrimSpace(os.Getenv(EnvDomain)); envDomain != "" {
		if r := regionFromDomain(envDomain); r != "" {
			return r, envDomain
		}
		return DefaultRegion, envDomain
	}

	return DefaultRegion, DefaultDomain
}

func fallbackRegion(region string) string {
	if region != "" {
		return region
	}
	if envRegion := strings.TrimSpace(os.Getenv(EnvRegion)); envRegion != "" {
		return envRegion
	}
	return DefaultRegion
}

// scheme is http when talking to a plain-HTTP deployment or to a local envd.
func (r resolved) scheme() string {
	if r.debug || r.insecureHTTP {
		return "http"
	}
	return "https"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// envBool reads a boolean environment variable, treating anything
// strconv.ParseBool rejects as unset.
func envBool(name string) bool {
	v, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(name)))
	return err == nil && v
}
