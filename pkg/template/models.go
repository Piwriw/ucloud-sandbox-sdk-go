// Package template builds and manages sandbox templates.
//
// A template is the image a sandbox boots from. Describe one with a Builder,
// then hand it to Service.Build, which creates the template, uploads any local
// files it copies in, starts the build and waits for it to finish:
//
//	tpls := c.Templates()
//
//	tpl := tpls.NewBuilder(template.BuilderOptions{}).
//	    FromImage("ubuntu:22.04").
//	    RunCmd("apt-get update && apt-get install -y python3").
//	    SetWorkdir("/app")
//
//	build, err := tpls.Build(ctx, tpl, "my-python", template.BuildOptions{
//	    CPUCount: 2,
//	    MemoryMB: 2048,
//	    OnLogs:   template.DefaultLogger(),
//	})
//
// A Builder with no explicit base starts from the platform's base image for the
// client's region; see NewBuilder.
package template

import (
	"time"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

// BuildStatus is where a build has got to.
type BuildStatus = api.TemplateBuildStatus

const (
	BuildStatusBuilding BuildStatus = api.TemplateBuildStatusBuilding
	BuildStatusWaiting  BuildStatus = api.TemplateBuildStatusWaiting
	BuildStatusReady    BuildStatus = api.TemplateBuildStatusReady
	BuildStatusError    BuildStatus = api.TemplateBuildStatusError
)

// LogLevel is the severity of a build log entry.
type LogLevel = api.LogLevel

const (
	LogLevelDebug LogLevel = api.LogLevelDebug
	LogLevelInfo  LogLevel = api.LogLevelInfo
	LogLevelWarn  LogLevel = api.LogLevelWarn
	LogLevelError LogLevel = api.LogLevelError
)

// LogEntry is one line of a build log.
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string

	// Step names the build step the entry came from, when the platform
	// attributed it to one.
	Step string
}

func (e LogEntry) String() string { return e.Message }

// BuildInfo identifies a build started by Service.Build.
type BuildInfo struct {
	TemplateID string
	BuildID    string
	Name       string
	Tags       []string
}

// StatusReason explains why a build ended the way it did. Present on failure.
type StatusReason struct {
	Message    string
	Step       string
	LogEntries []LogEntry
}

// Status is a build's state together with whatever logs came with it.
type Status struct {
	TemplateID string
	BuildID    string
	Status     BuildStatus
	Logs       []string
	LogEntries []LogEntry
	Reason     *StatusReason
}

// Info is a template as the listing endpoint reports it.
type Info struct {
	TemplateID string
	BuildID    string

	// Names are the template's names, "namespace/alias" where namespaced.
	Names []string

	// Aliases is the older, un-namespaced spelling of Names, still returned by
	// the platform.
	Aliases []string

	Public      bool
	CPUCount    int
	MemoryMB    int
	DiskSizeMB  int
	BuildCount  int
	SpawnCount  int64
	EnvdVersion string

	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastSpawnedAt *time.Time

	CreatedByEmail string
	CreatedByID    string
}

// Build is one build of a template.
type Build struct {
	BuildID     string
	Status      BuildStatus
	CPUCount    int
	MemoryMB    int
	DiskSizeMB  int
	EnvdVersion string

	CreatedAt  time.Time
	UpdatedAt  time.Time
	FinishedAt *time.Time
}

// WithBuilds is a template together with one page of its builds.
type WithBuilds struct {
	TemplateID string
	Public     bool
	Names      []string
	Aliases    []string
	SpawnCount int64

	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastSpawnedAt *time.Time

	Builds []Build

	// NextToken is the cursor for the next page of builds, empty on the last
	// page. It comes from the X-Next-Token response header.
	NextToken string
}

// Tag is a tag pointing at one build of a template.
type Tag struct {
	Tag       string
	BuildID   string
	CreatedAt time.Time
}

// AssignedTags is the result of assigning tags to a build.
type AssignedTags struct {
	BuildID string
	Tags    []string
}

// Alias is what a template alias resolves to.
type Alias struct {
	TemplateID string
	Public     bool
}

// logEntryFrom converts a generated log entry.
func logEntryFrom(e api.BuildLogEntry) LogEntry {
	entry := LogEntry{Timestamp: e.Timestamp, Level: e.Level, Message: e.Message}
	if e.Step != nil {
		entry.Step = *e.Step
	}
	return entry
}

// logEntriesFrom converts a slice of generated log entries.
func logEntriesFrom(entries []api.BuildLogEntry) []LogEntry {
	converted := make([]LogEntry, 0, len(entries))
	for _, e := range entries {
		converted = append(converted, logEntryFrom(e))
	}
	return converted
}
