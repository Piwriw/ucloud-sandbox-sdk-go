// Package sandbox is the Go client for UCloud Sandbox (E2B-compatible API).
//
// Create a client with your region domain and API key, then provision sandboxes,
// run commands, manage persistent volumes and files, and create snapshots.
//
//	apiDomain example: cn-wlcb.sandbox.ucloudai.com
//	apiKey: your sandbox API key
//
//	client := sandbox.NewClient(apiDomain, apiKey)
//	sbx, err := client.CreateSandbox(ctx, sandbox.WithTemplate("base"))
package sandbox
