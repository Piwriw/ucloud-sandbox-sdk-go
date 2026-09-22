package template

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// LogsDirection reads logs forward or backward from the cursor.
type LogsDirection = api.LogsDirection

const (
	LogsDirectionForward  LogsDirection = "forward"
	LogsDirectionBackward LogsDirection = "backward"
)

// LogsSource selects which store the entries are read from. A build's logs move
// from the temporary store to the persistent one once it finishes.
type LogsSource = api.LogsSource

const (
	LogsSourceTemporary  LogsSource = "temporary"
	LogsSourcePersistent LogsSource = "persistent"
)

// BuildLogsOptions are the optional arguments to BuildLogs.
type BuildLogsOptions struct {
	// CursorMs is the timestamp to read from, in milliseconds.
	CursorMs *int64

	// Limit caps how many entries come back, 1 to 100. Zero lets the platform
	// choose.
	Limit int

	// Direction reads forward or backward from the cursor.
	Direction LogsDirection

	// Level filters entries by minimum severity.
	Level LogLevel

	// Source picks the temporary or persistent log store.
	Source LogsSource
}

// BuildLogs returns a build's logs. The result is never nil.
//
// This reads the log store directly, which is what you want for a build that
// has already finished. To follow a build as it runs, use WaitForBuild, whose
// OnLogs callback receives entries as the build produces them.
//
// GET /templates/{templateID}/builds/{buildID}/logs
func (s *Service) BuildLogs(ctx context.Context, templateID, buildID string, opts BuildLogsOptions) ([]LogEntry, error) {
	params := &api.GetTemplatesTemplateIDBuildsBuildIDLogsParams{}
	if opts.CursorMs != nil {
		params.Cursor = opts.CursorMs
	}
	if opts.Limit > 0 {
		limit := int32(opts.Limit)
		params.Limit = &limit
	}
	if opts.Direction != "" {
		direction := opts.Direction
		params.Direction = &direction
	}
	if opts.Level != "" {
		level := opts.Level
		params.Level = &level
	}
	if opts.Source != "" {
		source := opts.Source
		params.Source = &source
	}

	resp, err := s.t.API().GetTemplatesTemplateIDBuildsBuildIDLogsWithResponse(ctx, templateID, buildID, params)
	if err != nil {
		return nil, err
	}
	logs, err := transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}
	return logEntriesFrom(logs.Logs), nil
}
