package sandbox

// Package-level defaults and envd version gates.
const (
	EnvdPort = 49983
	MCPPort  = 50005

	KeepalivePingIntervalSec = 50
	DefaultCommandTimeout    = 60
	DefaultSandboxTimeout    = 300
	DefaultTemplate          = "base"
	ManageByMetadataKey      = "manageby.sandbox.ucloudai.com"
	DefaultManageBy          = "ucloud-sandbox-sdk-go"
	ManageBySite             = "site"
	ManageByCodeBox          = "codebox"

	SDKVersion = "0.2.2"
	AllTraffic = "0.0.0.0/0"
)

var (
	envdVersionStdin          = [3]int{0, 3, 0}
	envdVersionRecursiveWatch = [3]int{0, 1, 4}
	envdVersionDefaultUser    = [3]int{0, 4, 0}
	envdVersionMetrics        = [3]int{0, 1, 5}
	envdVersionDiskMetrics    = [3]int{0, 2, 4}
)
