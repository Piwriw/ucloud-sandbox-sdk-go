package template

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// StartBuildV2 starts a build that CreateV3 allocated, using the template the
// builder describes.
//
// Any COPY bundles must already be uploaded; Build does that first. Returning
// does not mean the build finished -- poll with BuildStatus, or use WaitForBuild.
//
// POST /v2/templates/{templateID}/builds/{buildID}
func (s *Service) StartBuildV2(ctx context.Context, templateID, buildID string, b *Builder, opts StartBuildV2Options) error {
	if b.err != nil {
		return b.err
	}
	steps := opts.Steps
	if steps == nil {
		var err error
		if steps, err = b.prepareSteps(); err != nil {
			return err
		}
	}

	body, err := b.serialize(steps, opts.SkipCache)
	if err != nil {
		return err
	}

	resp, err := s.t.API().PostV2TemplatesTemplateIDBuildsBuildIDWithResponse(ctx, templateID, buildID, body)
	if err != nil {
		return err
	}
	return transport.Check(resp.HTTPResponse, resp.Body)
}

// StartBuildV2Options are the optional arguments to StartBuildV2.
type StartBuildV2Options struct {
	// SkipCache rebuilds every step, ignoring cached results.
	SkipCache bool

	// Steps overrides the steps taken from the builder. Build passes the steps
	// it already hashed, so they are not hashed twice.
	Steps []Instruction
}
