package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// Create starts a sandbox and returns a handle to it.
//
// The caller owns the sandbox's lifetime: it lives for TimeoutSeconds unless
// refreshed, and Kill ends it sooner.
//
// POST /sandboxes
func (s *Service) Create(ctx context.Context, opts CreateOptions) (*Sandbox, error) {
	body := api.PostSandboxesJSONRequestBody{
		TemplateID:          orDefault(opts.Template, DefaultTemplate),
		AutoPause:           opts.AutoPause,
		AutoPauseMemory:     opts.AutoPauseMemory,
		Secure:              opts.Secure,
		AllowInternetAccess: opts.AllowInternetAccess,
	}

	timeout := int32(orDefaultInt(opts.TimeoutSeconds, DefaultTimeoutSeconds))
	body.Timeout = &timeout

	metadata := withManageBy(opts.Metadata, opts.ManageBy)
	body.Metadata = (*api.SandboxMetadata)(&metadata)

	if len(opts.EnvVars) > 0 {
		envs := api.EnvVars(opts.EnvVars)
		body.EnvVars = &envs
	}
	if len(opts.VolumeMounts) > 0 {
		mounts := opts.VolumeMounts
		body.VolumeMounts = &mounts
	}
	if len(opts.MCP) > 0 {
		mcp := api.Mcp(opts.MCP)
		body.Mcp = &mcp
	}
	if opts.Network != nil {
		body.Network = opts.Network.toCreateConfig()
	}
	if opts.AutoResume != nil {
		body.AutoResume = &api.SandboxAutoResumeConfig{Enabled: *opts.AutoResume}
	}
	if len(opts.IAMTokens) > 0 {
		tokens := make(api.SandboxIamTokens, len(opts.IAMTokens))
		for name, token := range opts.IAMTokens {
			tokens[name] = api.SandboxIamToken{
				Audience:  token.Audience,
				TokenType: token.TokenType,
			}
		}
		body.Iam = &api.SandboxIam{Tokens: &tokens}
	}

	resp, err := s.t.API().PostSandboxesWithResponse(ctx, body)
	if err != nil {
		return nil, err
	}
	created, err := transport.Parsed(resp.JSON201, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}

	// A template built by an envd too old to talk to is worse than no sandbox:
	// every later call would fail in a way that points at the wrong thing. Shut
	// it down and say what is actually wrong.
	if !parseEnvdVersion(created.EnvdVersion).supports(envdVersionMinimum) {
		_, _ = s.Kill(ctx, created.SandboxID)
		return nil, &errdefs.TemplateError{SandboxError: errdefs.SandboxError{
			Message: "template is too old for this SDK; rebuild it with a current template build",
		}}
	}

	return s.newSandbox(created.SandboxID, *created)
}

// withManageBy adds the manage-by marker to a copy of the caller's metadata,
// leaving their map untouched.
func withManageBy(metadata map[string]string, manageBy string) map[string]string {
	merged := make(map[string]string, len(metadata)+1)
	for k, v := range metadata {
		merged[k] = v
	}
	if _, ok := merged[ManageByMetadataKey]; !ok {
		merged[ManageByMetadataKey] = orDefault(manageBy, ManageByDefault)
	}
	return merged
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func orDefaultInt(value, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}
