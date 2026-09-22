package sandbox

import (
	"time"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

// Several options below are pointers, so that "unset" is distinguishable from
// a zero value: Memory defaulting to true means false has to be expressible.
// Build one with new:
//
//	sandbox.PauseOptions{Memory: new(false)}

// CreateOptions are the optional arguments to Service.Create.
type CreateOptions struct {
	// Template the sandbox boots from. Empty means DefaultTemplate.
	Template string

	// TimeoutSeconds is how long the sandbox lives without being refreshed.
	// Zero means DefaultTimeoutSeconds.
	TimeoutSeconds int

	// Metadata is stored on the sandbox and can be filtered on in ListV2.
	Metadata map[string]string

	// EnvVars are set inside the sandbox. A value may be a secret placeholder;
	// see secret.Fill.
	EnvVars map[string]string

	// ManageBy records which product opened the sandbox, under
	// ManageByMetadataKey. Empty means ManageByDefault.
	ManageBy string

	// VolumeMounts are the volumes to mount. Build one with volume.Volume.Mount.
	VolumeMounts []api.SandboxVolumeMount

	// MCP configures MCP servers inside the sandbox.
	MCP MCPConfig

	// Network is the egress policy. nil leaves the platform's default.
	Network *NetworkConfig

	// IAMTokens are workload identity tokens the sandbox may mint, keyed by a
	// name of your choosing.
	IAMTokens map[string]IAMToken

	// AutoPause pauses the sandbox on timeout instead of killing it.
	AutoPause *bool

	// AutoPauseMemory controls what an auto-pause captures. true, the default,
	// snapshots memory as well as the filesystem. false persists only the
	// filesystem, so resuming cold-boots; such a snapshot cannot be
	// auto-resumed and so cannot be combined with AutoResume.
	AutoPauseMemory *bool

	// AutoResume lets arbitrary traffic resume a paused sandbox.
	AutoResume *bool

	// Secure encrypts all system communication with the sandbox.
	Secure *bool

	// AllowInternetAccess permits egress. false behaves like denying
	// AllTraffic in Network.
	AllowInternetAccess *bool
}

// ConnectOptions are the optional arguments to Service.Connect.
type ConnectOptions struct {
	// TimeoutSeconds extends the sandbox's life from now. Zero means
	// DefaultTimeoutSeconds.
	TimeoutSeconds int

	// Memory, when false and the sandbox is paused, resumes from disk only:
	// the sandbox cold-boots and the snapshot's memory is ignored, never
	// modified. nil leaves the platform's default of true.
	Memory *bool
}

// ListV2Options are the optional arguments to Service.ListV2.
type ListV2Options struct {
	// Metadata filters to sandboxes carrying all of these key/value pairs.
	Metadata map[string]string

	// State filters by lifecycle state.
	State []State

	// Template filters to sandboxes started from this template.
	Template string

	// StartedAfter filters to sandboxes started after this time.
	StartedAfter time.Time

	// OrderDescending returns newest first.
	OrderDescending bool

	// Limit is how many sandboxes to fetch per page. Zero lets the platform
	// choose.
	Limit int
}

// PauseOptions are the optional arguments to Service.Pause.
type PauseOptions struct {
	// Memory controls what the pause captures. nil or true snapshots memory
	// as well as the filesystem. false persists only the filesystem, so
	// resuming cold-boots and loses running processes and open connections;
	// such a sandbox must be resumed explicitly and cannot be auto-resumed.
	Memory *bool
}

// ResumeOptions are the optional arguments to Service.Resume.
type ResumeOptions struct {
	// TimeoutSeconds is how long the resumed sandbox lives. Zero means
	// DefaultTimeoutSeconds.
	TimeoutSeconds int

	// Memory, when false, resumes from disk state only. See
	// ConnectOptions.Memory.
	Memory *bool
}

// ForkOptions are the optional arguments to Service.Fork.
type ForkOptions struct {
	// Count is how many forks to create, 1 to 100. Zero means one. All forks
	// boot from a single snapshot, which is captured once.
	Count int

	// TimeoutSeconds is how long each fork lives. Zero means
	// DefaultTimeoutSeconds.
	TimeoutSeconds int
}

// MetricsOptions are the optional arguments to Service.Metrics.
type MetricsOptions struct {
	// StartUnix and EndUnix bound the window, in seconds since the epoch.
	// Zero leaves that end open.
	StartUnix int64
	EndUnix   int64
}

// LogsV2Options are the optional arguments to Service.LogsV2.
type LogsV2Options struct {
	// CursorMs is the timestamp to read from, in milliseconds.
	CursorMs *int64

	// Limit caps how many entries come back. Zero lets the platform choose.
	Limit int

	// Direction reads forward or backward from the cursor.
	Direction LogsDirection

	// Level filters entries by minimum severity.
	Level string

	// Search filters entries by substring.
	Search string
}

// LogsDirection reads logs forward or backward from the cursor.
type LogsDirection = api.LogsDirection

const (
	LogsDirectionForward  LogsDirection = "forward"
	LogsDirectionBackward LogsDirection = "backward"
)

// SnapshotOptions are the optional arguments to Sandbox.CreateSnapshot.
type SnapshotOptions struct {
	// Name for the snapshot template. Reusing a name adds a build to the
	// existing template instead of creating another one.
	Name string
}

// ListSnapshotsOptions are the optional arguments to Service.ListSnapshots.
type ListSnapshotsOptions struct {
	// SandboxID filters to snapshots taken of one sandbox.
	SandboxID string

	// Name filters by name or ID, optionally tag-qualified.
	Name string

	// Limit is how many snapshots to fetch per page. Zero lets the platform
	// choose.
	Limit int
}

// CommandOptions are the optional arguments to the Commands methods.
type CommandOptions struct {
	// Cwd is the working directory. Empty uses the sandbox's default.
	Cwd string

	// User to run as. Empty uses the sandbox's default user.
	User string

	// EnvVars are added to the command's environment.
	EnvVars map[string]string

	// TimeoutSeconds bounds the command. Zero means
	// DefaultCommandTimeoutSeconds; a negative value means no timeout.
	TimeoutSeconds int

	// Stdin keeps the command's stdin open, so SendStdin can write to it.
	Stdin bool

	// OnStdout and OnStderr receive output as it arrives. Both are called
	// from the stream's goroutine, one call at a time.
	OnStdout func(string)
	OnStderr func(string)
}

// FileOptions are the optional arguments to the Filesystem methods.
type FileOptions struct {
	// User to act as. Empty uses the sandbox's default user.
	User string

	// Depth is how far below the listed path to descend, for List. Zero lists
	// the immediate children.
	Depth uint32
}

// WatchOptions are the optional arguments to Filesystem.Watch and
// Filesystem.CreateWatcher.
type WatchOptions struct {
	// User to act as. Empty uses the sandbox's default user.
	User string

	// Recursive watches the whole tree below the path. Requires envd 0.1.4 or
	// newer.
	Recursive bool

	// IncludeEntry attaches each affected entry's metadata to its event,
	// where the entry still exists.
	IncludeEntry bool

	// AllowNetworkMounts permits watching a network filesystem, where events
	// may be unreliable or absent altogether.
	AllowNetworkMounts bool

	// TimeoutSeconds stops the watch after this long. Zero watches until the
	// handle is stopped or the context ends.
	TimeoutSeconds int

	// OnExit is called once the watch ends, with the error that ended it or
	// nil for a clean stop.
	OnExit func(error)
}

// FileURLOptions are the optional arguments to Sandbox.DownloadURL and
// Sandbox.UploadURL.
type FileURLOptions struct {
	// User the URL acts as. Empty uses the sandbox's default user.
	User string

	// ExpirationSeconds limits how long the signed URL stays valid. Zero
	// leaves it without an expiry.
	ExpirationSeconds int
}
