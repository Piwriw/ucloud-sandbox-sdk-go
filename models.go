package sandbox

import "time"

type SandboxInfo struct {
	SandboxID   string            `json:"sandboxID"`
	TemplateID  string            `json:"templateID"`
	Name        string            `json:"alias"`
	Metadata    map[string]string `json:"metadata"`
	StartedAt   time.Time         `json:"startedAt"`
	EndAt       time.Time         `json:"endAt"`
	State       string            `json:"state"`
	CPUCount    int               `json:"cpuCount"`
	DiskSizeMB  int32             `json:"diskSizeMB"`
	MemoryMB    int               `json:"memoryMB"`
	EnvdVersion string            `json:"envdVersion"`
}

type SandboxMetrics struct {
	CPUCount   int       `json:"cpuCount"`
	CPUUsedPct float64   `json:"cpuUsedPct"`
	DiskTotal  int64     `json:"diskTotal"`
	DiskUsed   int64     `json:"diskUsed"`
	MemTotal   int64     `json:"memTotal"`
	MemUsed    int64     `json:"memUsed"`
	Timestamp  time.Time `json:"timestamp"`
}

type SnapshotInfo struct {
	SnapshotID string   `json:"snapshotID"`
	Names      []string `json:"names,omitempty"`
}

type CommandResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
	Error    string `json:"error,omitempty"`
}

type EntryType string

const (
	EntryTypeFile EntryType = "file"
	EntryTypeDir  EntryType = "dir"
)

type EntryInfo struct {
	Name          string    `json:"name"`
	Path          string    `json:"path"`
	Type          EntryType `json:"type"`
	Size          int64     `json:"size"`
	Permissions   string    `json:"permissions"`
	Mode          uint32    `json:"mode"`
	Owner         string    `json:"owner"`
	Group         string    `json:"group"`
	ModifiedTime  time.Time `json:"modifiedTime"`
	SymlinkTarget *string   `json:"symlinkTarget,omitempty"`
}

type WriteInfo struct {
	Name string     `json:"name"`
	Type *EntryType `json:"type"`
	Path string     `json:"path"`
}

type WriteEntry struct {
	Path string
	Data any
}

type ProcessInfo struct {
	PID  int               `json:"pid"`
	Tag  string            `json:"tag"`
	Cmd  string            `json:"cmd"`
	Args []string          `json:"args"`
	Envs map[string]string `json:"envs"`
	Cwd  string            `json:"cwd"`
}

type PtySize struct {
	Rows int `json:"rows"`
	Cols int `json:"cols"`
}

type SandboxQuery struct {
	Metadata map[string]string
	State    []string
}

type NetworkOpts struct {
	AllowOut           []string `json:"allowOut,omitempty"`
	DenyOut            []string `json:"denyOut,omitempty"`
	AllowPublicTraffic *bool    `json:"allowPublicTraffic,omitempty"`
	MaskRequestHost    string   `json:"maskRequestHost,omitempty"`
}

type AutoResumePolicy string

const (
	AutoResumePolicyOff AutoResumePolicy = "off"
	AutoResumePolicyOn  AutoResumePolicy = "on"
)

type VolumeMount struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type MCPConfig map[string]any

type GitHubMCPServerConfig struct {
	RunCmd     string            `json:"run_cmd"`
	InstallCmd string            `json:"install_cmd,omitempty"`
	Envs       map[string]string `json:"envs,omitempty"`
}

func NewGitHubMCPConfig(servers map[string]GitHubMCPServerConfig) MCPConfig {
	config := MCPConfig{}
	for k, v := range servers {
		config[k] = v
	}
	return config
}

type FilesystemEventType string

const (
	EventTypeCreate FilesystemEventType = "create"
	EventTypeWrite  FilesystemEventType = "write"
	EventTypeRemove FilesystemEventType = "remove"
	EventTypeRename FilesystemEventType = "rename"
	EventTypeChmod  FilesystemEventType = "chmod"
)

type FilesystemEvent struct {
	Name string
	Type FilesystemEventType
}

// ExpiredAtUnix returns sandbox expiry as a Unix timestamp.
func (i *SandboxInfo) ExpiredAtUnix() int64 { return i.EndAt.Unix() }

type TemplateInfo struct {
	TemplateID    string    `json:"templateID"`
	BuildID       string    `json:"buildID,omitempty"`
	Names         []string  `json:"names,omitempty"`   // v2 API: namespace/alias format
	Aliases       []string  `json:"aliases,omitempty"` // deprecated but still returned
	Public        bool      `json:"public"`
	CPUCount      int       `json:"cpuCount"`
	MemoryMB      int       `json:"memoryMB"`
	DiskSizeMB    int       `json:"diskSizeMB,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt,omitempty"`
	LastSpawnedAt time.Time `json:"lastSpawnedAt,omitempty"`
	SpawnCount    int       `json:"spawnCount,omitempty"`
	BuildCount    int       `json:"buildCount,omitempty"`
	EnvdVersion   string    `json:"envdVersion,omitempty"`
	BuildStatus   string    `json:"buildStatus,omitempty"` // building, waiting, ready, error, uploaded
	CreatedBy     *struct {
		Email string `json:"email,omitempty"`
		ID    string `json:"id,omitempty"`
	} `json:"createdBy,omitempty"`
}
