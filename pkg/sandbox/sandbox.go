package sandbox

import (
	"context"
	"sync"
)

// Sandbox is a handle to one running sandbox.
//
// The three fields reach envd inside the sandbox; the methods reach the control
// plane outside it. Get one from Service.Create, Service.Connect or
// Service.Resume.
type Sandbox struct {
	// ID identifies the sandbox to the control plane.
	ID string

	// Domain is the base domain the sandbox's traffic is served from.
	Domain string

	// EnvdVersion is the version of the agent running inside the sandbox.
	// Some capabilities depend on it; the SDK checks before using them.
	EnvdVersion string

	// Commands runs and manages processes inside the sandbox.
	Commands *Commands

	// Files reads and writes the sandbox's filesystem.
	Files *Filesystem

	// Pty opens pseudo-terminals inside the sandbox.
	Pty *Pty

	svc         *Service
	conn        *envdConn
	envdVersion envdVersion

	mcpTokenOnce sync.Once
	mcpToken     string
	mcpTokenErr  error
}

// Kill shuts the sandbox down. It reports false when the sandbox was already
// gone.
func (s *Sandbox) Kill(ctx context.Context) (bool, error) {
	return s.svc.Kill(ctx, s.ID)
}

// Pause stops the sandbox and keeps its state, so it can be resumed later.
func (s *Sandbox) Pause(ctx context.Context, opts PauseOptions) error {
	return s.svc.Pause(ctx, s.ID, opts)
}

// SetTimeout resets how long the sandbox will live from now.
func (s *Sandbox) SetTimeout(ctx context.Context, timeoutSeconds int) error {
	return s.svc.SetTimeout(ctx, s.ID, timeoutSeconds)
}

// Refresh keeps the sandbox alive for another durationSeconds, up to an hour.
func (s *Sandbox) Refresh(ctx context.Context, durationSeconds int) error {
	return s.svc.Refresh(ctx, s.ID, durationSeconds)
}

// GetInfo returns the sandbox's current state.
func (s *Sandbox) GetInfo(ctx context.Context) (*Info, error) {
	return s.svc.Get(ctx, s.ID)
}

// Metrics returns the sandbox's resource use over a window.
func (s *Sandbox) Metrics(ctx context.Context, opts MetricsOptions) ([]Metrics, error) {
	return s.svc.Metrics(ctx, s.ID, opts)
}

// LogsV2 returns the sandbox's logs.
func (s *Sandbox) LogsV2(ctx context.Context, opts LogsV2Options) ([]LogEntry, error) {
	return s.svc.LogsV2(ctx, s.ID, opts)
}

// UpdateNetwork replaces the sandbox's egress policy.
func (s *Sandbox) UpdateNetwork(ctx context.Context, update NetworkUpdate) error {
	return s.svc.UpdateNetwork(ctx, s.ID, update)
}

// Fork starts copies of the sandbox from a single snapshot of it.
func (s *Sandbox) Fork(ctx context.Context, opts ForkOptions) ([]ForkResult, error) {
	return s.svc.Fork(ctx, s.ID, opts)
}

// CreateSnapshot captures the sandbox as a template that can be booted later.
func (s *Sandbox) CreateSnapshot(ctx context.Context, opts SnapshotOptions) (*SnapshotInfo, error) {
	return s.svc.CreateSnapshot(ctx, s.ID, opts)
}

// Host returns the host:port that reaches a port inside the sandbox.
func (s *Sandbox) Host(port int) string {
	return s.svc.t.SandboxHost(s.ID, s.Domain, port)
}

// EnvdURL returns the base URL of the sandbox's envd.
func (s *Sandbox) EnvdURL() string { return s.conn.baseURL }

// resolveUser picks the user an envd call acts as.
//
// envd gained a notion of a default user in 0.4.0. Older agents have no such
// default and fall back to root unless told otherwise, so "user" is named
// explicitly for them.
func (s *Sandbox) resolveUser(user string) string {
	if user != "" {
		return user
	}
	if s.envdVersion.supports(envdVersionDefaultUser) {
		return ""
	}
	return "user"
}
