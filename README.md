# ucloud-sandbox-sdk-go

Go SDK for UCloud Sandbox: cloud VMs that boot in about a second, for running
code you did not write.

## Install

```bash
go get github.com/ucloud/ucloud-sandbox-sdk-go
```

Requires Go 1.26 or newer.

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/client"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox"
)

func main() {
	ctx := context.Background()

	c, err := client.New(client.Options{APIKey: "your-api-key"})
	if err != nil {
		log.Fatal(err)
	}

	sbx, err := c.Sandboxes().Create(ctx, sandbox.CreateOptions{Template: "base"})
	if err != nil {
		log.Fatal(err)
	}
	defer sbx.Kill(ctx)

	out, err := sbx.Commands.Run(ctx, "uname -a", sandbox.CommandOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(out.Stdout)
}
```

Runnable versions of this and more are in [`examples/`](examples/).

## Configuration

`client.New` takes everything through `client.Options`. Leave a field empty and
it falls back to an environment variable, then to a default:

| Field | Environment variable | Default |
| --- | --- | --- |
| `APIKey` | `UCLOUD_SANDBOX_API_KEY` | required — `New` fails without it |
| `Region` | `UCLOUD_SANDBOX_REGION` | `cn-wlcb` |
| `Domain` | `UCLOUD_SANDBOX_DOMAIN` | `{region}.sandbox.ucloudai.com` |
| `APIURL` | `UCLOUD_SANDBOX_API_URL` | `https://api.{domain}` |
| `InsecureHTTP` | `UCLOUD_SANDBOX_INSECURE_HTTP` | `false` |

`Region` expands to a domain; `Domain` overrides it; `APIURL` overrides both, so
a private deployment can be named by address:

```go
c, err := client.New(client.Options{
	APIKey:       apiKey,
	APIURL:       "http://10.10.0.5:8080",
	InsecureHTTP: true,
})
```

## Packages

| Package | What it does |
| --- | --- |
| [`pkg/client`](pkg/client) | Entry point. `New` plus the four services below. |
| [`pkg/sandbox`](pkg/sandbox) | Sandboxes, and the commands, files and PTYs inside them. |
| [`pkg/template`](pkg/template) | Building the images sandboxes boot from. |
| [`pkg/volume`](pkg/volume) | Persistent volumes. |
| [`pkg/secret`](pkg/secret) | Project secrets. |
| [`pkg/errdefs`](pkg/errdefs) | Error types, and the mappings that produce them. |
| [`pkg/transport`](pkg/transport) | HTTP plumbing. Rarely used directly. |
| [`pkg/api`](pkg/api), [`pkg/envd`](pkg/envd) | Generated clients. See below. |

### Conventions

Required arguments are positional; optional ones go in a struct whose zero value
means "defaults":

```go
sbx, err := c.Sandboxes().Create(ctx, sandbox.CreateOptions{TimeoutSeconds: 3600})
out, err := sbx.Commands.Run(ctx, "ls", sandbox.CommandOptions{})
```

A field that must distinguish "unset" from a zero value is a pointer; `new`
builds one:

```go
sbx.Pause(ctx, sandbox.PauseOptions{Memory: new(false)})
```

Durations that travel to the server are named `...Seconds` and are plain ints,
so `3600` means an hour rather than 3.6 microseconds.

Where an endpoint has versions, the method carries the version and only the
newest is exposed: `Sandboxes().ListV2`, `Templates().CreateV3`. Deprecated
versions are reachable through `pkg/api` if you need them.

### Errors

Every failure, whether it came over HTTP or over envd's RPC, lands on the same
types:

```go
if errors.Is(err, errdefs.ErrNotFound) { ... }

var exit *errdefs.CommandExitError
if errors.As(err, &exit) {
	log.Print(exit.Stderr)
}
```

### Endpoints this SDK does not wrap

Teams, API keys and the admin surface are generated but not wrapped. Reach them
through the generated client, whose method names come from the generator:

```go
resp, err := c.API().GetTeamsWithResponse(ctx)
```

## Development

The clients in `pkg/api` and `pkg/envd` are generated. Regenerating them needs
the submodules:

```bash
git submodule update --init
make generate
make test
```

The generated code is committed, so `go get` works without any of this.

`spec/openapi.yml` describes the control plane. envd's specs come from
[e2b-dev/runtime](https://github.com/e2b-dev/runtime), vendored under
`submodules/` and pinned to a tag; see [`pkg/envd/README.md`](pkg/envd/README.md).
`submodules/ucloud-sandbox-sdk-python` is the Python SDK, kept only as a
reference for behaviour the two should share.

## Migrating from v0.2

v0.3 is a rewrite. Everything moved out of the root package, and functional
options are gone.

```go
// v0.2
c := sandbox.NewClient("cn-wlcb.sandbox.ucloudai.com", apiKey)
sbx, err := c.CreateSandbox(ctx, sandbox.WithTemplate("base"), sandbox.WithTimeout(300))
out, err := sbx.Commands.Run(ctx, "ls", sandbox.WithCwd("/app"))

// v0.3
c, err := client.New(client.Options{APIKey: apiKey, Region: "cn-wlcb"})
sbx, err := c.Sandboxes().Create(ctx, sandbox.CreateOptions{Template: "base", TimeoutSeconds: 300})
out, err := sbx.Commands.Run(ctx, "ls", sandbox.CommandOptions{Cwd: "/app"})
```

Renames worth knowing:

| v0.2 | v0.3 |
| --- | --- |
| `sandbox.NewClient(domain, key)` | `client.New(client.Options{...})`, returns an error |
| `client.CreateSandbox` | `c.Sandboxes().Create` |
| `client.ListSandboxes` | `c.Sandboxes().ListV2` |
| `client.GetSandboxLogs` | `c.Sandboxes().LogsV2` |
| `client.BuildTemplate` | `c.Templates().Build` |
| `client.ListTemplates` | `c.Templates().ListV2` |
| `client.CreateVolume` | `c.Volumes().Create` |
| `sandbox.NewTemplate()` | `c.Templates().NewBuilder(...)` |
| `WithXxx(v)` options | `XxxOptions{Xxx: v}` structs |

Three behaviour changes to check before upgrading:

- **TLS verification is on.** v0.2 disabled certificate checking for everyone.
  If your deployment uses a self-signed certificate, set
  `client.Options{InsecureSkipTLS: true}` explicitly.
- **A sandbox created with `AllowInternetAccess: new(false)` now really has no
  internet access.** v0.2 sent the wrong field name, so the setting was ignored.
- **Listings paginate.** `ListV2` and `ListSnapshots` follow the cursor;
  v0.2 stopped after one page.

Dropped: the volume content API (`Volume.ReadFile`, `WriteFile` and friends).
UCloud does not serve it. Read and write a volume from inside a sandbox that
mounts it.

## Licence

MIT. `pkg/envd` is generated from Apache-2.0 specs; see
[`pkg/envd/LICENSE-e2b`](pkg/envd/LICENSE-e2b).
