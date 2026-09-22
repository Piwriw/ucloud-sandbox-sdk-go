// Package sandbox creates and drives sandboxes.
//
// A sandbox is a VM that boots from a template and is reached two ways. The
// control plane, on Service, creates and manages sandboxes from the outside.
// envd, the agent inside the sandbox, runs commands and touches files, and is
// reached through the fields on Sandbox:
//
//	sbx, err := c.Sandboxes().Create(ctx, sandbox.CreateOptions{Template: "base"})
//	defer sbx.Kill(ctx)
//
//	out, err := sbx.Commands.Run(ctx, "python -c 'print(1+1)'", sandbox.CommandOptions{})
//	entries, err := sbx.Files.List(ctx, "/home/user", sandbox.FileOptions{})
package sandbox

import (
	"time"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

// State is a sandbox's lifecycle state.
type State = api.SandboxState

// Info describes a sandbox, as reported by Get or ListV2.
type Info struct {
	SandboxID  string
	TemplateID string

	// Name is the template's alias, when it has one.
	Name string

	Metadata map[string]string
	State    State

	CPUCount   int
	MemoryMB   int
	DiskSizeMB int

	StartedAt time.Time
	EndAt     time.Time

	EnvdVersion string

	// Domain is the base domain the sandbox's traffic is served from. Empty
	// unless the control plane named one.
	Domain string

	// AllowInternetAccess reports whether internet access was explicitly
	// turned on or off. nil means it was never set either way.
	AllowInternetAccess *bool

	// Network is the sandbox's egress configuration, when Get reported one.
	// ListV2 does not return it.
	Network *NetworkConfig

	// VolumeMounts are the volumes mounted into the sandbox.
	VolumeMounts []api.SandboxVolumeMount
}

// ExpiresAt returns when the sandbox will be shut down.
func (i *Info) ExpiresAt() time.Time { return i.EndAt }

// Metrics is one sample of a sandbox's resource use.
type Metrics struct {
	Timestamp time.Time

	CPUCount   int
	CPUUsedPct float64

	MemTotal int64
	MemUsed  int64

	DiskTotal int64
	DiskUsed  int64
}

// SnapshotInfo identifies a snapshot taken of a sandbox.
type SnapshotInfo struct {
	SnapshotID string
	Names      []string
}

// ForkResult is the outcome of one requested fork. Exactly one of Sandbox and
// Err is set: each fork succeeds or fails on its own.
type ForkResult struct {
	Sandbox *Sandbox
	Err     error
}

// NetworkConfig is a sandbox's egress policy.
type NetworkConfig struct {
	// AllowOut permits egress to these CIDRs.
	AllowOut []string

	// DenyOut blocks egress to these CIDRs. Use AllTraffic to block everything.
	DenyOut []string

	// AllowPublicTraffic controls whether the sandbox is reachable from
	// outside without a token. nil leaves the platform's default.
	AllowPublicTraffic *bool

	// MaskRequestHost rewrites the Host header on outgoing requests.
	MaskRequestHost string
}

// IAMToken defines a workload identity token the sandbox can mint.
type IAMToken struct {
	// Audience is stored exactly as given.
	Audience string

	// TokenType is the kind of token, for example "JWT-SVID".
	TokenType string
}

// MCPConfig configures MCP servers inside the sandbox, keyed by server name.
type MCPConfig map[string]any

// GitHubMCPServerConfig describes one MCP server backed by a GitHub project.
type GitHubMCPServerConfig struct {
	RunCmd     string            `json:"run_cmd"`
	InstallCmd string            `json:"install_cmd,omitempty"`
	Envs       map[string]string `json:"envs,omitempty"`
}

// NewGitHubMCPConfig builds an MCPConfig from named GitHub-backed servers.
func NewGitHubMCPConfig(servers map[string]GitHubMCPServerConfig) MCPConfig {
	config := make(MCPConfig, len(servers))
	for name, server := range servers {
		config[name] = server
	}
	return config
}

// CommandResult is what a finished command produced.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int

	// Error is envd's description of an abnormal exit, empty when the command
	// simply returned a non-zero code.
	Error string
}

// EntryType tells a file apart from a directory or a symlink.
type EntryType string

const (
	EntryTypeFile    EntryType = "file"
	EntryTypeDir     EntryType = "dir"
	EntryTypeSymlink EntryType = "symlink"
	EntryTypeUnknown EntryType = "unknown"
)

// EntryInfo is the metadata of one filesystem entry inside a sandbox.
type EntryInfo struct {
	Name string
	Path string
	Type EntryType

	Size        int64
	Mode        uint32
	Permissions string

	Owner string
	Group string

	ModifiedTime time.Time

	// SymlinkTarget is where a symlink points, nil for anything else.
	SymlinkTarget *string

	// Metadata is the user-defined extended attributes on the entry.
	Metadata map[string]string
}

// WriteInfo describes a file written into a sandbox.
type WriteInfo struct {
	Name string
	Path string
	Type EntryType
}

// WriteEntry is one file in a multi-file write. Data may be a string, a
// []byte or an io.Reader.
type WriteEntry struct {
	Path string
	Data any
}

// ProcessInfo describes a process running inside a sandbox.
type ProcessInfo struct {
	PID  int
	Tag  string
	Cmd  string
	Args []string
	Envs map[string]string
	Cwd  string
}

// PtySize is a pseudo-terminal's dimensions, in character cells.
type PtySize struct {
	Rows int
	Cols int
}

// FilesystemEventType is what happened to a watched entry.
type FilesystemEventType string

const (
	EventTypeCreate FilesystemEventType = "create"
	EventTypeWrite  FilesystemEventType = "write"
	EventTypeRemove FilesystemEventType = "remove"
	EventTypeRename FilesystemEventType = "rename"
	EventTypeChmod  FilesystemEventType = "chmod"
)

// FilesystemEvent reports a change under a watched directory.
type FilesystemEvent struct {
	// Name is the entry's path relative to the watched directory.
	Name string

	Type FilesystemEventType

	// Entry is the affected entry's metadata, set only when the watch asked
	// for it and the entry still exists. A remove leaves it nil.
	Entry *EntryInfo
}

// LogEntry is one line of a sandbox's log.
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
	Fields    map[string]string
}

func (e LogEntry) String() string { return e.Message }
