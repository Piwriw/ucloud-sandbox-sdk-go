package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// LogsV2 returns a sandbox's logs. The result is never nil.
//
// GET /v2/sandboxes/{sandboxID}/logs
func (s *Service) LogsV2(ctx context.Context, sandboxID string, opts LogsV2Options) ([]LogEntry, error) {
	params := &api.GetV2SandboxesSandboxIDLogsParams{}
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
		level := api.LogLevel(opts.Level)
		params.Level = &level
	}
	if opts.Search != "" {
		params.Search = &opts.Search
	}

	resp, err := s.t.API().GetV2SandboxesSandboxIDLogsWithResponse(ctx, sandboxID, params)
	if err != nil {
		return nil, err
	}
	logs, err := transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}

	entries := make([]LogEntry, 0, len(logs.Logs))
	for _, entry := range logs.Logs {
		entries = append(entries, LogEntry{
			Timestamp: entry.Timestamp,
			Level:     string(entry.Level),
			Message:   entry.Message,
			Fields:    entry.Fields,
		})
	}
	return entries, nil
}
