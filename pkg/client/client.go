// Package client is the SDK's entry point. It resolves configuration once and
// hands out the domain services:
//
//	c, err := client.New(client.Options{APIKey: apiKey})
//
//	sbx, err := c.Sandboxes().Create(ctx, sandbox.CreateOptions{Template: "base"})
//	defer sbx.Kill(ctx)
//
//	out, err := sbx.Commands.Run(ctx, "uname -a", sandbox.CommandOptions{})
//
// Endpoints the SDK does not wrap -- teams, API keys, the admin surface --
// remain reachable through API, which returns the generated client.
package client

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/secret"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/template"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume"
)

// Options configures a Client.
//
// The zero value works when the environment supplies an API key: every field
// falls back to an environment variable and then to a default. See
// transport.Config, which this is an alias for, for the resolution order.
type Options = transport.Config

// Client holds the resolved configuration and the domain services built on it.
// It is safe for concurrent use.
type Client struct {
	t *transport.Client

	sandboxes *sandbox.Service
	templates *template.Service
	volumes   *volume.Service
	secrets   *secret.Service
}

// New resolves opts and builds a Client. It fails when no API key can be found
// in either the options or the environment.
func New(opts Options) (*Client, error) {
	t, err := transport.New(opts)
	if err != nil {
		return nil, err
	}

	return &Client{
		t:         t,
		sandboxes: sandbox.NewService(t),
		templates: template.NewService(t),
		volumes:   volume.NewService(t),
		secrets:   secret.NewService(t),
	}, nil
}

// Sandboxes creates and manages sandboxes.
func (c *Client) Sandboxes() *sandbox.Service { return c.sandboxes }

// Templates builds and manages the images sandboxes boot from.
func (c *Client) Templates() *template.Service { return c.templates }

// Volumes manages persistent volumes.
func (c *Client) Volumes() *volume.Service { return c.volumes }

// Secrets manages project secrets.
func (c *Client) Secrets() *secret.Service { return c.secrets }

// API returns the generated control-plane client, for endpoints this SDK does
// not wrap. Its method names come from the generator, so they read less well
// than the services above.
func (c *Client) API() *api.ClientWithResponses { return c.t.API() }

// Transport returns the underlying transport client, for building a domain
// service by hand.
func (c *Client) Transport() *transport.Client { return c.t }

// Region returns the resolved region, for example "cn-wlcb".
func (c *Client) Region() string { return c.t.Region() }

// Domain returns the resolved sandbox domain.
func (c *Client) Domain() string { return c.t.Domain() }

// Health checks that the control plane is reachable.
func (c *Client) Health(ctx context.Context) error {
	return c.sandboxes.Health(ctx)
}
