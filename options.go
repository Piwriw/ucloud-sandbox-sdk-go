package sandbox

import (
	"time"
)

type Logger interface {
	Printf(format string, args ...any)
}

type sandboxConfig struct {
	template            string
	timeout             int
	metadata            map[string]string
	manageBy            string
	envVars             map[string]string
	autoPause           *bool
	secure              *bool
	allowInternetAccess *bool
	network             *NetworkOpts
	autoResume          *AutoResumePolicy
	volumeMounts        []VolumeMount
	mcp                 MCPConfig
}

type SandboxOption func(*sandboxConfig)

func WithTemplate(template string) SandboxOption {
	return func(c *sandboxConfig) { c.template = template }
}
func WithTimeout(seconds int) SandboxOption {
	return func(c *sandboxConfig) { c.timeout = seconds }
}
func WithMetadata(metadata map[string]string) SandboxOption {
	return func(c *sandboxConfig) { c.metadata = metadata }
}

// WithManageBy identifies the product scenario managing the sandbox.
func WithManageBy(manageBy string) SandboxOption {
	return func(c *sandboxConfig) { c.manageBy = manageBy }
}

func WithEnvVars(envs map[string]string) SandboxOption {
	return func(c *sandboxConfig) { c.envVars = envs }
}
func WithAutoPause(autoPause bool) SandboxOption {
	return func(c *sandboxConfig) { c.autoPause = &autoPause }
}
func WithSecure(secure bool) SandboxOption {
	return func(c *sandboxConfig) { c.secure = &secure }
}
func WithAllowInternetAccess(allow bool) SandboxOption {
	return func(c *sandboxConfig) { c.allowInternetAccess = &allow }
}
func WithNetwork(opts NetworkOpts) SandboxOption {
	return func(c *sandboxConfig) { c.network = &opts }
}
func WithAutoResume(policy AutoResumePolicy) SandboxOption {
	return func(c *sandboxConfig) { c.autoResume = &policy }
}
func WithVolumeMounts(mounts []VolumeMount) SandboxOption {
	return func(c *sandboxConfig) { c.volumeMounts = mounts }
}
func WithMCP(config MCPConfig) SandboxOption {
	return func(c *sandboxConfig) { c.mcp = config }
}

func applySandboxDefaults(c *sandboxConfig) {
	if c.template == "" {
		c.template = DefaultTemplate
	}
	if c.timeout == 0 {
		c.timeout = DefaultSandboxTimeout
	}
	if c.secure == nil {
		secure := true
		c.secure = &secure
	}

	metadata := make(map[string]string, len(c.metadata)+1)
	for key, value := range c.metadata {
		metadata[key] = value
	}
	if c.manageBy != "" {
		metadata[ManageByMetadataKey] = c.manageBy
	} else if _, ok := metadata[ManageByMetadataKey]; !ok {
		metadata[ManageByMetadataKey] = DefaultManageBy
	}
	c.metadata = metadata
}

type commandConfig struct {
	cwd        string
	user       string
	envVars    map[string]string
	onStdout   func(string)
	onStderr   func(string)
	timeout    int
	timeoutSet bool
	stdin      bool
}

type CommandOption func(*commandConfig)

func WithCwd(cwd string) CommandOption    { return func(c *commandConfig) { c.cwd = cwd } }
func WithUser(user string) CommandOption  { return func(c *commandConfig) { c.user = user } }
func WithCommandEnvVars(envs map[string]string) CommandOption {
	return func(c *commandConfig) { c.envVars = envs }
}
func WithOnStdout(fn func(string)) CommandOption {
	return func(c *commandConfig) { c.onStdout = fn }
}
func WithOnStderr(fn func(string)) CommandOption {
	return func(c *commandConfig) { c.onStderr = fn }
}
func WithCommandTimeout(seconds int) CommandOption {
	return func(c *commandConfig) { c.timeout = seconds; c.timeoutSet = true }
}
func WithStdin(enabled bool) CommandOption {
	return func(c *commandConfig) { c.stdin = enabled }
}

type fileURLConfig struct {
	user       string
	expiration int
}

type FileURLOption func(*fileURLConfig)

func WithFileUser(user string) FileURLOption {
	return func(c *fileURLConfig) { c.user = user }
}
func WithSignatureExpiration(seconds int) FileURLOption {
	return func(c *fileURLConfig) { c.expiration = seconds }
}

type filesystemConfig struct {
	user         string
	depth        int
	recursive    bool
	watchTimeout int
	onExit       func(error)
}

type FilesystemOption func(*filesystemConfig)

func WithFsUser(user string) FilesystemOption {
	return func(c *filesystemConfig) { c.user = user }
}
func WithDepth(depth int) FilesystemOption {
	return func(c *filesystemConfig) { c.depth = depth }
}
func WithRecursive(recursive bool) FilesystemOption {
	return func(c *filesystemConfig) { c.recursive = recursive }
}
func WithWatchTimeout(seconds int) FilesystemOption {
	return func(c *filesystemConfig) { c.watchTimeout = seconds }
}
func WithOnFsExit(fn func(error)) FilesystemOption {
	return func(c *filesystemConfig) { c.onExit = fn }
}

type metricsConfig struct {
	start time.Time
	end   time.Time
}

type MetricsOption func(*metricsConfig)

func WithMetricsStart(t time.Time) MetricsOption { return func(c *metricsConfig) { c.start = t } }
func WithMetricsEnd(t time.Time) MetricsOption   { return func(c *metricsConfig) { c.end = t } }
