package sandbox

import (
	"maps"
	"time"
)

// ClientOption configures Client construction.
type ClientOption func(*ConnectionConfig)

// WithInsecureSkipTLS disables TLS certificate verification for sandbox HTTP calls.
func WithInsecureSkipTLS(skip bool) ClientOption {
	return func(c *ConnectionConfig) { c.InsecureSkipTLS = skip }
}

// WithInsecureHTTP uses HTTP instead of HTTPS for control-plane and sandbox calls.
// Sandbox filesystem and process RPC calls also use HTTP without TLS.
func WithInsecureHTTP(insecure bool) ClientOption {
	return func(c *ConnectionConfig) { c.InsecureHTTP = insecure }
}

// WithDebug routes sandbox traffic to localhost (for local envd debugging).
func WithDebug(debug bool) ClientOption {
	return func(c *ConnectionConfig) { c.Debug = debug }
}

// WithRequestTimeout sets the default HTTP timeout for control-plane requests.
func WithRequestTimeout(timeout time.Duration) ClientOption {
	return func(c *ConnectionConfig) { c.RequestTimeout = timeout }
}

// WithAPIURL overrides the control-plane base URL (default https://api.{domain}).
func WithAPIURL(url string) ClientOption {
	return func(c *ConnectionConfig) { c.APIURL = url }
}

// WithVolumeAPIURL overrides the volume content API base URL.
// By default, volume content calls use the control-plane API URL.
func WithVolumeAPIURL(url string) ClientOption {
	return func(c *ConnectionConfig) { c.VolumeAPIURL = url }
}

// WithHeaders adds headers to control-plane and volume content API requests.
func WithHeaders(headers map[string]string) ClientOption {
	return func(c *ConnectionConfig) { c.Headers = maps.Clone(headers) }
}
