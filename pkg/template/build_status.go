package template

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// BuildStatusOptions are the optional arguments to BuildStatus.
type BuildStatusOptions struct {
	// LogsOffset skips this many log entries, so a poller only sees what it
	// has not seen before. WaitForBuild maintains it for you.
	LogsOffset int

	// Limit caps how many log entries come back. Zero lets the platform
	// choose.
	Limit int

	// Level filters entries by minimum severity.
	Level LogLevel
}

// BuildStatus reports where a build has got to, along with any logs produced
// since LogsOffset.
//
// GET /templates/{templateID}/builds/{buildID}/status
func (s *Service) BuildStatus(ctx context.Context, templateID, buildID string, opts BuildStatusOptions) (*Status, error) {
	params := &api.GetTemplatesTemplateIDBuildsBuildIDStatusParams{}
	if opts.LogsOffset > 0 {
		offset := int32(opts.LogsOffset)
		params.LogsOffset = &offset
	}
	if opts.Limit > 0 {
		limit := int32(opts.Limit)
		params.Limit = &limit
	}
	if opts.Level != "" {
		level := opts.Level
		params.Level = &level
	}

	resp, err := s.t.API().GetTemplatesTemplateIDBuildsBuildIDStatusWithResponse(ctx, templateID, buildID, params)
	if err != nil {
		return nil, err
	}
	info, err := transport.Parsed(resp.JSON200, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}
	return statusFrom(*info), nil
}
