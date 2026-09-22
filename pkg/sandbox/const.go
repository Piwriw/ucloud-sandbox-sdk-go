package sandbox

// Ports exposed inside every sandbox.
const (
	// EnvdPort is where envd, the in-sandbox agent, listens.
	EnvdPort = 49983

	// MCPPort is where the MCP gateway listens.
	MCPPort = 50005
)

// Defaults applied when CreateOptions or CommandOptions leave a field unset.
const (
	// DefaultTemplate is the template a sandbox starts from.
	DefaultTemplate = "base"

	// DefaultTimeoutSeconds is how long a sandbox lives without being
	// refreshed.
	DefaultTimeoutSeconds = 300

	// DefaultCommandTimeoutSeconds bounds a single command.
	DefaultCommandTimeoutSeconds = 60
)

// keepalivePingIntervalSec tells envd how often to send a keep-alive on a
// stream, so an idle stream is not mistaken for a dead one.
const keepalivePingIntervalSec = 50

// SDKVersion is this SDK's version, reported to the platform.
const SDKVersion = "0.3.0"

// Metadata the SDK attaches to every sandbox it creates, recording which
// product opened it. Set it with CreateOptions.ManageBy.
const (
	ManageByMetadataKey = "manageby.sandbox.ucloudai.com"

	ManageByUnknown  = "unknown"
	ManageByDefault  = "ucloud-sandbox-sdk-go"
	ManageBySite     = "site"
	ManageByCodeBox  = "codebox"
	ManageByRagView  = "ragview"
	ManageBySkillLab = "skill-lab"
)

// AllTraffic matches every address, for NetworkConfig's allow and deny lists.
const AllTraffic = "0.0.0.0/0"

// ParseManageBy reads the manage-by marker out of a sandbox's metadata,
// reporting ManageByUnknown for anything it does not recognise.
func ParseManageBy(metadata map[string]string) string {
	switch value := metadata[ManageByMetadataKey]; value {
	case ManageByDefault, ManageBySite, ManageByCodeBox, ManageByRagView, ManageBySkillLab:
		return value
	default:
		return ManageByUnknown
	}
}
