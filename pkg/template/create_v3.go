package template

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// CreateV3Options are the optional arguments to CreateV3.
type CreateV3Options struct {
	// Tags to assign to the resulting build.
	Tags []string

	// CPUCount for sandboxes started from the template. Zero uses the team's
	// default.
	CPUCount int

	// MemoryMB for sandboxes started from the template. Zero uses the team's
	// default.
	MemoryMB int

	// MinFreeDiskMB is the free space to leave after the build steps have run.
	// Zero uses the team's default; a pointer to 0 asks for no growth.
	MinFreeDiskMB *int
}

// CreateV3 registers a template and allocates a build for it. It does not run
// the build; see Build, which does the whole sequence.
//
// name may carry a tag, as in "my-template:v1", which counts as if the tag had
// been passed in CreateV3Options.Tags.
//
// POST /v3/templates
func (s *Service) CreateV3(ctx context.Context, name string, opts CreateV3Options) (*BuildInfo, error) {
	body := api.PostV3TemplatesJSONRequestBody{Name: &name}
	if len(opts.Tags) > 0 {
		tags := opts.Tags
		body.Tags = &tags
	}
	if opts.CPUCount > 0 {
		cpu := int32(opts.CPUCount)
		body.CpuCount = &cpu
	}
	if opts.MemoryMB > 0 {
		memory := int32(opts.MemoryMB)
		body.MemoryMB = &memory
	}
	if opts.MinFreeDiskMB != nil {
		minFree := int32(*opts.MinFreeDiskMB)
		body.MinFreeDiskMb = &minFree
	}

	resp, err := s.t.API().PostV3TemplatesWithResponse(ctx, body)
	if err != nil {
		return nil, err
	}
	created, err := transport.Parsed(resp.JSON202, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}

	return &BuildInfo{
		TemplateID: created.TemplateID,
		BuildID:    created.BuildID,
		Name:       name,
		Tags:       created.Tags,
	}, nil
}
