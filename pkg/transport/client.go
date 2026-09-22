// Package transport carries the SDK's HTTP plumbing: endpoint resolution,
// authentication, retries and pagination. It holds no domain knowledge.
//
// Domain packages (pkg/sandbox, pkg/template, pkg/volume, pkg/secret) build on
// a *transport.Client, and pkg/client assembles them into a single entry point.
// The split exists so that pkg/client can import the domain packages without
// them importing it back.
//
// Most callers construct this indirectly:
//
//	c, err := client.New(client.Options{APIKey: key})
package transport

import (
	"context"
	"crypto/tls"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

// defaultRequestTimeout bounds a control-plane request when Config leaves
// RequestTimeout unset. Generous, because template builds and file transfers
// share this client.
const defaultRequestTimeout = 5 * time.Minute

// EnvdPort is the port envd listens on inside a sandbox.
const EnvdPort = 49983

// Client holds the resolved configuration and the HTTP machinery shared by
// every domain package. It is safe for concurrent use.
type Client struct {
	cfg  resolved
	http *http.Client
	api  *api.ClientWithResponses
}

// New resolves cfg and builds a Client. It fails when no API key can be found,
// or when the generated client rejects the resolved base URL.
func New(cfg Config) (*Client, error) {
	r, err := cfg.resolve()
	if err != nil {
		return nil, err
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = newHTTPClient(r)
	}

	c := &Client{cfg: r, http: httpClient}

	apiClient, err := api.NewClientWithResponses(
		r.apiURL,
		api.WithHTTPClient(httpClient),
		api.WithRequestEditorFn(c.authorize),
	)
	if err != nil {
		return nil, err
	}
	c.api = apiClient

	return c, nil
}

// newHTTPClient builds the client used for every request the SDK makes.
//
// Compression is disabled because envd streams responses, and a compressed
// stream is buffered by the decompressor rather than delivered as it arrives.
func newHTTPClient(r resolved) *http.Client {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		base = &http.Transport{Proxy: http.ProxyFromEnvironment}
	}
	tr := base.Clone()
	tr.Proxy = http.ProxyFromEnvironment
	tr.DisableCompression = true

	if r.insecureSkipTLS {
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = &tls.Config{}
		} else {
			tr.TLSClientConfig = tr.TLSClientConfig.Clone()
		}
		tr.TLSClientConfig.InsecureSkipVerify = true //nolint:gosec // opt-in via Config.InsecureSkipTLS
	}

	var rt http.RoundTripper = tr
	if r.retries > 0 {
		rt = &retryTransport{base: tr, retries: r.retries}
	}

	timeout := r.requestTimeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	return &http.Client{Timeout: timeout, Transport: rt}
}

// authorize adds the API key and any configured headers. Registered with the
// generated client, and applied by hand in Do for requests outside the spec.
func (c *Client) authorize(_ context.Context, req *http.Request) error {
	for k, v := range c.cfg.headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("X-API-Key", c.cfg.apiKey)
	return nil
}

// API returns the generated control-plane client. Domain packages call it, and
// it is exposed so that endpoints this SDK does not wrap — teams, API keys,
// admin — remain reachable.
func (c *Client) API() *api.ClientWithResponses { return c.api }

// HTTPClient returns the shared HTTP client, for the surfaces that are not
// described by the control-plane spec, namely envd.
func (c *Client) HTTPClient() *http.Client { return c.http }

// APIKey returns the resolved API key.
func (c *Client) APIKey() string { return c.cfg.apiKey }

// Region returns the resolved region, for example "cn-wlcb".
func (c *Client) Region() string { return c.cfg.region }

// Domain returns the resolved sandbox domain.
func (c *Client) Domain() string { return c.cfg.domain }

// APIURL returns the resolved control-plane base URL, without a trailing slash.
func (c *Client) APIURL() string { return c.cfg.apiURL }

// Headers returns the extra headers added to every request. The returned map
// must not be modified.
func (c *Client) Headers() map[string]string { return c.cfg.headers }

// Debug reports whether sandbox traffic is routed to localhost.
func (c *Client) Debug() bool { return c.cfg.debug }

// RequestTimeout returns the configured per-request timeout, or zero if the
// client default applies.
func (c *Client) RequestTimeout() time.Duration { return c.cfg.requestTimeout }

// SandboxHost returns the host:port reaching a port inside a sandbox.
func (c *Client) SandboxHost(sandboxID, sandboxDomain string, port int) string {
	if c.cfg.debug {
		return "localhost:" + strconv.Itoa(port)
	}
	return strconv.Itoa(port) + "-" + sandboxID + "." + c.sandboxDomain(sandboxDomain)
}

// SandboxURL returns the base URL of a sandbox's envd.
func (c *Client) SandboxURL(sandboxID, sandboxDomain string) string {
	if c.cfg.sandboxURL != "" {
		return c.cfg.sandboxURL
	}
	return c.cfg.scheme() + "://" + c.SandboxHost(sandboxID, sandboxDomain, EnvdPort)
}

// SandboxScheme returns the scheme used to reach sandboxes.
func (c *Client) SandboxScheme() string { return c.cfg.scheme() }

// sandboxDomain falls back to the configured domain when the control plane
// reports one that is not a domain, which older platform versions do.
func (c *Client) sandboxDomain(reported string) string {
	if reported == "" || !strings.Contains(reported, ".") {
		return c.cfg.domain
	}
	return reported
}

// SandboxDomain resolves the domain to use for a sandbox, given whatever the
// control plane reported for it.
func (c *Client) SandboxDomain(reported string) string { return c.sandboxDomain(reported) }
