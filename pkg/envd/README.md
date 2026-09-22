# pkg/envd

Generated clients for **envd**, the agent running inside each sandbox. Do not
edit by hand — run `make generate` at the repository root instead.

envd exposes two surfaces, and this package mirrors both:

| Package | Surface | Source spec |
| --- | --- | --- |
| `api` | HTTP: `/health`, `/metrics`, `/envs`, `/files`, `/files/compose` | `packages/envd/spec/envd.yaml` |
| `process`, `process/processconnect` | connect-rpc: start, connect, signal, stdin | `packages/envd/spec/process/process.proto` |
| `filesystem`, `filesystem/filesystemconnect` | connect-rpc: stat, list, move, remove, watch | `packages/envd/spec/filesystem/filesystem.proto` |

## Provenance

The specs come from [e2b-dev/runtime](https://github.com/e2b-dev/runtime),
vendored as the `submodules/e2b-runtime` submodule and pinned to tag
**`2026.30`**. That project is licensed under Apache License 2.0, a copy of
which is in [`LICENSE-e2b`](./LICENSE-e2b).

To regenerate after moving to a newer e2b release:

```bash
git -C submodules/e2b-runtime fetch --tags
git -C submodules/e2b-runtime checkout <new-tag>
make generate
```

## Hand-written files

Everything here is generated except [`api/secure_token.go`](./api/secure_token.go),
which supplies a type that envd's spec references but only defines inside envd
itself. See the comment in that file.
